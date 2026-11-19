package claude

import (
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/prompt"
)

type State struct {
	OpenTurn    string    `json:"open_turn,omitempty"` // prompt_id of the running turn
	LastEventAt time.Time `json:"last_event_at"`
}

type Input struct {
	Hook      Hook
	Now       time.Time
	Salt      []byte
	ProjectID string
}

// pure, no io
func Map(st State, in Input) (State, []event.Event) {
	h := in.Hook
	m := mapper{in: in}

	switch h.Event {
	case "SessionStart":
		if h.Source != "compact" { // same session continues
			m.emit(event.SessionStart, in.Now, event.SessionStartData{Model: h.Model})
		}

	case "SessionEnd":
		m.emit(event.SessionEnd, in.Now, event.SessionEndData{Reason: "exit"})

	case "UserPromptSubmit":
		m.emit(event.PromptSent, in.Now, event.PromptSentData{
			Hash:         prompt.Hash(in.Salt, h.Prompt),
			ApprovalLike: prompt.ApprovalLike(h.Prompt),
		})
		m.emit(event.AgentTurnStart, in.Now, event.AgentTurnStartData{TurnID: h.PromptID})
		st.OpenTurn = h.PromptID

	case "Stop", "StopFailure":
		reason := "done"
		if h.Event == "StopFailure" {
			reason = "error"
		}
		m.emit(event.AgentTurnEnd, in.Now, event.AgentTurnEndData{TurnID: h.PromptID, Reason: reason})
		st.OpenTurn = ""

	case "PostToolUse":
		// ts is the end, duration gives the start
		m.emit(event.ToolCall, in.Now, event.ToolCallData{
			Tool:       category(h.ToolName),
			DurationMs: h.DurationMs,
		})
	}

	st.LastEventAt = in.Now
	return st, m.out
}

type mapper struct {
	in  Input
	out []event.Event
}

func (m *mapper) emit(kind event.Kind, ts time.Time, data any) {
	e, err := event.New(kind, ts, data)
	if err != nil {
		panic(err) // unknown kind, a bug here
	}
	e.SessionID = m.in.Hook.SessionID
	e.ProjectID = m.in.ProjectID
	e.Harness = Harness
	e.Source = event.SourceAdapter
	m.out = append(m.out, e)
}

func category(tool string) event.ToolCategory {
	switch tool {
	case "Bash", "BashOutput", "KillShell":
		return event.ToolShell
	case "Edit", "MultiEdit", "Write", "NotebookEdit":
		return event.ToolEdit
	case "Read", "Glob", "Grep", "LS":
		return event.ToolRead
	case "WebFetch", "WebSearch":
		return event.ToolWeb
	case "Agent", "Task":
		return event.ToolAgent
	}
	return event.ToolOther
}
