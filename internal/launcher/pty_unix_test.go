//go:build !windows

package launcher

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func runPipe(t *testing.T, script, input string) (int, string) {
	code, out, _ := runPipeTicks(t, script, input)
	return code, out
}

func runPipeTicks(t *testing.T, script, input string) (int, string, []Tick) {
	t.Helper()
	c := exec.Command("sh", "-c", script)
	ptmx, err := pty.Start(c)
	if err != nil {
		t.Fatal(err)
	}
	defer ptmx.Close()
	var out bytes.Buffer
	var ticks []Tick
	code, err := pipe(c, ptmx, strings.NewReader(input), &out, Options{
		OnTick:  func(tk Tick) { ticks = append(ticks, tk) },
		OnFocus: func(bool, time.Time) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	return code, out.String(), ticks
}

func TestHarnessSeesATerminal(t *testing.T) {
	_, out := runPipe(t, `test -t 0 && test -t 1 && echo is-a-tty`, "")
	if !strings.Contains(out, "is-a-tty") {
		t.Fatalf("got %q", out)
	}
}

func TestInputReachesHarness(t *testing.T) {
	_, out := runPipe(t, `read x; echo "got:$x"`, "hello\n")
	if !strings.Contains(out, "got:hello") {
		t.Fatalf("got %q", out)
	}
}

func TestPtyExitCode(t *testing.T) {
	if code, _ := runPipe(t, `exit 4`, ""); code != 4 {
		t.Fatalf("got %d, want 4", code)
	}
}

func TestTypingIsTicked(t *testing.T) {
	_, _, ticks := runPipeTicks(t, `read x`, "hello\n")
	if len(ticks) != 1 || ticks[0].Keys != 1 || ticks[0].Pasted {
		t.Fatalf("got %+v, want one tick with the input chunk", ticks)
	}
}
