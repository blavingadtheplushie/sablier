// cmds here should stay thin: parse input, call into internal/core, format output
package cli

import (
	"github.com/spf13/cobra"
)

// flags shared by every subcommand
type globalOptions struct {
	verbose bool
}

// exported for tools/gendocs
func NewRootCmd() *cobra.Command {
	opts := &globalOptions{}

	root := &cobra.Command{
		Use:           "sablier",
		Short:         "sablier - TODO: one-line description",
		Long:          "sablier - TODO: longer description shown in `sablier --help`.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable verbose output")

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
