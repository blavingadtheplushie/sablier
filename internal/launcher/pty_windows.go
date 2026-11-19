//go:build windows

package launcher

import "os/exec"

// todo: conpty
func runTerminal(c *exec.Cmd) (int, error) { return runDirect(c) }
