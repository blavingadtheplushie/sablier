package launcher

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"golang.org/x/term"
)

// runs a harness attached to this terminal with SABLIER_RUN_ID set, returns its exit code
func Run(name string, args []string) (int, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return 0, fmt.Errorf("%s not found in PATH", name)
	}

	id, err := newRunID()
	if err != nil {
		return 0, err
	}

	c := exec.Command(path, args...)
	c.Env = append(os.Environ(), "SABLIER_RUN_ID="+id)

	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return runTerminal(c)
	}
	return runDirect(c)
}

func runDirect(c *exec.Cmd) (int, error) {
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	return exitCode(c.Run())
}

func exitCode(err error) (int, error) {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}

func newRunID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
