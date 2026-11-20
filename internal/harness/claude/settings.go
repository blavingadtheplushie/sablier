package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var hookEvents = []string{
	"SessionStart", "SessionEnd", "UserPromptSubmit", "Stop", "StopFailure",
	"PreToolUse", "PermissionRequest", "PostToolUse",
}

// args to launch claude code with hooks
func WithHooks(exe string, args []string) ([]string, error) {
	settings := map[string]any{}
	var rest []string
	for i := 0; i < len(args); i++ {
		v, ok := strings.CutPrefix(args[i], "--settings=")
		if !ok && args[i] == "--settings" && i+1 < len(args) {
			v, ok = args[i+1], true
			i++
		}
		if !ok {
			rest = append(rest, args[i])
			continue
		}
		var err error
		if settings, err = readSettings(v); err != nil {
			return nil, fmt.Errorf("can't add sablier's hooks to your --settings: %w", err)
		}
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	command := quote(exe) + " hook"
	for _, ev := range hookEvents {
		groups, _ := hooks[ev].([]any)
		hooks[ev] = append(groups, map[string]any{
			"hooks": []any{map[string]any{"type": "command", "command": command}},
		})
	}
	settings["hooks"] = hooks

	b, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	return append([]string{"--settings", string(b)}, rest...), nil
}

func readSettings(v string) (map[string]any, error) {
	b := []byte(v)
	if !strings.HasPrefix(strings.TrimSpace(v), "{") {
		var err error
		if b, err = os.ReadFile(v); err != nil {
			return nil, err
		}
	}
	m := map[string]any{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func quote(path string) string {
	if runtime.GOOS == "windows" {
		return `"` + filepath.ToSlash(path) + `"`
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
