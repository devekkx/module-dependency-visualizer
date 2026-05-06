package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version and metadata",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), deps.BuildInfo.String())
			return nil
		},
	}
}
