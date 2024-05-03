package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gdanko/netspeed/globals"
	"github.com/gdanko/netspeed/util"
)

func ProcessOutput(netspeedData globals.NetspeedData) {
	jsonBytes, err := json.Marshal(netspeedData)
	if err != nil {
		util.ExitOnError(err.Error())
	}

	if globals.GetOutputFile() == "" {
		fmt.Println(string(jsonBytes))
	} else {
		err = os.WriteFile(globals.GetOutputFile(), jsonBytes, 0644)
		if err != nil {
			util.ExitOnError(err.Error())
		}
	}
}

func ShowAvailableInterfaces() {
	fmt.Println("Available Interfaces:")
	for _, interfaceName := range globals.GetInterfaceList() {
		fmt.Printf("  %s\n", interfaceName)
	}
	util.ExitCleanly()
}
