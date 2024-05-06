package main

// https://ieftimov.com/posts/four-steps-daemonize-your-golang-programs/

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdanko/netspeed/internal"
	"github.com/gdanko/netspeed/iostat"
	"github.com/gdanko/netspeed/util"
	flags "github.com/jessevdk/go-flags"
)

type Config struct {
	JSON         bool
	PrintVersion bool
	Lockfile     string
	OutputFile   string
}

type Options struct {
	JSON         bool `short:"j" long:"json" description:"Save the output to /tmp/netspeed.json instead of to STDOUT.\nOnly the current iteration is saved to file."`
	PrintVersion bool `short:"V" long:"version" description:"Print program version"`
}

type NetspeedInterfaceData struct {
	Interface   string  `json:"interface"`
	BytesRecv   float64 `json:"bytes_recv"`
	BytesSent   float64 `json:"bytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
}

type NetspeedData struct {
	Timestamp  uint64                  `json:"timestamp"`
	Interfaces []NetspeedInterfaceData `json:"interfaces"`
}

type IOStatData struct {
	Interfaces []iostat.IOStatData `json:"interfaces"`
}

func (c *Config) init(args []string) error {
	var (
		opts   Options
		parser *flags.Parser
	)

	opts = Options{}
	parser = flags.NewParser(&opts, flags.Default)
	parser.Usage = `[-j, --json] [-V, --version] 
  netspeed prints bytes in/out per second and packets sent/received per second for all interfaces`
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	c.JSON = opts.JSON
	c.PrintVersion = opts.PrintVersion
	c.Lockfile = "/tmp/netspeed.lock"
	c.OutputFile = "/tmp/netspeed.json"

	return nil
}

func (c *Config) ExitError(errorMessage string) {
	c.CleanUp()
	fmt.Fprintf(os.Stderr, "%s\n", errorMessage)
	os.Exit(1)
}

func (c *Config) ExitCleanly() {
	c.CleanUp()
	os.Exit(0)
}

func (c *Config) CreateLockfile() (err error) {
	f, err := os.Create(c.Lockfile)
	if err != nil {
		return fmt.Errorf("failed to create the lockfile \"%s\"", c.Lockfile)
	}
	defer f.Close()

	return nil
}

func (c *Config) ShowVersion() {
	fmt.Fprintf(os.Stdout, "netspeed version %s\n", internal.Version(false, true))
}

func (c *Config) ValidateOptions() (err error) {

	return nil
}

func (c *Config) ProcessOutput(netspeedData NetspeedData) {
	jsonBytes, err := json.Marshal(netspeedData)
	if err != nil {
		c.ExitError(err.Error())
	}

	if !c.JSON {
		fmt.Fprintln(os.Stdout, string(jsonBytes))
	} else {
		err = os.WriteFile(c.OutputFile, jsonBytes, 0644)
		if err != nil {
			c.ExitError(err.Error())
		}
	}
}

func (c *Config) CleanUp() (err error) {
	for _, filename := range []string{c.OutputFile, c.Lockfile} {
		err = util.DeleteFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
	}
	return nil
}

func (c *Config) FindInterface(interfaceName string, interfaceList []iostat.IOStatData) (iostatEntry iostat.IOStatData, err error) {
	for _, iostatEntry = range interfaceList {
		if interfaceName == iostatEntry.Interface {
			return iostatEntry, nil
		}
	}
	return iostat.IOStatData{}, fmt.Errorf("the interface \"%s\" was not found in this block", interfaceName)
}

func Run(ctx context.Context, c *Config, out io.Writer) error {
	if c.PrintVersion {
		c.ShowVersion()
		c.ExitCleanly()
	}

	var iostatDataOld = IOStatData{}
	var iostatDataNew = IOStatData{}
	var netspeedData = NetspeedData{}

	if util.FileExists(c.Lockfile) {
		return fmt.Errorf("the lockfile \"%s\" already exists - the program is probably already running", c.Lockfile)
	}

	err := c.CreateLockfile()
	if err != nil {
		return err
	}

	log.SetOutput(out)

	// Get the first sample
	data, err := iostat.GetData()
	if err != nil {
		c.ExitError(err.Error())
	}
	iostatDataOld.Interfaces = data
	time.Sleep(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Clear out New at each iteration
			netspeedData = NetspeedData{
				Timestamp: util.GetTimestamp(),
			}

			data, err := iostat.GetData()
			if err != nil {
				// breaking out here will cause disruption for one iteration but
				// should normalize iterself naturally
				break
			}
			iostatDataNew.Interfaces = data

			for _, iostatBlock := range iostatDataNew.Interfaces {
				var foundInOld, foundInNew = false, false

				interfaceName := iostatBlock.Interface
				interfaceOld, err := c.FindInterface(interfaceName, iostatDataOld.Interfaces)
				if err == nil {
					foundInOld = true
				}
				foundInOld = true

				interfaceNew, err := c.FindInterface(interfaceName, iostatDataNew.Interfaces)
				if err == nil {
					foundInNew = true
				}

				// Only add the block if the interface name was found in both old and new blocks
				if foundInOld && foundInNew {
					netspeedData.Interfaces = append(netspeedData.Interfaces, NetspeedInterfaceData{
						Interface:   interfaceNew.Interface,
						BytesSent:   interfaceNew.BytesSent - interfaceOld.BytesSent,
						BytesRecv:   interfaceNew.BytesRecv - interfaceOld.BytesRecv,
						PacketsSent: interfaceNew.PacketsSent - interfaceOld.PacketsSent,
						PacketsRecv: interfaceNew.PacketsRecv - interfaceOld.PacketsRecv,
					})
				}
			}

			c.ProcessOutput(netspeedData)

			iostatDataOld.Interfaces = iostatDataNew.Interfaces
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	var err error

	c := &Config{}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case s := <-signalChan:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Printf("Got SIGINT/SIGTERM, exiting.")
				err = c.CleanUp()
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err.Error())
					os.Exit(1)
				}
				os.Exit(1)
			case syscall.SIGHUP:
				log.Printf("Got SIGHUP, reloading.")
				c.init(os.Args)
			}
		case <-ctx.Done():
			log.Printf("Done.")
			err = c.CleanUp()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}
			os.Exit(1)
		}
	}()

	defer func() {
		cancel()
	}()

	err = c.init(os.Args)
	if err != nil {
		c.ExitError(err.Error())
	}

	err = c.ValidateOptions()
	if err != nil {
		c.ExitError(err.Error())
	}

	if err := Run(ctx, c, os.Stdout); err != nil {
		c.ExitError(err.Error())
	}
}
