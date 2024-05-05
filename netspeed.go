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
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/gdanko/netspeed/internal"
	"github.com/gdanko/netspeed/iostat"
	"github.com/gdanko/netspeed/util"
	flags "github.com/jessevdk/go-flags"
)

const defaultTick = 1 * time.Second

type Config struct {
	InterfaceName  string
	OutputFile     string
	ListInterfaces bool
	PrintVersion   bool
	InterfaceList  []string
	Tick           time.Duration
}

type Options struct {
	ListInterfaces bool   `short:"l" long:"list" description:"Display a list of interfaces and exit"`
	InterfaceName  string `short:"i" long:"interface" description:"The name of the network interface to use, e.g., en0" required:"false"`
	OutputFile     string `short:"o" long:"outfile" description:"Location of the JSON output file - output will not be written to screen" required:"false"`
	PrintVersion   bool   `short:"V" long:"version" description:"Print program version"`
}

type NetspeedData struct {
	Timestamp   uint64  `json:"timestamp"`
	Interface   string  `json:"interface"`
	KBytesRecv  float64 `json:"kbytes_recv"`
	KBytesSent  float64 `json:"kbytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
}

func (c *Config) init(args []string) error {
	var (
		opts   Options
		parser *flags.Parser
	)

	opts = Options{}
	parser = flags.NewParser(&opts, flags.Default)
	parser.Usage = `--interface <interface_name> [--outfile </path/to/output.json>] [--list]
  netspeed calculates KiB in/out per second and optionally writes the output to a JSON file.`
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if len(os.Args) == 1 {
		parser.WriteHelp(os.Stderr)
		c.ExitError("")
	}

	c.InterfaceName = opts.InterfaceName
	c.OutputFile = opts.OutputFile
	c.ListInterfaces = opts.ListInterfaces
	c.PrintVersion = opts.PrintVersion

	return nil
}

func (c *Config) ExitError(errorMessage string) {
	fmt.Fprintf(os.Stderr, "%s\n", errorMessage)
	os.Exit(1)
}

func (c *Config) ExitCleanly() {
	os.Exit(0)
}

func (c *Config) PopulateInterfaces() (err error) {
	c.InterfaceList, err = iostat.GetInterfaceList()
	if err != nil {
		return fmt.Errorf("failed to populate the list of interfaces: %s", err)
	}
	return nil
}

func (c *Config) ShowVersion() {
	fmt.Fprintf(os.Stdout, "netspeed version %s\n", internal.Version(false, true))
}

func (c *Config) ShowInterfaces() {
	fmt.Fprintf(os.Stderr, "Available Interfaces:\n")
	for _, interfaceName := range c.InterfaceList {
		fmt.Fprintf(os.Stderr, "  %s\n", interfaceName)
	}
}

func (c *Config) ValidateOptions() (err error) {
	err = c.PopulateInterfaces()
	if err != nil {
		return err
	}

	if c.PrintVersion {
		c.ShowVersion()
		c.ExitCleanly()
	}

	if c.ListInterfaces {
		c.ShowInterfaces()
		c.ExitCleanly()
	}

	if c.InterfaceName == "" {
		return fmt.Errorf("the required flag `-i, --interface' was not specified")
	}

	if !slices.Contains(c.InterfaceList, c.InterfaceName) {
		return fmt.Errorf("the interface \"%s\" does not exist", c.InterfaceName)
	}

	if c.OutputFile != "" {
		var path = ""
		if strings.Contains(c.OutputFile, "/") {
			absolutePath, err := filepath.Abs(c.OutputFile)
			if err != nil {
				return fmt.Errorf("failed to determine the absolute path for %s", c.OutputFile)
			}
			path = filepath.Dir(absolutePath)
			err = util.PathExistsAndIsWritable(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) ProcessOutput(netspeedData NetspeedData) {
	jsonBytes, err := json.Marshal(netspeedData)
	if err != nil {
		c.ExitError(err.Error())
	}

	if c.OutputFile == "" {
		fmt.Fprintln(os.Stdout, string(jsonBytes))
	} else {
		err = os.WriteFile(c.OutputFile, jsonBytes, 0644)
		if err != nil {
			c.ExitError(err.Error())
		}
	}
}

func (c *Config) CleanUp() (err error) {
	err = util.DeleteFile(c.OutputFile)
	if err != nil {
		return err
	}
	return nil
}

func Run(ctx context.Context, c *Config, out io.Writer) error {
	log.SetOutput(out)

	var oldBytesSent, oldBytesRecv, newBytesSent, newBytesRecv float64 = 0, 0, 0, 0
	var oldPacketsSent, oldPacketsRecv, newPacketsSent, newPacketsRecv uint64 = 0, 0, 0, 0

	for {
		select {
		case <-ctx.Done():
			return nil
		// case <-time.Tick(c.Tick):
		default:
			iostatDataNew, err := iostat.GetData()
			if err != nil {
				c.ExitError(err.Error())
			}
			for _, iostatBlock := range iostatDataNew {
				if iostatBlock.Interface == c.InterfaceName {
					newBytesSent = iostatBlock.BytesSent
					newBytesRecv = iostatBlock.BytesRecv
					newPacketsSent = iostatBlock.PacketsSent
					newPacketsRecv = iostatBlock.PacketsRecv
				}
			}
			netspeedData := NetspeedData{
				Timestamp:   util.GetTimestamp(),
				Interface:   c.InterfaceName,
				KBytesSent:  (newBytesSent - oldBytesSent) / 1024,
				KBytesRecv:  (newBytesRecv - oldBytesRecv) / 1024,
				PacketsSent: newPacketsSent - oldPacketsSent,
				PacketsRecv: newPacketsRecv - oldPacketsRecv,
			}

			c.ProcessOutput(netspeedData)

			oldBytesSent = newBytesSent
			oldBytesRecv = newBytesRecv
			oldPacketsSent = newPacketsSent
			oldPacketsRecv = newPacketsRecv
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
				// cancel()
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
