//go:build !windows

package launcher

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func runTerminal(c *exec.Cmd, o Options) (int, error) {
	ptmx, err := pty.Start(c)
	if err != nil {
		return 0, err
	}
	defer ptmx.Close()

	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	defer signal.Stop(resize)
	go func() {
		for range resize {
			pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	resize <- syscall.SIGWINCH // initial size

	old, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		c.Process.Kill()
		c.Wait()
		return 0, err
	}
	defer term.Restore(int(os.Stdin.Fd()), old)

	return pipe(c, ptmx, os.Stdin, os.Stdout, o)
}

// copies in -> pty and pty -> out until the harness exits, counting input on the way
func pipe(c *exec.Cmd, ptmx *os.File, in io.Reader, out io.Writer, o Options) (int, error) {
	var keys counter
	stop, ticked := make(chan struct{}), make(chan struct{})
	go ticker(&keys, tickEvery, o.OnTick, stop, ticked)

	go io.Copy(ptmx, countingReader{in, &keys, &focusWatcher{onFocus: o.OnFocus}})

	done := make(chan struct{})
	go func() {
		io.Copy(out, ptmx)
		close(done)
	}()

	code, err := exitCode(c.Wait())

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
	}
	close(stop)
	<-ticked
	return code, err
}
