package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTodayEndToEnd(t *testing.T) {
	t.Setenv("SABLIER_HOME", t.TempDir())
	t.Setenv("SABLIER_RUN_ID", "run1")
	runHookCmd(t, `{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/tmp"}`)
	runHookCmd(t, `{"hook_event_name":"UserPromptSubmit","session_id":"s1","prompt_id":"p1","cwd":"/tmp","prompt":"hi"}`)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"today"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 actions") || !strings.Contains(out.String(), "tmp") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
}
