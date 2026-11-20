package cli

import (
	"github.com/espcaa/sablier/internal/harness/claude"
	"github.com/spf13/cobra"
)

func newClaudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "claude [args...]",
		Short:              "run claude code through sablier",
		DisableFlagParsing: true, // every arg goes to claude, even --help
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarness(cmd, claude.Harness, "claude", args)
		},
	}
}
