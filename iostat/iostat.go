package iostat

import (
	"sort"

	"github.com/gdanko/netspeed/globals"
	"github.com/shirou/gopsutil/net"
)

func GetInterfaceList() (interfaceList []string, err error) {
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

func GetData() (output []globals.IOStatsData, err error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return []globals.IOStatsData{}, err
	}

	for _, ifaceBlock := range ioCounters {
		output = append(output, globals.IOStatsData{
			Interface:   ifaceBlock.Name,
			BytesSent:   float64(ifaceBlock.BytesSent),
			BytesRecv:   float64(ifaceBlock.BytesRecv),
			PacketsSent: uint64(ifaceBlock.PacketsSent),
			PacketsRecv: uint64(ifaceBlock.PacketsRecv),
		})
	}
	return output, nil
}
