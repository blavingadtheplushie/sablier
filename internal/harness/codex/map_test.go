package codex

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
)

func TestMap(t *testing.T) {
	steps := []struct {
		at float64
		h  Hook
	}{
		{0, Hook{Event: "SessionStart", Source: "startup"}},
		{0, Hook{Event: "UserPromptSubmit", TurnID: "t1", Prompt: "run ls"}},
		{4.1, Hook{Event: "PreToolUse", TurnID: "t1", ToolName: "Bash", ToolUseID: "a"}},
		{4.3, Hook{Event: "PostToolUse", TurnID: "t1", ToolName: "Bash", ToolUseID: "a"}},
		{16.4, Hook{Event: "Stop", TurnID: "t1"}},
		{40.7, Hook{Event: "UserPromptSubmit", TurnID: "t2", Prompt: "curl it"}},
		{43.7, Hook{Event: "PreToolUse", TurnID: "t2", ToolName: "Bash", ToolUseID: "b"}},
		{43.8, Hook{Event: "PermissionRequest", TurnID: "t2", ToolName: "Bash"}},
		{46.7, Hook{Event: "PostToolUse", TurnID: "t2", ToolName: "Bash", ToolUseID: "b"}},
		{51.7, Hook{Event: "Stop", TurnID: "t2"}},
		{53.9, Hook{Event: "UserPromptSubmit", TurnID: "t3", Prompt: "write a poem"}},
		{61.3, Hook{Event: "Interrupt", TurnID: "t3"}},
		{67.7, Hook{Event: "UserPromptSubmit", TurnID: "t4", Prompt: "rm it"}},
		{72.1, Hook{Event: "PreToolUse", TurnID: "t4", ToolName: "Bash", ToolUseID: "c"}},
		{72.1, Hook{Event: "PermissionRequest", TurnID: "t4", ToolName: "Bash"}},
		{75.4, Hook{Event: "Interrupt", TurnID: "t4"}},
	}

	t0 := time.Date(2026, 11, 20, 21, 0, 0, 0, time.UTC)
	var st State
	var got []string
	for _, s := range steps {
		s.h.SessionID = "c1"
		now := t0.Add(time.Duration(s.at * float64(time.Second)))
		var out []event.Event
		st, out = Map(st, Input{Hook: s.h, Now: now, Salt: []byte("salt"), ProjectID: "p1", Session: "run:x"})
		for _, e := range out {
			got = append(got, describe(t, t0, e))
		}
	}

	want := []string{
		"0.0 session_start",
		"0.0 prompt_sent",
		"0.0 agent_turn_start t1",
		"4.3 tool_call shell 200ms",
		"16.4 agent_turn_end t1 done",
		"40.7 prompt_sent",
		"40.7 agent_turn_start t2",
		"46.7 approval allow=true shell",
		"46.7 tool_call shell 3000ms",
		"51.7 agent_turn_end t2 done",
		"53.9 prompt_sent",
		"53.9 agent_turn_start t3",
		"61.3 interrupt",
		"61.3 agent_turn_end t3 interrupted",
		"67.7 prompt_sent",
		"67.7 agent_turn_start t4",
		"75.4 approval allow=false shell",
		"75.4 interrupt",
		"75.4 agent_turn_end t4 interrupted",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if len(st.Pending) != 0 || len(st.Tools) != 0 || st.OpenTurn != "" {
		t.Fatalf("leftover state %+v", st)
	}
}

func describe(t *testing.T, t0 time.Time, e event.Event) string {
	if e.SessionID != "run:x" || e.ProjectID != "p1" || e.Harness != Harness || e.Source != event.SourceAdapter {
		t.Fatalf("envelope %+v", e)
	}
	p, err := e.Payload()
	if err != nil {
		t.Fatal(err)
	}
	s := fmt.Sprintf("%.1f %s", e.TS.Sub(t0).Seconds(), e.Kind)
	switch d := p.(type) {
	case *event.PromptSentData:
		if d.Hash == "" || strings.Contains(string(e.Data), "run ls") {
			t.Fatalf("prompt not hashed: %s", e.Data)
		}
	case *event.AgentTurnStartData:
		s += " " + d.TurnID
	case *event.AgentTurnEndData:
		s += " " + d.TurnID + " " + d.Reason
	case *event.ApprovalData:
		s += fmt.Sprintf(" allow=%v %s", d.Allow, d.Tool)
	case *event.ToolCallData:
		s += fmt.Sprintf(" %s %dms", d.Tool, d.DurationMs)
	}
	return s
}
