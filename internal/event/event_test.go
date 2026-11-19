package event

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	ts := time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)

	cases := map[Kind]any{
		SessionStart:   &SessionStartData{HarnessVersion: "2.1.0", Model: "x"},
		SessionEnd:     &SessionEndData{Reason: "exit"},
		InputTick:      &InputTickData{WindowMs: 10_000, Keys: 42},
		Focus:          &FocusData{In: true},
		PromptSent:     &PromptSentData{ComposeMs: 9000, Queued: true, Hash: "abc"},
		Approval:       &ApprovalData{Allow: true, LatencyMs: 1200, Tool: ToolShell},
		Interrupt:      &InterruptData{},
		AgentTurnStart: &AgentTurnStartData{TurnID: "t1"},
		AgentTurnEnd:   &AgentTurnEndData{TurnID: "t1", Reason: "done"},
		ToolCall:       &ToolCallData{Tool: ToolEdit, DurationMs: 300},
		GitCommit:      &GitCommitData{SHA: "deadbeef", Time: ts},
	}

	if len(cases) != len(payloads) {
		t.Fatalf("test covers %d kinds, %d exist", len(cases), len(payloads))
	}

	for kind, want := range cases {
		e, err := New(kind, ts, want)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}

		// through json, like it goes over the wire
		b, _ := json.Marshal(e)
		var back Event
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}

		got, err := back.Payload()
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %+v, want %+v", kind, got, want)
		}
	}
}

func TestUnknownKind(t *testing.T) {
	if _, err := New("nope", time.Now(), nil); err == nil {
		t.Fatal("expected error")
	}
}
