package cli

import (
	"os"

	"github.com/espcaa/sablier/internal/launcher"
	"github.com/spf13/cobra"
)

func newClaudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "claude [args...]",
		Short:              "run claude code through sablier",
		DisableFlagParsing: true, // every arg goes to claude, even --help
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := launcher.Run("claude", args)
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
