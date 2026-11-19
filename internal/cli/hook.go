package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/espcaa/sablier/internal/paths"
	"github.com/spf13/cobra"
)

// claude code calls this on every hook event with a json payload on stdin.
// rules: never print to stdout (it goes into claude's context), never exit non-zero
// (exit 2 blocks the prompt or tool call).
func newHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook",
		Short:  "receive a claude code hook event (called by claude, not by you)",
		Hidden: true,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := recordHook(cmd.InOrStdin(), time.Now()); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "sablier hook:", err)
			}
		},
	}
}

// for now, append the raw payload to hooks-debug.jsonl so we can see what claude sends.
// it contains prompt text, so it stays local and is temporary.
func recordHook(r io.Reader, now time.Time) error {
	raw, err := io.ReadAll(io.LimitReader(r, 4<<20))
	if err != nil {
		return err
	}
	if !json.Valid(raw) {
		return fmt.Errorf("payload is not json")
	}

	dir, err := paths.DataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	line, err := json.Marshal(struct {
		ReceivedAt time.Time       `json:"received_at"`
		Payload    json.RawMessage `json:"payload"`
	}{now.UTC(), raw})
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(dir, "hooks-debug.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}
