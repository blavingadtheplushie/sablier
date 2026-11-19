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
			events, err := s.Between(cmd.Context(), midnight, now)
			if err != nil {
				return err
			}
			names, err := s.Projects(cmd.Context())
			if err != nil {
				return err
			}
			printReport(cmd.OutOrStdout(), credit.Compute(events, credit.Default, now), names)
			return nil
		},
	}
}

func printReport(w io.Writer, r credit.Report, names map[string]string) {
	if r.Actions == 0 {
		fmt.Fprintln(w, "nothing tracked today")
		return
	}

	fmt.Fprintf(w, "%s today\n", dur(r.Credited))
	fmt.Fprintf(w, "%s agent busy · %d actions · %s span\n\n", dur(r.AgentBusy), r.Actions, dur(r.Last.Sub(r.First)))

	ids := make([]string, 0, len(r.ByProject))
	for id := range r.ByProject {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return r.ByProject[ids[i]] > r.ByProject[ids[j]] })
	for _, id := range ids {
		fmt.Fprintf(w, "  %-8s %s\n", dur(r.ByProject[id]), projectName(id, names))
	}
}

func projectName(id string, names map[string]string) string {
	root, ok := names[id]
	if !ok {
		if id == "" {
			return "(no project)"
		}
		return "(unknown " + id[:min(8, len(id))] + ")"
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(root, home) {
		return "~" + strings.TrimPrefix(root, home)
	}
	return root
}

// minutes, like 0m, 27m, 1h05m
func dur(d time.Duration) string {
	m := int(d.Round(time.Minute).Minutes())
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh%02dm", m/60, m%60)
}
