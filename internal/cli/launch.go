package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/launcher"
	"github.com/espcaa/sablier/internal/paths"
	"github.com/espcaa/sablier/internal/project"
	"github.com/espcaa/sablier/internal/store"
	"github.com/spf13/cobra"
)

// runs a harness binary through sablier, and does all the shiny things
func runHarness(cmd *cobra.Command, harness, binary string, args []string) error {
	id, err := launcher.NewRunID()
	if err != nil {
		return err
	}
	rec := newRecorder(context.Background(), harness, id, cmd.ErrOrStderr())
	stopSync := startAutoSync(rec.s, syncEvery, cmd.ErrOrStderr())

	code, err := launcher.Run(binary, args, launcher.Options{RunID: id, OnTick: rec.tick, OnFocus: rec.focus})
	stopSync() // before close, it uses the store
	rec.close()
	if err != nil {
		return err
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

func runSession(runID string) string { return "run:" + runID }

type recorder struct {
	ctx     context.Context
	s       *store.Store // nil when tracking is off
	harness string
	runID   string
	project string
	warn    io.Writer

	mu      sync.Mutex
	failed  int
	lastErr error
}

func newRecorder(ctx context.Context, harness, runID string, warn io.Writer) *recorder {
	r := &recorder{ctx: ctx, harness: harness, runID: runID, warn: warn}

	dir, err := paths.DataDir()
	if err != nil {
		fmt.Fprintln(warn, "sablier: not tracking input:", err)
		return r
	}
	r.s, err = store.Open(filepath.Join(dir, "sablier.db"))
	if err != nil {
		fmt.Fprintln(warn, "sablier: not tracking input:", err)
		return r
	}

	if cwd, err := os.Getwd(); err == nil {
		r.project = project.Detect(cwd)
	}
	return r
}

func (r *recorder) tick(tk launcher.Tick) {
	r.save(event.InputTick, tk.At, event.InputTickData{
		WindowMs: tk.Window.Milliseconds(),
		Keys:     tk.Keys,
		Mouse:    tk.Mouse,
		Pasted:   tk.Pasted,
	})
}

func (r *recorder) focus(in bool, at time.Time) {
	r.save(event.Focus, at, event.FocusData{In: in})
}

func (r *recorder) save(kind event.Kind, at time.Time, data any) {
	if r.s == nil {
		return
	}
	e, err := event.New(kind, at, data)
	if err == nil {
		e.SessionID = runSession(r.runID)
		e.ProjectID = r.project
		e.Harness = r.harness
		e.Source = event.SourceLauncher
		_, err = r.s.Append(r.ctx, e)
	}
	if err != nil {
		r.mu.Lock()
		r.failed++
		r.lastErr = err
		r.mu.Unlock()
	}
}

func (r *recorder) close() {
	if r.s == nil {
		return
	}
	r.mu.Lock()
	failed, lastErr := r.failed, r.lastErr
	r.mu.Unlock()
	if failed > 0 {
		fmt.Fprintf(r.warn, "sablier: %d events not saved: %v\n", failed, lastErr)
	}
	r.s.Close()
}
