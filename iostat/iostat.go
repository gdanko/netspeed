package iostat

import (
	"sort"

	"github.com/shirou/gopsutil/net"
)

type IOStatData struct {
	Interface   string  `json:"interface"`
	BytesRecv   float64 `json:"bytes_recv"`
	BytesSent   float64 `json:"bytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
}

func GetInterfaceList() (interfaceList []string, err error) {
	interfaces, err := net.IOCounters(true)
	if err != nil {
		return interfaceList, nil
	}
	for _, ifaceBlock := range interfaces {
		interfaceList = append(interfaceList, ifaceBlock.Name)
	}
	sort.Slice(interfaceList, func(i, j int) bool {
		return interfaceList[i] < interfaceList[j]
	})
	return interfaceList, nil
}

func GetData() (output []IOStatData, err error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return []IOStatData{}, err
	}

	for _, ifaceBlock := range ioCounters {
		output = append(output, IOStatData{
			Interface:   ifaceBlock.Name,
			BytesSent:   float64(ifaceBlock.BytesSent),
			BytesRecv:   float64(ifaceBlock.BytesRecv),
			PacketsSent: uint64(ifaceBlock.PacketsSent),
			PacketsRecv: uint64(ifaceBlock.PacketsRecv),
		})
	}
	return output, nil
}
