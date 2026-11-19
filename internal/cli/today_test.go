package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/credit"
)

func TestDur(t *testing.T) {
	cases := map[time.Duration]string{
		0: "0m", 29 * time.Second: "0m", 27*time.Minute + 51*time.Second: "28m",
		time.Hour: "1h00m", 65 * time.Minute: "1h05m",
	}
	for d, want := range cases {
		if got := dur(d); got != want {
			t.Errorf("dur(%s) = %s, want %s", d, got, want)
		}
	}
}

func TestPrintReport(t *testing.T) {
	t0 := time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)
	r := credit.Report{
		Credited:  30 * time.Minute,
		ByProject: map[string]time.Duration{"a": 20 * time.Minute, "b": 10 * time.Minute},
		AgentBusy: 7 * time.Minute,
		Actions:   16,
		First:     t0,
		Last:      t0.Add(3 * time.Hour),
	}
	var out bytes.Buffer
	printReport(&out, r, map[string]string{"a": "/work/sablier"})

	want := `30m credited
7m agent busy · 16 actions · 3h00m span

  20m      /work/sablier
  10m      (unknown b)
`
	if out.String() != want {
		t.Errorf("got\n%s\nwant\n%s", out.String(), want)
	}
}

func TestTodayEndToEnd(t *testing.T) {
	t.Setenv("SABLIER_HOME", t.TempDir())
	runHookCmd(t, `{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/tmp"}`)
	runHookCmd(t, `{"hook_event_name":"UserPromptSubmit","session_id":"s1","prompt_id":"p1","cwd":"/tmp","prompt":"hi"}`)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"today"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 actions") || !strings.Contains(out.String(), "/tmp") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
}
