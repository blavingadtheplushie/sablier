package cli

import (
	"os"

	"github.com/espcaa/sablier/internal/harness/claude"
	"github.com/spf13/cobra"
)

func newClaudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "claude [args...]",
		Short:              "run claude code through sablier",
		DisableFlagParsing: true, // every arg goes to claude, even --help
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			args, err = claude.WithHooks(exe, args)
			if err != nil {
				return err
			}
			return runHarness(cmd, claude.Harness, "claude", args)
		},
	}
}
