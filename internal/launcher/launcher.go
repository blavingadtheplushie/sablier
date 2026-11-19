package launcher

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"golang.org/x/term"
)

type Options struct {
	RunID   string
	OnTick  func(Tick) // input activity
	OnFocus func(in bool, at time.Time)
}

const tickEvery = 10 * time.Second

// runs a harness attached to this terminal with SABLIER_RUN_ID set, returns its exit code
func Run(name string, args []string, o Options) (int, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return 0, fmt.Errorf("%s not found in PATH", name)
	}
	if o.OnTick == nil {
		o.OnTick = func(Tick) {}
	}
	if o.OnFocus == nil {
		o.OnFocus = func(bool, time.Time) {}
	}

	c := exec.Command(path, args...)
	c.Env = append(os.Environ(), "SABLIER_RUN_ID="+o.RunID)

	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return runTerminal(c, o)
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

func NewRunID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
