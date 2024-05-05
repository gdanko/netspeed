package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/iostat"
	"github.com/jessevdk/go-flags"
	"golang.org/x/sys/unix"
)

func GetTimestamp() (timestamp uint64) {
	return uint64(time.Now().Unix())
}

// Options
func ProcessOptions(opts globals.Options) (err error) {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = `--interface <interface_name> --outfile </path/to/output.json>
  netspeed calculates KiB in/out per second and writes the output to a JSON file.`
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	interfaceList, err := iostat.GetInterfaceList()
	if err != nil {
		ExitOnError(err.Error())
	}
	globals.SetInterfaceList(interfaceList)

	// if opts.ListInterfaces {
	// 	output.ListInterfaces()
	// }

	if opts.InterfaceName == "" {
		return fmt.Errorf("the required flag `-i, --interface' was not specified")
	}

	if opts.OutputFile != "" {
		var path = ""
		if strings.Contains(opts.OutputFile, "/") {
			path = filepath.Dir(opts.OutputFile)
		} else {
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("unable to detect the current working directory")
			}
		}
		err = PathExistsAndIsWritable(path)
		if err != nil {
			return err
		}
	}

	// Test the interface
	// if !slices.Contains(interfaceList, globals.GetInterfaceList()) {
	// 	return fmt.Errorf("the specified interface \"%s\" does not exist", opts.InterfaceName)
	// }

	globals.SetInterfaceName(opts.InterfaceName)
	globals.SetOutputFile(opts.OutputFile)

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
