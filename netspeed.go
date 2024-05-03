package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/iostat"
	"github.com/gdanko/netspeed/output"
	"github.com/gdanko/netspeed/util"
)

const VERSION = "0.2.0"

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		sig := <-sigChan
		fmt.Println("Received signal:", sig)
		util.ExitCleanly()
	}()

	var oldBytesSent, oldBytesRecv, newBytesSent, newBytesRecv float64 = 0, 0, 0, 0
	var oldPacketsSent, oldPacketsRecv, newPacketsSent, newPacketsRecv uint64 = 0, 0, 0, 0

	homeDir, err := util.GetHomeDir()
	if err != nil {
		message := fmt.Errorf("failed to determine your home directory: %s", err)
		util.ExitOnError(message.Error())
	}
	globals.SetPid(os.Getpid())
	globals.SetPidFile(util.GetPidFilename(homeDir))

	interfaceList, err := iostat.GetInterfaceList()
	if err != nil {
		util.ExitOnError(err.Error())
	}
	globals.SetInterfaceList(interfaceList)

	opts := globals.Options{}

	opts.Version = func() {
		fmt.Printf("netspeed version %s\n", VERSION)
		util.ExitCleanly()
	}

	err = util.ProcessOptions(opts)
	if err != nil {
		util.ExitOnError(err.Error())
	}

	err = util.CreatePidFile()
	if err != nil {
		util.ExitOnError(err.Error())
	}

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

	// globals.SetNetspeedChan(make(chan bool))
	// defer close(globals.GetNetspeedChan())

	forever := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	globals.SetExitNetspeed(cancel)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
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
		}
	}(ctx)

	<-forever

	// for {
	// 	iostatDataNew, err := iostat.GetData()
	// 	if err != nil {
	// 		util.ExitOnError(err.Error())
	// 	}
	// 	for _, iostatBlock := range iostatDataNew {
	// 		if iostatBlock.Interface == globals.GetInterfaceName() {
	// 			newBytesSent = iostatBlock.BytesSent
	// 			newBytesRecv = iostatBlock.BytesRecv
	// 			newPacketsSent = iostatBlock.PacketsSent
	// 			newPacketsRecv = iostatBlock.PacketsRecv
	// 		}
	// 	}

	// 	netspeedData := globals.NetspeedData{
	// 		Timestamp:   util.GetTimestamp(),
	// 		Interface:   globals.GetInterfaceName(),
	// 		KBytesSent:  (newBytesSent - oldBytesSent) / 1024,
	// 		KBytesRecv:  (newBytesRecv - oldBytesRecv) / 1024,
	// 		PacketsSent: newPacketsSent - oldPacketsSent,
	// 		PacketsRecv: newPacketsRecv - oldPacketsRecv,
	// 	}

	// 	output.ProcessOutput(netspeedData)

	// 	oldBytesSent = newBytesSent
	// 	oldBytesRecv = newBytesRecv
	// 	oldPacketsSent = newPacketsSent
	// 	oldPacketsRecv = newPacketsRecv
	// 	time.Sleep(1 * time.Second)
	// }
}
