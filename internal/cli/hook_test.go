package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/store"
)

func runHookCmd(t *testing.T, stdin string) (stdout, stderr string) {
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

func TestHookStoresEvents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SABLIER_HOME", home)
	t.Setenv("SABLIER_RUN_ID", "run1")

	for _, payload := range []string{
		`{"hook_event_name":"UserPromptSubmit","session_id":"s1","prompt_id":"p1","cwd":"/tmp","prompt":"secret words"}`,
		`{"hook_event_name":"Stop","session_id":"s1","prompt_id":"p1","cwd":"/tmp"}`,
	} {
		stdout, stderr := runHookCmd(t, payload)
		if stdout != "" {
			t.Fatalf("hook printed to stdout (goes into claude's context): %q", stdout)
		}
		if stderr != "" {
			t.Fatalf("unexpected error: %s", stderr)
		}
	}

	s, err := store.Open(filepath.Join(home, "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events, err := s.Between(context.Background(), time.Time{}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	var kinds []string
	for _, e := range events {
		kinds = append(kinds, string(e.Kind))
		if e.SessionID != "run:run1" {
			t.Fatalf("event not grouped under the launch: %+v", e)
		}
		if strings.Contains(string(e.Data), "secret") {
			t.Fatalf("prompt text stored: %s", e.Data)
		}
	}
	want := []string{string(event.PromptSent), string(event.AgentTurnStart), string(event.AgentTurnEnd)}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}

	if _, err := os.Stat(filepath.Join(home, "hooks-debug.jsonl")); !os.IsNotExist(err) {
		t.Fatal("debug log written without SABLIER_DEBUG=1")
	}
}

func TestHookDebugLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SABLIER_HOME", home)
	t.Setenv("SABLIER_RUN_ID", "run1")
	t.Setenv("SABLIER_DEBUG", "1")

	runHookCmd(t, `{"hook_event_name":"Stop","session_id":"s1"}`)
	b, err := os.ReadFile(filepath.Join(home, "hooks-debug.jsonl"))
	if err != nil || !strings.Contains(string(b), `"Stop"`) {
		t.Fatalf("debug log missing: %s %v", b, err)
	}
}

func TestHookBadInputStaysQuiet(t *testing.T) {
	t.Setenv("SABLIER_HOME", t.TempDir())
	t.Setenv("SABLIER_RUN_ID", "run1")

	stdout, stderr := runHookCmd(t, "not json")
	if stdout != "" {
		t.Fatalf("stdout should be empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "not json") {
		t.Fatalf("expected error on stderr, got %q", stderr)
	}
}

func TestHookIgnoresPlainClaude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SABLIER_HOME", home)
	t.Setenv("SABLIER_RUN_ID", "")

	stdout, stderr := runHookCmd(t, `{"hook_event_name":"UserPromptSubmit","session_id":"s1","prompt_id":"p1","prompt":"hi"}`)
	if stdout != "" || stderr != "" {
		t.Fatalf("expected silence, got %q %q", stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(home, "sablier.db")); !os.IsNotExist(err) {
		t.Fatal("db touched for a session not launched through sablier")
	}
}
