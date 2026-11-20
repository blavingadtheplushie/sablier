package cli

import (
	"os"

	"github.com/espcaa/sablier/internal/harness/codex"
	"github.com/spf13/cobra"
)

func newCodexCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "codex [args...]",
		Short:              "run codex through sablier",
		DisableFlagParsing: true, // every arg goes to codex, even --help
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			return exitWith(runHarness(cmd, harnessRun{harness: codex.Harness, binary: "codex", args: codex.WithHooks(exe, args)}))
		},
	}
}
