package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gdanko/netspeed/globals"
	"golang.org/x/sys/unix"
)

func GetTimestamp() (timestamp uint64) {
	return uint64(time.Now().Unix())
}

func GetHomeDir() (path string, err error) {
	user, err := user.Current()
	if err != nil {
		return path, err
	}
	return user.HomeDir, nil
}

func GetPidFilename() (pidfile string) {
	return filepath.Join(globals.GetHomeDir(), ".netspeed.pid")
}

func CreatePidFile() (err error) {
	if fileExists(globals.GetPidFile()) {
		err = verifyProcess()
		fmt.Println(err)
		os.Exit(0)
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

func DeleteFile(filename string) (err error) {
	if fileExists(filename) {
		fmt.Println(filename)
		err = os.Remove(filename)
		fmt.Println(err)
		if err != nil {
			return fmt.Errorf("failed to remove the file \"%s\", %s", filename, err)
		}
	}
	return nil
}

func ExitCleanly() {
	Exit("")
}

func ExitOnError(errorMessage string) {
	Exit(errorMessage)
}

func Exit(errorMessage string) {
	var err error

	fmt.Println("Shutting down")

	globals.GetExitNetspeed()
	time.Sleep(500 * time.Millisecond)

	for _, filename := range []string{globals.GetPidFile(), globals.GetOutputFile()} {
		err = DeleteFile(filename)
		if err != nil {
			fmt.Println(err)
			fmt.Println("please delete it manually")
		}
	}

	if errorMessage != "" {
		fmt.Println(errorMessage)
		os.Exit(1)
	}
	os.Exit(0)
}
