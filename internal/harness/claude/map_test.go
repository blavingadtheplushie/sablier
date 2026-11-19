package claude

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
)

var t0 = time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)

// one hook, `at` seconds after t0
type step struct {
	at   float64
	hook Hook
}

func run(t *testing.T, steps ...step) []event.Event {
	t.Helper()
	var st State
	var all []event.Event
	for _, s := range steps {
		if s.hook.SessionID == "" {
			s.hook.SessionID = "s1"
		}
		var out []event.Event
		st, out = Map(st, Input{
			Hook:      s.hook,
			Now:       t0.Add(time.Duration(s.at * float64(time.Second))),
			Salt:      []byte("test-salt"),
			ProjectID: "p1",
		})
		all = append(all, out...)
	}
	return all
}

func describe(t *testing.T, events []event.Event) string {
	t.Helper()
	var b strings.Builder
	for _, e := range events {
		p, err := e.Payload()
		if err != nil {
			t.Fatal(err)
		}
		// hashes are tested in the prompt package
		if ps, ok := p.(*event.PromptSentData); ok && len(ps.Hash) == 32 {
			ps.Hash = "<hash>"
		}
		fmt.Fprintf(&b, "%g %s %+v\n", e.TS.Sub(t0).Seconds(), e.Kind, p)
	}
	return b.String()
}

func expect(t *testing.T, got []event.Event, want string) {
	t.Helper()
	g := strings.TrimSpace(describe(t, got))
	w := strings.TrimSpace(strings.ReplaceAll(want, "\t", ""))
	if g != w {
		t.Errorf("events differ\n--- got\n%s\n--- want\n%s", g, w)
	}
}

func TestSimpleTurn(t *testing.T) {
	got := run(t,
		step{0, Hook{Event: "SessionStart", Source: "startup", Model: "opus"}},
		step{1, Hook{Event: "UserPromptSubmit", PromptID: "p1", Prompt: "fix the test"}},
		step{3, Hook{Event: "PreToolUse", PromptID: "p1", ToolName: "Bash", ToolUseID: "t1"}},
		step{4, Hook{Event: "PostToolUse", PromptID: "p1", ToolName: "Bash", ToolUseID: "t1", DurationMs: 500}},
		step{6, Hook{Event: "Stop", PromptID: "p1"}},
		step{9, Hook{Event: "SessionEnd"}},
	)
	expect(t, got, `
		0 session_start &{HarnessVersion: Model:opus}
		1 prompt_sent &{Pasted:false ComposeMs:0 Queued:false ApprovalLike:false Hash:<hash>}
		1 agent_turn_start &{TurnID:p1}
		4 tool_call &{Tool:shell DurationMs:500}
		6 agent_turn_end &{TurnID:p1 Reason:done}
		9 session_end &{Reason:exit}
	`)
}

func TestEnvelope(t *testing.T) {
	got := run(t, step{0, Hook{Event: "UserPromptSubmit", SessionID: "abc", PromptID: "p1"}})
	for _, e := range got {
		if e.SessionID != "abc" || e.ProjectID != "p1" || e.Harness != Harness || e.Source != event.SourceAdapter {
			t.Errorf("bad envelope: %+v", e)
		}
	}
}

func TestStopFailureAndCompact(t *testing.T) {
	got := run(t,
		step{0, Hook{Event: "SessionStart", Source: "compact"}},
		step{1, Hook{Event: "UserPromptSubmit", PromptID: "p1", Prompt: "yes"}},
		step{2, Hook{Event: "StopFailure", PromptID: "p1"}},
	)
	expect(t, got, `
		1 prompt_sent &{Pasted:false ComposeMs:0 Queued:false ApprovalLike:true Hash:<hash>}
		1 agent_turn_start &{TurnID:p1}
		2 agent_turn_end &{TurnID:p1 Reason:error}
	`)
}

func TestPromptTextNeverStored(t *testing.T) {
	secret := "my secret api key is hunter2"
	for _, e := range run(t, step{0, Hook{Event: "UserPromptSubmit", PromptID: "p1", Prompt: secret}}) {
		if strings.Contains(string(e.Data), "hunter2") {
			t.Fatalf("prompt text leaked into %s: %s", e.Kind, e.Data)
		}
	}
}

func TestCategory(t *testing.T) {
	cases := map[string]event.ToolCategory{
		"Bash": event.ToolShell, "Edit": event.ToolEdit, "Write": event.ToolEdit,
		"Read": event.ToolRead, "Grep": event.ToolRead, "WebFetch": event.ToolWeb,
		"Agent": event.ToolAgent, "mcp__figma__x": event.ToolOther, "NewTool": event.ToolOther,
	}
	for tool, want := range cases {
		if got := category(tool); got != want {
			t.Errorf("category(%q) = %s, want %s", tool, got, want)
		}
	}
}
