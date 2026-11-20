package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/espcaa/sablier/internal/harness/opencode"
	"github.com/espcaa/sablier/internal/launcher"
	"github.com/espcaa/sablier/internal/project"
	"github.com/spf13/cobra"
)

func newOpencodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "opencode [args...]",
		Short:              "run opencode through sablier",
		DisableFlagParsing: true, // every arg goes to opencode, even --help
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opencode.RunsSession(args) {
				return exitWith(launcher.Run("opencode", args, launcher.Options{}))
			}
			srv, err := opencode.Start("opencode")
			if err != nil {
				return err
			}
			defer srv.Stop()
			code, err := runHarness(cmd, harnessRun{
				harness: opencode.Harness,
				binary:  "opencode",
				args:    opencode.WithServer(srv.URL, args),
				env:     srv.Env(),
				watch:   func(rec *recorder) func() { return watchOpencode(rec, srv) },
			})
			srv.Stop()
			return exitWith(code, err)
		},
	}
}

func watchOpencode(rec *recorder, srv *opencode.Server) (stop func()) {
	salt, err := rec.s.Salt(rec.ctx)
	if err != nil {
		fmt.Fprintln(rec.warn, "sablier: not tracking opencode's events:", err)
		return func() {}
	}
	m := &opencode.Mapper{Salt: salt, Session: runSession(rec.runID), Project: project.Detect, Default: rec.project}

	ctx, cancel := context.WithCancel(rec.ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.Watch(ctx, func(e opencode.Event) { rec.append(m.Map(e)) })
	}()
	return func() {
		srv.Stop() // the stream ends once the server sent everything
		cancel()
		<-done
		rec.append(m.Close(time.Now()))
	}
}
