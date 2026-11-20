package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type settings struct {
	Theme string
	Hooks map[string][]struct {
		Hooks []struct{ Type, Command string }
	}
}

func parse(t *testing.T, args []string) (settings, []string) {
	t.Helper()
	if len(args) < 2 || args[0] != "--settings" {
		t.Fatalf("args %q", args)
	}
	var s settings
	if err := json.Unmarshal([]byte(args[1]), &s); err != nil {
		t.Fatal(err)
	}
	return s, args[2:]
}

func TestWithHooks(t *testing.T) {
	args, err := WithHooks("/Users/a b/it's/sablier", []string{"-p", "hi"})
	if err != nil {
		t.Fatal(err)
	}
	s, rest := parse(t, args)
	if len(s.Hooks) != len(hookEvents) || strings.Join(rest, " ") != "-p hi" {
		t.Fatalf("got %+v, rest %q", s, rest)
	}
	if got := s.Hooks["UserPromptSubmit"][0].Hooks[0]; got.Command != `'/Users/a b/it'\''s/sablier' hook claude` {
		t.Fatalf("got %+v", got)
	}
}

// claude keeps only the last --settings, so the user's must be merged, not stacked
func TestWithHooksMergesUserSettings(t *testing.T) {
	own := `{"theme":"dark","hooks":{"Stop":[{"hooks":[{"type":"command","command":"notify-send done"}]}]}}`
	file := filepath.Join(t.TempDir(), "s.json")
	os.WriteFile(file, []byte(own), 0o644)

	for name, in := range map[string][]string{
		"json":   {"--settings", own, "-p", "hi"},
		"equals": {"--settings=" + own, "-p", "hi"},
		"file":   {"-p", "hi", "--settings", file},
	} {
		args, err := WithHooks("/bin/sablier", in)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s, rest := parse(t, args)
		if s.Theme != "dark" || len(s.Hooks["Stop"]) != 2 || s.Hooks["Stop"][0].Hooks[0].Command != "notify-send done" {
			t.Errorf("%s: user settings lost: %+v", name, s)
		}
		if strings.Join(rest, " ") != "-p hi" || strings.Contains(strings.Join(rest, " "), "settings") {
			t.Errorf("%s: rest %q", name, rest)
		}
	}
}

func TestWithHooksBadUserSettings(t *testing.T) {
	if _, err := WithHooks("/bin/sablier", []string{"--settings", "/no/such/file.json"}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := WithHooks("/bin/sablier", []string{"--settings", "{not json"}); err == nil {
		t.Fatal("expected error")
	}
}
