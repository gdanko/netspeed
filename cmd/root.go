package cmd

import "github.com/spf13/cobra"

var (
	err            error
	homeDir        string
	interfaceName  string
	newBytesRecv   float64
	newBytesSent   float64
	newPacketsRecv uint64
	newPacketsSent uint64
	oldBytesRecv   float64
	oldBytesSent   float64
	oldPacketsRecv uint64
	oldPacketsSent uint64
	outputFile     string
	timestamp      uint64
	rootCmd        = &cobra.Command{
		Use:   "netspeed",
		Short: "netspeed calculates KiB in/out per second",
		Long:  "netspeed calculates KiB in/out per second",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {

}
