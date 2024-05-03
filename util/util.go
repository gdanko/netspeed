package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdanko/netspeed/globals"
	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/unix"
)

func GetTimestamp() (timestamp uint64) {
	return uint64(time.Now().Unix())
}

// Process functions
func FindProcess(pid int) (process ps.Process, err error) {
	process, err = ps.FindProcess(int(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to get a process for pid %d", pid)
	}

	if process == nil {
		return nil, fmt.Errorf("no process found with the pid %d", pid)
	}
	return process, nil
}

func VerifyProcess() (err error) {
	if !FileExists(globals.GetPidFile()) {
		// pidfile doesn't exist
		return nil
	}

	contents, err := ReadFile(globals.GetPidFile())
	if err != nil {
		// failed to read the file
		return err
	}

	pidString := strings.TrimSuffix(string(contents), "\n")
	pid, _ := strconv.ParseInt(pidString, 10, 64)

	// try to get the process using the pid
	process, err := FindProcess(int(pid))
	if err != nil {
		err = DeleteFile(globals.GetPidFile())
		if err != nil {
			return err
		}
	}

	if process.Executable() == "netspeed" {
		return fmt.Errorf("a process named netspeed with the pid %d is already running", pid)
	}

	return nil
}

// PID functions
func GetPidFilename() (pidfile string) {
	return filepath.Join(globals.GetHomeDir(), ".netspeed.pid")
}

func CreatePidFile() (err error) {
	if FileExists(globals.GetPidFile()) {
		err = VerifyProcess()
		if err != nil {
			return err
		}
	}
	err = os.WriteFile(globals.GetPidFile(), []byte(strconv.Itoa(globals.GetPid())), 0644)
	if err != nil {
		return err
	}
	return nil
}

// Path and file functions
func GetHomeDir() (path string, err error) {
	user, err := user.Current()
	if err != nil {
		return path, err
	}
	return user.HomeDir, nil
}

func PathExistsAndIsWritable(path string) (err error) {
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

func FileExists(path string) (exists bool) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func ReadFile(filename string) (string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read \"%s\"", filename)
	}
	return string(bytes), nil
}

func DeleteFile(filename string) (err error) {
	if FileExists(filename) {
		err = os.Remove(filename)
		if err != nil {
			return fmt.Errorf("failed to remove the file \"%s\", %s", filename, err)
		}
	}
	return nil
}

func CleanUp() {
	for _, filename := range []string{globals.GetPidFile(), globals.GetOutputFile()} {
		err := DeleteFile(filename)
		if err != nil {
			fmt.Println(err)
			fmt.Println("please delete it manually")
		}
	}
}

// Exit functions
func ExitCleanly() {
	Exit("")
}

func ExitOnError(errorMessage string) {
	Exit(errorMessage)
}

func Exit(errorMessage string) {
	globals.GetExitNetspeed()
	time.Sleep(500 * time.Millisecond)

	CleanUp()

	if errorMessage != "" {
		fmt.Println(errorMessage)
		os.Exit(1)
	}
	os.Exit(0)
}
