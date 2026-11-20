package server

import (
	"context"
	"sort"
	"time"

	"github.com/espcaa/sablier/internal/credit"
	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/server/db/sqlc"
	"github.com/espcaa/sablier/internal/server/views"
	"github.com/jackc/pgx/v5/pgtype"
)

// credited time since midnight (server time), per project folder name.
// the same folder on two machines has two ids but one name, so it's summed
func (s *Server) today(ctx context.Context, userID string, now time.Time) (time.Duration, []views.ProjectTime, error) {
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	rows, err := s.q.UserEventsBetween(ctx, sqlc.UserEventsBetweenParams{
		UserID: userID,
		FromTs: pgtype.Timestamptz{Time: midnight, Valid: true},
		ToTs:   pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return 0, nil, err
	}
	events := make([]event.Event, len(rows))
	for i, e := range rows {
		events[i] = event.Event{
			Seq:       e.Seq,
			TS:        e.Ts.Time,
			Kind:      event.Kind(e.Kind),
			SessionID: e.SessionID,
			ProjectID: e.ProjectID,
			Harness:   e.Harness,
			Source:    event.Source(e.Source),
			Data:      e.Data,
		}
	}
	rep := credit.Compute(events, credit.Default, now)

	projects, err := s.q.UserProjects(ctx, userID)
	if err != nil {
		return 0, nil, err
	}
	names := map[string]string{}
	for _, p := range projects {
		names[p.ProjectID] = p.Name
	}
	byName := map[string]time.Duration{}
	for id, d := range rep.ByProject {
		name := names[id]
		if name == "" {
			name = "(unnamed)"
		}
		byName[name] += d
	}

	list := make([]views.ProjectTime, 0, len(byName))
	for name, d := range byName {
		list = append(list, views.ProjectTime{Name: name, Time: d})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Time != list[j].Time {
			return list[i].Time > list[j].Time
		}
		return list[i].Name < list[j].Name
	})
	return rep.Credited, list, nil
}
