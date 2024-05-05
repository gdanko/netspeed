package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/internal"
	"github.com/gdanko/netspeed/iostat"
	"github.com/gdanko/netspeed/output"
	"github.com/gdanko/netspeed/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		sig := <-sigChan
		fmt.Println("Received signal:", sig)
		util.ExitCleanly()
	}()

	var err error
	var opts globals.Options
	var oldBytesSent, oldBytesRecv, newBytesSent, newBytesRecv float64 = 0, 0, 0, 0
	var oldPacketsSent, oldPacketsRecv, newPacketsSent, newPacketsRecv uint64 = 0, 0, 0, 0

	log.SetFormatter(&log.TextFormatter{
		DisableColors:          false,
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		TimestampFormat:        "2006-01-02 15:04:05",
	})

	interfaceList, err := iostat.GetInterfaceList()
	if err != nil {
		util.ExitOnError(err.Error())
	}
	globals.SetInterfaceList(interfaceList)

	opts = globals.Options{}

	opts.Version = func() {
		fmt.Printf("netspeed version %s\n", internal.Version(false, true))
		os.Exit(0)
	}

	err = util.ProcessOptions(opts)
	if err != nil {
		util.ExitOnError(err.Error())
	}

	// Get the first sample
	iostatDataOld, err := iostat.GetData()
	if err != nil {
		util.ExitOnError(err.Error())
	}
	for _, iostatBlock := range iostatDataOld {
		if iostatBlock.Interface == globals.GetInterfaceName() {
			oldBytesSent = iostatBlock.BytesSent
			oldBytesRecv = iostatBlock.BytesRecv
			oldPacketsSent = iostatBlock.PacketsSent
			oldPacketsRecv = iostatBlock.PacketsRecv
		}
	}
	time.Sleep(1 * time.Second)

	for {
		iostatDataNew, err := iostat.GetData()
		if err != nil {
			util.ExitOnError(err.Error())
		}
		for _, iostatBlock := range iostatDataNew {
			if iostatBlock.Interface == globals.GetInterfaceName() {
				newBytesSent = iostatBlock.BytesSent
				newBytesRecv = iostatBlock.BytesRecv
				newPacketsSent = iostatBlock.PacketsSent
				newPacketsRecv = iostatBlock.PacketsRecv
			}
		}

		netspeedData := globals.NetspeedData{
			Timestamp:   util.GetTimestamp(),
			Interface:   globals.GetInterfaceName(),
			KBytesSent:  (newBytesSent - oldBytesSent) / 1024,
			KBytesRecv:  (newBytesRecv - oldBytesRecv) / 1024,
			PacketsSent: newPacketsSent - oldPacketsSent,
			PacketsRecv: newPacketsRecv - oldPacketsRecv,
		}

		output.ProcessOutput(netspeedData)

		oldBytesSent = newBytesSent
		oldBytesRecv = newBytesRecv
		oldPacketsSent = newPacketsSent
		oldPacketsRecv = newPacketsRecv
		time.Sleep(1 * time.Second)
	}
}
