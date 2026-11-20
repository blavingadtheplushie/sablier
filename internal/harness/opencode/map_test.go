package opencode

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
)

// shaped like a real capture: a prompt with a tool call and an approval, a subagent, then an interrupt
const stream = `
{"type":"session.inbox.enqueued","created":1000,"location":{"directory":"/w/app"},"data":{"sessionID":"s1","inboxID":"in1","item":{"type":"user","payload":{"text":"fix it"}}}}
{"type":"session.execution.started","created":1001,"data":{"sessionID":"s1"}}
{"type":"session.text.delta","created":1500,"data":{"sessionID":"s1","delta":"secret"}}
{"type":"session.tool.input.started","created":2000,"data":{"sessionID":"s1","id":"c1","name":"shell"}}
{"type":"permission.v2.asked","created":2100,"data":{"id":"p1","sessionID":"s1","action":"shell"}}
{"type":"permission.v2.replied","created":5100,"data":{"sessionID":"s1","requestID":"p1","reply":"once"}}
{"type":"session.tool.called","created":5200,"data":{"sessionID":"s1","id":"c1","input":{"command":"rm -rf"}}}
{"type":"session.tool.success","created":7200,"data":{"sessionID":"s1","id":"c1","content":["out"]}}
{"type":"session.created","created":7300,"data":{"sessionID":"kid","parentID":"s1"}}
{"type":"session.inbox.enqueued","created":7400,"data":{"sessionID":"kid","item":{"type":"user","payload":{"text":"sub"}}}}
{"type":"session.execution.started","created":7401,"data":{"sessionID":"kid"}}
{"type":"permission.v2.asked","created":7500,"data":{"id":"p2","sessionID":"kid","action":"edit"}}
{"type":"permission.v2.replied","created":8500,"data":{"sessionID":"kid","requestID":"p2","reply":"reject"}}
{"type":"session.execution.succeeded","created":9000,"data":{"sessionID":"s1"}}
{"type":"session.inbox.enqueued","created":10000,"data":{"sessionID":"s1","inboxID":"in2","item":{"type":"user","payload":{"text":"go"}}}}
{"type":"session.execution.started","created":10001,"data":{"sessionID":"s1"}}
{"type":"session.execution.interrupted","created":12000,"data":{"sessionID":"s1","reason":"user"}}
`

func TestMap(t *testing.T) {
	m := &Mapper{Salt: []byte("salt"), Session: "run:x", Default: "start", Project: func(dir string) string { return "github.com/a/" + dir[3:] }}
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(stream), "\n") {
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		for _, ev := range m.Map(e) {
			got = append(got, describe(t, ev))
		}
	}
	for _, ev := range m.Close(time.UnixMilli(20000)) {
		got = append(got, describe(t, ev))
	}

	want := []string{
		"1000 session_start github.com/a/app",
		"1000 prompt_sent github.com/a/app",
		"1001 agent_turn_start in1",
		"5100 approval allow=true 3000ms shell",
		"7200 tool_call shell 2000ms",
		"8500 approval allow=false 1000ms edit", // a subagent's, on the parent's session
		"9000 agent_turn_end in1 done",
		"10000 prompt_sent github.com/a/app",
		"10001 agent_turn_start in2",
		"12000 interrupt",
		"12000 agent_turn_end in2 interrupted",
		"20000 session_end",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func describe(t *testing.T, e event.Event) string {
	if e.SessionID != "run:x" || e.Harness != Harness || e.Source != event.SourceAdapter {
		t.Fatalf("envelope %+v", e)
	}
	p, err := e.Payload()
	if err != nil {
		t.Fatal(err)
	}
	s := fmt.Sprintf("%d %s", e.TS.UnixMilli(), e.Kind)
	switch d := p.(type) {
	case *event.SessionStartData:
		s += " " + e.ProjectID
	case *event.PromptSentData:
		if d.Hash == "" || strings.Contains(string(e.Data), "fix it") {
			t.Fatalf("prompt not hashed: %s", e.Data)
		}
		s += " " + e.ProjectID
	case *event.AgentTurnStartData:
		s += " " + d.TurnID
	case *event.AgentTurnEndData:
		s += " " + d.TurnID + " " + d.Reason
	case *event.ApprovalData:
		s += fmt.Sprintf(" allow=%v %dms %s", d.Allow, d.LatencyMs, d.Tool)
	case *event.ToolCallData:
		s += fmt.Sprintf(" %s %dms", d.Tool, d.DurationMs)
	}
	return s
}
