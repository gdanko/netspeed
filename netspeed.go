package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/shirou/gopsutil/net"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const VERSION = "0.1.1"

type NetspeedData struct {
	Timestamp   uint64  `json:"timestamp"`
	Interface   string  `json:"interface"`
	KBytesRecv  float64 `json:"kbytes_recv"`
	KBytesSent  float64 `json:"kbytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
}

type Options struct {
	ListInterfaces bool   `short:"l" long:"list" description:"Display a list of interfaces and exit"`
	InterfaceName  string `short:"i" long:"interface" description:"The name of the network interface to use, e.g., en0" required:"false"`
	OutputFile     string `short:"o" long:"outfile" description:"Location of the JSON output file - output will not be written to screen" required:"false"`
	Version        func() `short:"V" long:"version" description:"Print program version"`
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

func processOptions(opts Options, interfaceList []string) (interfaceName, outputFile string, listInterfaces bool, err error) {
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

	if opts.ListInterfaces {
		return interfaceName, outputFile, true, nil
	}

	if opts.InterfaceName == "" {
		return "", "", true, fmt.Errorf("the required flag `-i, --interface' was not specified")
	}

	interfaceName = opts.InterfaceName

	if opts.OutputFile != "" {
		var path = ""
		if strings.Contains(opts.OutputFile, "/") {
			path = filepath.Dir(opts.OutputFile)
		} else {
			path, err = os.Getwd()
			if err != nil {
				return "", "", false, fmt.Errorf("unable to detect the current working directory")
			}
		}
		err = pathExistsAndIsWritable(path)
		if err != nil {
			return "", "", false, err
		}
	}

	// Test the interface
	if !slices.Contains(interfaceList, opts.InterfaceName) {
		return interfaceName, opts.OutputFile, false, fmt.Errorf("the specified interface \"%s\" does not exist", opts.InterfaceName)
	}
	return interfaceName, opts.OutputFile, false, nil
}

func getInterfaceList() (interfaceList []string, err error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return interfaceList, nil
	}
	for _, ifaceBlock := range ioCounters {
		interfaceList = append(interfaceList, ifaceBlock.Name)
	}
	sort.Slice(interfaceList, func(i, j int) bool {
		return interfaceList[i] < interfaceList[j]
	})
	return interfaceList, nil
}

func getData(iface string) (bytesSent float64, bytesRecv float64, packetsSent uint64, packetsRecv uint64, err error) {
	var interfaceFound = false
	var myInterface net.IOCountersStat
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return bytesSent, bytesRecv, packetsSent, packetsRecv, err
	}

	for _, ifaceBlock := range ioCounters {
		if ifaceBlock.Name == iface {
			interfaceFound = true
			myInterface = ifaceBlock
		}
	}

	if !interfaceFound {
		return bytesSent, bytesRecv, packetsSent, packetsRecv, fmt.Errorf("the specified interface \"%s\" was not found", iface)
	}

	bytesSent = float64(myInterface.BytesSent)
	bytesRecv = float64(myInterface.BytesRecv)
	packetsSent = uint64(myInterface.PacketsSent)
	packetsRecv = uint64(myInterface.PacketsRecv)

	return bytesSent, bytesRecv, packetsSent, packetsRecv, nil
}

func getTimestamp() (timestamp uint64) {
	return uint64(time.Now().Unix())
}
func main() {
	pid := os.Getpid()
	fmt.Println(pid)
	os.Exit(0)
	log.SetFormatter(&log.TextFormatter{
		DisableColors:          false,
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		TimestampFormat:        "2006-01-02 15:04:05",
	})

	interfaceList, err := getInterfaceList()
	if err != nil {
		log.Fatal(err)
	}

	opts := Options{}

	opts.Version = func() {
		fmt.Printf("netspeed version %s\n", VERSION)
		os.Exit(0)
	}

	interfaceName, outputFile, listInterfaces, err := processOptions(opts, interfaceList)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if listInterfaces {
		fmt.Println("Available Interfaces:")
		for _, interfaceName := range interfaceList {
			fmt.Printf("  %s\n", interfaceName)
		}
		os.Exit(0)
	}

	// Get the first sample
	oldBytesSent, oldBytesRecv, oldPacketsSent, oldPacketsRecv, err := getData(interfaceName)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	for {
		newBytesSent, newBytesRecv, newPacketsSent, newPacketsRecv, err := getData(interfaceName)
		if err != nil {
			panic(err)
		}

		netspeedData := NetspeedData{
			Timestamp:   getTimestamp(),
			Interface:   interfaceName,
			KBytesSent:  (newBytesSent - oldBytesSent) / 1024,
			KBytesRecv:  (newBytesRecv - oldBytesRecv) / 1024,
			PacketsSent: newPacketsSent - oldPacketsSent,
			PacketsRecv: newPacketsRecv - oldPacketsRecv,
		}

		jsonBytes, err := json.Marshal(netspeedData)
		if err != nil {
			log.Fatal(err)
		}

		if outputFile == "" {
			fmt.Println(string(jsonBytes))
		} else {
			err = os.WriteFile(outputFile, jsonBytes, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}

		oldBytesSent = newBytesSent
		oldBytesRecv = newBytesRecv
		oldPacketsSent = newPacketsSent
		oldPacketsRecv = newPacketsRecv
		time.Sleep(1 * time.Second)
	}
}
