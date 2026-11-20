// cmds here should stay thin: parse input, call into internal/core, format output
package cli

import (
	"github.com/spf13/cobra"
)

// exported for tools/gendocs
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "sablier",
		Short: "track the time you spend coding with ai agents",
		Long: `sablier runs your ai coding agent (claude code, opencode, codex) and counts
the time you actually spend working with it.`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(
		newVersionCmd(),
		newHookCmd(),
		newClaudeCmd(),
		newOpencodeCmd(),
		newCodexCmd(),
		newTodayCmd(),
		newLoginCmd(),
		newSyncCmd(),
		newStatusCmd(),
	)

	return root
}

func Execute() error {
	return NewRootCmd().Execute()
}
