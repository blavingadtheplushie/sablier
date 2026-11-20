package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/espcaa/sablier/internal/credit"
	"github.com/espcaa/sablier/internal/paths"
	"github.com/espcaa/sablier/internal/store"
	"github.com/spf13/cobra"
)

func newTodayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "today",
		Short: "time since midnight",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := paths.DataDir()
			if err != nil {
				return err
			}
			s, err := store.Open(filepath.Join(dir, "sablier.db"))
			if err != nil {
				return err
			}
			defer s.Close()

			now := time.Now()
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			// the end is exclusive, so events from this very millisecond need the +1ms
			events, err := s.Between(cmd.Context(), midnight, now.Add(time.Millisecond))
			if err != nil {
				return err
			}
			printReport(cmd.OutOrStdout(), credit.Compute(events, credit.Default, now))
			return nil
		},
	}
}

func printReport(w io.Writer, r credit.Report) {
	if r.Actions == 0 {
		fmt.Fprintln(w, "nothing tracked today")
		return
	}

	fmt.Fprintf(w, "%s today\n", credit.Format(r.Credited))
	fmt.Fprintf(w, "%s agent busy · %d actions · %s span\n\n", credit.Format(r.AgentBusy), r.Actions, credit.Format(r.Last.Sub(r.First)))

	ids := make([]string, 0, len(r.ByProject))
	for id := range r.ByProject {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return r.ByProject[ids[i]] > r.ByProject[ids[j]] })
	for _, id := range ids {
		fmt.Fprintf(w, "  %-8s %s\n", credit.Format(r.ByProject[id]), projectLabel(id))
	}
}

func projectLabel(id string) string {
	if id == "" {
		return "(no project)"
	}
	return id
}

// shortens paths under the home dir to ~/...
func tilde(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
