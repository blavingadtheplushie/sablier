// credited time from raw events, bump Version when a rule changes
package credit

import (
	"sort"
	"time"

	"github.com/espcaa/sablier/internal/event"
)

const Version = 1

type Rules struct {
	Cap time.Duration // n, max window before and after each action
}

var Default = Rules{Cap: 5 * time.Minute}

type Report struct {
	Credited  time.Duration
	ByProject map[string]time.Duration
	AgentBusy time.Duration
	Actions   int
	First     time.Time // wall clock span
	Last      time.Time
}

type span struct{ start, end time.Time }

// a credited window, anchor is action that started it
type window struct {
	span
	project string
	anchor  time.Time
}

// rules, per session, n = r.Cap:
//  1. each action earns up to n after it
//  2. and up to n before it, only once the agent was don
//  3. overlapping time counts once
//
// nothing is counted after until.
func Compute(events []event.Event, r Rules, until time.Time) Report {
	rep := Report{ByProject: map[string]time.Duration{}}
	if len(events) == 0 {
		return rep
	}

	sorted := append([]event.Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TS.Before(sorted[j].TS) })
	rep.First, rep.Last = sorted[0].TS, sorted[len(sorted)-1].TS

	sessions := map[string][]event.Event{}
	var order []string
	for _, e := range sorted {
		if _, ok := sessions[e.SessionID]; !ok {
			order = append(order, e.SessionID)
		}
		sessions[e.SessionID] = append(sessions[e.SessionID], e)
	}

	var windows []window
	var busy []span
	for _, id := range order {
		w, b, n := session(sessions[id], r.Cap)
		windows = append(windows, w...)
		busy = append(busy, b...)
		rep.Actions += n
	}

	for i := range windows {
		windows[i].span = clip(windows[i].span, until)
	}
	for i := range busy {
		busy[i] = clip(busy[i], until)
	}

	rep.Credited = attribute(windows, rep.ByProject)
	rep.AgentBusy = union(busy)
	return rep
}

func session(events []event.Event, n time.Duration) ([]window, []span, int) {
	var (
		start    time.Time // session_start, if seen
		turns    []span
		open     = map[string]time.Time{}
		actions  []event.Event
		latency  = map[int]time.Duration{} // approval index in actions -> dialog open time
		lastSeen = events[len(events)-1].TS
	)

	for _, e := range events {
		switch e.Kind {
		case event.SessionStart:
			start = e.TS
		case event.AgentTurnStart:
			if p, err := e.Payload(); err == nil {
				open[p.(*event.AgentTurnStartData).TurnID] = e.TS
			}
		case event.AgentTurnEnd:
			if p, err := e.Payload(); err == nil {
				id := p.(*event.AgentTurnEndData).TurnID
				if s, ok := open[id]; ok {
					turns = append(turns, span{s, e.TS})
					delete(open, id)
				}
			}
		case event.PromptSent, event.Interrupt:
			actions = append(actions, e)
		case event.Approval:
			if p, err := e.Payload(); err == nil {
				latency[len(actions)] = time.Duration(p.(*event.ApprovalData).LatencyMs) * time.Millisecond
			}
			actions = append(actions, e)
		}
	}
	for _, s := range open {
		turns = append(turns, span{s, lastSeen}) // never ended, assume busy until the last event
	}

	var out []window
	for i, a := range actions {
		h := a.TS
		out = append(out, window{span{h, h.Add(n)}, a.ProjectID, h})

		var from time.Time
		if l, ok := latency[i]; ok {
			from = h.Add(-l) // reading the dialog
		} else {
			if busyAt(turns, h) {
				continue
			}
			from = latest(start, lastEnd(turns, h))
			if i > 0 {
				from = latest(from, actions[i-1].TS)
			}
			if from.IsZero() {
				continue // nothing to bound it, don't guess
			}
		}
		from = latest(from, h.Add(-n))
		out = append(out, window{span{from, h}, a.ProjectID, h})
	}
	return out, turns, len(actions)
}

func busyAt(turns []span, t time.Time) bool {
	for _, s := range turns {
		if s.start.Before(t) && t.Before(s.end) {
			return true
		}
	}
	return false
}

// latest turn end at or before t
func lastEnd(turns []span, t time.Time) time.Time {
	var last time.Time
	for _, s := range turns {
		if !s.end.After(t) && s.end.After(last) {
			last = s.end
		}
	}
	return last
}

func latest(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func clip(s span, until time.Time) span {
	if s.end.After(until) {
		s.end = until
	}
	if s.start.After(s.end) {
		s.start = s.end
	}
	return s
}

// splits time at every window edge, gives each piece to the window with the latest anchor
func attribute(ws []window, by map[string]time.Duration) time.Duration {
	var points []time.Time
	for _, w := range ws {
		points = append(points, w.start, w.end)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Before(points[j]) })

	var total time.Duration
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		if !a.Before(b) {
			continue
		}
		best := -1
		for j, w := range ws {
			if !w.start.After(a) && !w.end.Before(b) && (best < 0 || w.anchor.After(ws[best].anchor)) {
				best = j
			}
		}
		if best >= 0 {
			by[ws[best].project] += b.Sub(a)
			total += b.Sub(a)
		}
	}
	return total
}

func union(spans []span) time.Duration {
	ws := make([]window, len(spans))
	for i, s := range spans {
		ws[i] = window{span: s}
	}
	return attribute(ws, map[string]time.Duration{})
}
