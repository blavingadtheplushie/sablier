package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/espcaa/sablier/internal/harness/claude"
	"github.com/espcaa/sablier/internal/harness/codex"
	"github.com/espcaa/sablier/internal/paths"
	"github.com/espcaa/sablier/internal/store"
	"github.com/spf13/cobra"
)

// never print to stdout (goes into claude's context), never exit non-zero (2 blocks claude)
func newHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook [claude|codex]",
		Short:  "do not run this, for harnesses",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "claude" // default mmmh
			if len(args) == 1 {
				name = args[0]
			}
			if err := runHook(cmd.Context(), name, cmd.InOrStdin(), time.Now()); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "sablier hook:", err)
			}
		},
	}
}

type recordFunc func(ctx context.Context, s *store.Store, raw []byte, now time.Time, session string) error

var hookHarnesses = map[string]recordFunc{
	"claude": claude.Record,
	"codex":  codex.Record,
}

func runHook(ctx context.Context, name string, r io.Reader, now time.Time) error {
	if ctx == nil {
		ctx = context.Background()
	}
	record, ok := hookHarnesses[name]
	if !ok {
		return fmt.Errorf("unknown harness %q", name)
	}
	runID := os.Getenv("SABLIER_RUN_ID")
	if runID == "" {
		return nil // don't run the hook if not started with our cli
	}
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

	if os.Getenv("SABLIER_DEBUG") == "1" {
		if err := debugLog(dir, raw, now); err != nil {
			return err
		}
	}

	s, err := store.Open(filepath.Join(dir, "sablier.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	return record(ctx, s, raw, now, runSession(runID))
}

// contains prompt text, local only
func debugLog(dir string, raw []byte, now time.Time) error {
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
