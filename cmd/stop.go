package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	stopCommand = &cobra.Command{
		Use:     "stop",
		Short:   "Stop netspeed if it's running",
		Long:    "Stop netspeed if it's running",
		PreRunE: stopPreRunCmd,
		RunE:    stopRunCmd,
	}
)

func init() {
	rootCmd.AddCommand(stopCommand)
}

func stopPreRunCmd(cmd *cobra.Command, args []string) error {
	// if err = GetCommonFlagValues(cmd); err != nil {
	// 	return err
	// }
	// if err = GetCommonSearchLatestFlagValues(cmd); err != nil {
	// 	return err
	// }
	fmt.Println("stop")
	return nil
}

func stopRunCmd(cmd *cobra.Command, args []string) error {
	fmt.Println("stop")
	return nil
}
