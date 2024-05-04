package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/iostat"
	"github.com/gdanko/netspeed/output"
	"github.com/gdanko/netspeed/util"
	"github.com/spf13/cobra"
)

var (
	startCommand = &cobra.Command{
		Use:     "start",
		Short:   "Start running netspeed against an interface",
		Long:    "Start running netspeed against an interface",
		PreRunE: startPreRunCmd,
		RunE:    startRunCmd,
	}
)

func init() {
	startCommand.Flags().StringVarP(&interfaceName, "interface", "i", "", "The name of the network interface to use, e.g., en0")
	startCommand.Flags().StringVarP(&outputFile, "outfile", "o", "", "Location of the JSON output file - output will not be written to screen")
	rootCmd.AddCommand(startCommand)
}

func startPreRunCmd(cmd *cobra.Command, args []string) error {
	// Determine the home directory
	homeDir, err = util.GetHomeDir()
	if err != nil {
		message := fmt.Errorf("failed to determine your home directory: %s", err)
		util.ExitOnError(message.Error())
	}
	globals.SetHomeDir(homeDir)
	globals.SetPid(os.Getpid())
	globals.SetPidFile(util.GetPidFilename())

	// Get a list of interfaces
	interfaceList, err := iostat.GetInterfaceList()
	if err != nil {
		util.ExitOnError(err.Error())
	}
	globals.SetInterfaceList(interfaceList)

	// Make sure the specified interface is valid
	if !slices.Contains(globals.GetInterfaceList(), interfaceName) {
		util.ExitOnError(fmt.Sprintf("the specified interface \"%s\" does not exist", interfaceName))
	}
	globals.SetInterfaceName(interfaceName)

	// Verify the output file's directory is writable
	if outputFile != "" {
		var path = ""
		if strings.Contains(outputFile, "/") {
			absPath, err := filepath.Abs(outputFile)
			if err != nil {
				return fmt.Errorf("unable to determine the absolute path for \"%s\"", outputFile)
			}
			outputFile = absPath
			path = filepath.Dir(outputFile)
		} else {
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("unable to detect the current working directory")
			}
		}
		err = util.PathExistsAndIsWritable(path)
		if err != nil {
			return err
		}
		globals.SetOutputFile(outputFile)
	}

	return nil
}

func startRunCmd(cmd *cobra.Command, args []string) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		sig := <-sigChan
		fmt.Println("Received signal:", sig)
		util.ExitCleanly()
	}()

	err = util.CreatePidFile()
	if err != nil {
		util.ExitOnError(err.Error())
	}

	iostatDataOld, err := iostat.GetData()
	if err != nil {
		util.ExitOnError(err.Error())
	}
	for _, iostatBlock := range iostatDataOld {
		if iostatBlock.Interface == globals.GetInterfaceName() {
			oldBytesSent = iostatBlock.BytesSent
			oldBytesRecv = iostatBlock.BytesRecv
			oldPacketsSent = iostatBlock.PacketsSent
			oldPacketsRecv = iostatBlock.PacketsRecv
		}
	}
	time.Sleep(1 * time.Second)

	forever := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	globals.SetExitNetspeed(cancel)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for {
					iostatDataNew, err := iostat.GetData()
					if err != nil {
						util.ExitOnError(err.Error())
					}
					for _, iostatBlock := range iostatDataNew {
						if iostatBlock.Interface == globals.GetInterfaceName() {
							newBytesSent = iostatBlock.BytesSent
							newBytesRecv = iostatBlock.BytesRecv
							newPacketsSent = iostatBlock.PacketsSent
							newPacketsRecv = iostatBlock.PacketsRecv
						}
					}

					netspeedData := globals.NetspeedData{
						Timestamp:   util.GetTimestamp(),
						Interface:   globals.GetInterfaceName(),
						KBytesSent:  (newBytesSent - oldBytesSent) / 1024,
						KBytesRecv:  (newBytesRecv - oldBytesRecv) / 1024,
						PacketsSent: newPacketsSent - oldPacketsSent,
						PacketsRecv: newPacketsRecv - oldPacketsRecv,
					}

					output.ProcessOutput(netspeedData)

					oldBytesSent = newBytesSent
					oldBytesRecv = newBytesRecv
					oldPacketsSent = newPacketsSent
					oldPacketsRecv = newPacketsRecv
					time.Sleep(1 * time.Second)
				}
			}
		}
	}(ctx)

	<-forever

	return nil
}
