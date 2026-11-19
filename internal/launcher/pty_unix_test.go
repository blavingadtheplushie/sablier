//go:build !windows

package launcher

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func runPipe(t *testing.T, script, input string) (int, string) {
	t.Helper()
	c := exec.Command("sh", "-c", script)
	ptmx, err := pty.Start(c)
	if err != nil {
		t.Fatal(err)
	}
	defer ptmx.Close()
	var out bytes.Buffer
	code, err := pipe(c, ptmx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatal(err)
	}
	return code, out.String()
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
