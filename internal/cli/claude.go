package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/harness/claude"
	"github.com/espcaa/sablier/internal/launcher"
	"github.com/espcaa/sablier/internal/paths"
	"github.com/espcaa/sablier/internal/project"
	"github.com/espcaa/sablier/internal/store"
	"github.com/spf13/cobra"
)

func newClaudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "claude [args...]",
		Short:              "run claude code through sablier",
		DisableFlagParsing: true, // every arg goes to claude, even --help
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := launcher.NewRunID()
			if err != nil {
				return err
			}
			onTick, closeRec := tickRecorder(context.Background(), id, cmd.ErrOrStderr())

			code, err := launcher.Run("claude", args, launcher.Options{RunID: id, OnTick: onTick})
			closeRec()
			if err != nil {
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}
}

// saves input ticks as events
func tickRecorder(ctx context.Context, runID string, warn io.Writer) (func(launcher.Tick), func()) {
	noop := func(launcher.Tick) {}

	dir, err := paths.DataDir()
	if err != nil {
		fmt.Fprintln(warn, "sablier: not tracking input:", err)
		return noop, func() {}
	}
	s, err := store.Open(filepath.Join(dir, "sablier.db"))
	if err != nil {
		fmt.Fprintln(warn, "sablier: not tracking input:", err)
		return noop, func() {}
	}

	var pid string
	if salt, err := s.Salt(ctx); err == nil {
		if cwd, err := os.Getwd(); err == nil {
			pid = project.ID(salt, cwd)
			s.Update(ctx, func(tx *store.Tx) error { return tx.PutProject(ctx, pid, project.Root(cwd)) })
		}
	}

	var failed int
	var lastErr error
	onTick := func(tk launcher.Tick) {
		e, err := event.New(event.InputTick, tk.At, event.InputTickData{
			WindowMs: tk.Window.Milliseconds(),
			Keys:     tk.Keys,
			Pasted:   tk.Pasted,
		})
		if err == nil {
			e.SessionID = "run:" + runID // linked to claude's session later
			e.ProjectID = pid
			e.Harness = claude.Harness
			e.Source = event.SourceLauncher
			_, err = s.Append(ctx, e)
		}
		if err != nil {
			failed++
			lastErr = err
		}
	}
	closeRec := func() {
		if failed > 0 {
			fmt.Fprintf(warn, "sablier: %d input ticks not saved: %v\n", failed, lastErr)
		}
		s.Close()
	}
	return onTick, closeRec
}
