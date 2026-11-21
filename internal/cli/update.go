package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/espcaa/sablier/internal/update"
	"github.com/espcaa/sablier/internal/version"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update sablier to the latest release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			current := version.Version
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			if exe, err = filepath.EvalSymlinks(exe); err != nil {
				return err
			}

			c := http.DefaultClient
			tag := to
			if tag == "" {
				if tag, err = update.Latest(cmd.Context(), c); err != nil {
					return err
				}
				if !update.Newer(tag, current) {
					fmt.Fprintf(out, "sablier %s is the latest\n", current)
					return nil
				}
			}

			base := os.Getenv("SABLIER_RELEASE_URL")
			if base == "" {
				base = update.ReleaseURL(tag)
			}
			fmt.Fprintf(out, "downloading sablier %s\n", tag)
			bin, err := update.Download(cmd.Context(), c, base)
			if err != nil {
				return err
			}
			if err := update.Replace(exe, bin); err != nil {
				return err
			}
			fmt.Fprintf(out, "updated %s from %s to %s\n", exe, current, tag)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "version", "", "install this release instead of the latest, like v0.1.0")
	return cmd
}
