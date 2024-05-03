package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/output"
	flags "github.com/jessevdk/go-flags"
)

func ProcessOptions(opts globals.Options, interfaceList []string) (err error) {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = `--interface <interface_name> --outfile </path/to/output.json>
  netspeed calculates KiB in/out per second and writes the output to a JSON file.`
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			ExitCleanly()
		} else {
			ExitOnError("")
		}
	}

	if opts.ListInterfaces {
		output.ShowAvailableInterfaces()
	}

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
		err = pathExistsAndIsWritable(path)
		if err != nil {
			return err
		}
	}

	// Test the interface
	if !slices.Contains(interfaceList, opts.InterfaceName) {
		return fmt.Errorf("the specified interface \"%s\" does not exist", opts.InterfaceName)
	}
	globals.SetInterfaceName(opts.InterfaceName)
	globals.SetOutputFile(opts.OutputFile)

	return nil
}

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

func GetPidFilename(homeDir string) (pidfile string) {
	return filepath.Join(homeDir, ".netspeed.pid")
}

func DeletePidFile() (err error) {
	if fileExists(globals.GetPidFile()) {
		err = os.Remove(globals.GetPidFile())
		if err != nil {
			return fmt.Errorf("failed to remove the pidfile \"%s\", %s", globals.GetPidFile(), err)
		}
	}
	return nil
}

func CreatePidFile() (err error) {
	if fileExists(globals.GetPidFile()) {
		err = verifyProcess()
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

func ExitCleanly() {
	err := DeletePidFile()
	if err != nil {
		fmt.Println(err)
		fmt.Println("please delete it manually")
		os.Exit(1)
	}
	os.Exit(0)
}

func ExitOnError(message string) {
	err := DeletePidFile()
	if err != nil {
		fmt.Println(err)
		fmt.Println("please delete it manually")
		os.Exit(1)
	}
	if message != "" {
		fmt.Println(message)
	}
	os.Exit(1)
}
