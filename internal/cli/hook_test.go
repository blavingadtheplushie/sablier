package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runHook(t *testing.T, stdin string) (stdout, stderr string) {
	t.Helper()
	root := NewRootCmd()
	var out, errOut bytes.Buffer
	root.SetIn(strings.NewReader(stdin))
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"hook"})
	if err := root.Execute(); err != nil {
		t.Fatalf("hook must never fail, got %v", err)
	}
	return out.String(), errOut.String()
}

func TestHookRecordsPayload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SABLIER_HOME", home)

	stdout, _ := runHook(t, `{"hook_event_name":"Stop","session_id":"s1"}`)
	if stdout != "" {
		t.Fatalf("hook printed to stdout (goes into claude's context): %q", stdout)
	}

	b, err := os.ReadFile(filepath.Join(home, "hooks-debug.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"hook_event_name":"Stop"`) {
		t.Fatalf("payload not recorded: %s", b)
	}
}

func TestHookBadInputStaysQuiet(t *testing.T) {
	t.Setenv("SABLIER_HOME", t.TempDir())

	stdout, stderr := runHook(t, "not json")
	if stdout != "" {
		t.Fatalf("stdout should be empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "not json") {
		t.Fatalf("expected error on stderr, got %q", stderr)
	}
}
