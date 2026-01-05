package main

import (
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/pci"
)

func main() {
	str, err := pci.GetSerialRAIDPCIBus()
	if err != nil {
		panic(err)
	}

	for _, v := range str {
		pciInfo := pci.New(v)
		if err := pciInfo.Collect(); err != nil {
			panic(err)
		}

		fmt.Println(pciInfo)
	}
}
