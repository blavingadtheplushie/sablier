package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/espcaa/sablier/internal/api"
	"github.com/espcaa/sablier/internal/config"
	"github.com/espcaa/sablier/internal/credit"
	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/harness/claude"
	"github.com/espcaa/sablier/internal/paths"
	"github.com/espcaa/sablier/internal/store"
	"github.com/spf13/cobra"
)

// harness id and the binary the launcher runs
var harnesses = []struct{ id, bin string }{
	{claude.Harness, "claude"},
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "check the account, sync and each harness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return err
			}
			dir, err := paths.DataDir()
			if err != nil {
				return err
			}
			s, err := store.Open(filepath.Join(dir, "sablier.db"))
			if err != nil {
				return err
			}
			defer s.Close()
			return printStatus(cmd.Context(), cmd.OutOrStdout(), c, s, time.Now())
		},
	}
}

func printStatus(ctx context.Context, w io.Writer, c config.Config, s *store.Store, now time.Time) error {
	if ctx == nil {
		ctx = context.Background()
	}
	fmt.Fprintln(w, account(ctx, c))

	pending, last, err := s.SyncStatus(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%d events waiting · last sync %s\n\n", pending, ago(last, now))

	for _, h := range harnesses {
		path, err := exec.LookPath(h.bin)
		if err != nil {
			path = h.bin + " not found"
		}
		hooks, err := s.LastEventFrom(ctx, h.id, event.SourceAdapter)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s  %s · hooks last ran %s\n", h.id, tilde(path), ago(hooks, now))
	}
	return nil
}

func account(ctx context.Context, c config.Config) string {
	if c.APIKey == "" {
		return "not signed in, run sablier login"
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	u, err := api.New(c.APIURL, c.APIKey).Me(ctx)
	switch {
	case errors.Is(err, api.ErrUnauthorized):
		return "api key rejected by " + c.APIURL + ", run sablier login again"
	case err != nil:
		return err.Error()
	}
	return "signed in as " + u.Name + " on " + c.APIURL
}

func ago(t, now time.Time) string {
	switch d := now.Sub(t); {
	case t.IsZero():
		return "never"
	case d < time.Minute:
		return "just now"
	case d < 24*time.Hour:
		return credit.Format(d) + " ago"
	default:
		return t.Format("Jan 2")
	}
}
