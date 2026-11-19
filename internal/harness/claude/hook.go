package claude

// tool_input, tool_response and last_assistant_message are never decoded so we never touch prompt data
type Hook struct {
	Event      string `json:"hook_event_name"`
	SessionID  string `json:"session_id"`
	PromptID   string `json:"prompt_id"`
	Cwd        string `json:"cwd"`
	Model      string `json:"model"`
	Source     string `json:"source"` // startup, resume, clear, compact
	Prompt     string `json:"prompt"` // hashed, never stored
	ToolName   string `json:"tool_name"`
	ToolUseID  string `json:"tool_use_id"`
	DurationMs int64  `json:"duration_ms"`
	AgentID    string `json:"agent_id"` // set for subagent hooks
}

const Harness = "claude-code"
