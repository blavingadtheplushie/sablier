//go:build !windows

package launcher

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

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

	wait := func() (int, error) { return exitCode(c.Wait()) }
	return relay(ptmx, wait, os.Stdin, os.Stdout, o)
}
