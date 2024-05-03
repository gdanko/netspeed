package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/util"
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
	homeDir, err = util.GetHomeDir()
	if err != nil {
		message := fmt.Errorf("failed to determine your home directory: %s", err)
		util.ExitOnError(message.Error())
	}
	globals.SetHomeDir(homeDir)
	globals.SetPidFile(util.GetPidFilename())

	fmt.Println(globals.GetPidFile())

	if util.FileExists(globals.GetPidFile()) {
		contents, err := util.ReadFile(globals.GetPidFile())
		if err != nil {
			util.ExitOnError(err.Error())
		}

		pidString := strings.TrimSuffix(string(contents), "\n")
		pid, _ := strconv.ParseInt(pidString, 10, 64)

		process, err := util.FindProcess(int(pid))
		if err != nil {
			err = util.DeleteFile(globals.GetPidFile())
			if err != nil {
				return err
			}
		}

		if process.Executable() == "netspeed" {
			globals.SetPid(int(pid))
		}
	}

	return nil
}

func stopRunCmd(cmd *cobra.Command, args []string) error {
	util.CleanUp()
	err := syscall.Kill(globals.GetPid(), syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to kill pid %d: %s", globals.GetPid(), err)
	}
	return nil
}
