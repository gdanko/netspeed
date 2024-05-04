package cmd

import (
	"fmt"

	"github.com/gdanko/netspeed/internal"
	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:          "version",
		Short:        "Prints the current version",
		Long:         "Prints the current version",
		RunE:         runVersionCmd,
		SilenceUsage: true,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersionCmd(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "netspeed %s\n", internal.Version(true, true))

	return nil
}
