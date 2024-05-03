package util

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdanko/netspeed/globals"
	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
)

func verifyProcess() (err error) {
	if fileExists(globals.GetPidFile()) {
		contents, err := os.ReadFile(globals.GetPidFile())
		// file exists but can't read it
		if err != nil {
			return fmt.Errorf("the pidfile \"%s\" exists but cannot be read", globals.GetPidFile())
		}
		pidString := strings.TrimSuffix(string(contents), "\n")
		pid, _ := strconv.ParseInt(pidString, 10, 64)
		process, err := ps.FindProcess(int(pid))
		if err != nil {
			return fmt.Errorf("could not find the pid listed in \"%s\"", globals.GetPidFile())
		}

		// pidfile exists but no process found
		if process == nil {
			err = deleteFile(globals.GetPidFile())
			if err != nil {
				return err
			}
			return nil
		}
		// pidfile exists, pid found, netspeed process exists with the pid
		if process.Executable() == "netspeed" {
			return fmt.Errorf("a process named netspeed with the pid %d is already running", pid)
		}
	}
	return nil
}

func pathExistsAndIsWritable(path string) (err error) {
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("the path \"%s\" does not exist - please choose another path", path)
	}
	ok := unix.Access(path, unix.W_OK)
	if ok != nil {
		return fmt.Errorf("the path \"%s\" is not writable - please choose another path", path)
	}
	return nil
}

func fileExists(path string) (exists bool) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func deleteFile(filename string) (err error) {
	if fileExists(filename) {
		err = os.Remove(filename)
		if err != nil {
			return fmt.Errorf("failed to remove the pidfile \"%s\", %s", filename, err)
		}
	}
	return nil
}
