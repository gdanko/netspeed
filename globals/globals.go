package globals

import (
	"context"
	"sync"
)

type Options struct {
	ListInterfaces bool   `short:"l" long:"list" description:"Display a list of interfaces and exit"`
	InterfaceName  string `short:"i" long:"interface" description:"The name of the network interface to use, e.g., en0" required:"false"`
	OutputFile     string `short:"o" long:"outfile" description:"Location of the JSON output file - output will not be written to screen" required:"false"`
	Version        func() `short:"V" long:"version" description:"Print program version"`
}

type RuntimeConfig struct {
	InterfaceName string
	OutputFile    string
}

type NetspeedData struct {
	Timestamp   uint64  `json:"timestamp"`
	Interface   string  `json:"interface"`
	KBytesRecv  float64 `json:"kbytes_recv"`
	KBytesSent  float64 `json:"kbytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
}

type IOStatsData struct {
	Interface   string  `json:"interface"`
	BytesRecv   float64 `json:"bytes_recv"`
	BytesSent   float64 `json:"bytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
}

var (
	homeDir       string
	interfaceList []string
	interfaceName string
	mu            sync.RWMutex
	exitNetspeed  context.CancelFunc
	outputFile    string
	pid           int
	pidFile       string
)

func SetPid(x int) {
	mu.Lock()
	pid = x
	mu.Unlock()
}

func GetPid() (x int) {
	mu.Lock()
	x = pid
	mu.Unlock()
	return x
}

func SetPidFile(x string) {
	mu.Lock()
	pidFile = x
	mu.Unlock()
}

func GetPidFile() (x string) {
	mu.Lock()
	x = pidFile
	mu.Unlock()
	return x
}

func SetInterfaceName(x string) {
	mu.Lock()
	interfaceName = x
	mu.Unlock()
}

func GetInterfaceName() (x string) {
	mu.Lock()
	x = interfaceName
	mu.Unlock()
	return x
}

func SetOutputFile(x string) {
	mu.Lock()
	outputFile = x
	mu.Unlock()
}

func GetOutputFile() (x string) {
	mu.Lock()
	x = outputFile
	mu.Unlock()
	return x
}

func SetInterfaceList(x []string) {
	mu.Lock()
	interfaceList = x
	mu.Unlock()
}

func GetInterfaceList() (x []string) {
	mu.Lock()
	x = interfaceList
	mu.Unlock()
	return x
}

func SetExitNetspeed(x context.CancelFunc) {
	mu.Lock()
	exitNetspeed = x
	mu.Unlock()
}

func GetExitNetspeed() (x context.CancelFunc) {
	mu.Lock()
	x = exitNetspeed
	mu.Unlock()
	return x
}

func SetHomeDir(x string) {
	mu.Lock()
	homeDir = x
	mu.Unlock()
}

func GetHomeDir() (x string) {
	mu.Lock()
	x = homeDir
	mu.Unlock()
	return x
}
