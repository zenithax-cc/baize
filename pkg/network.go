package pkg

import (
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/network"
)

type Network struct {
	*network.Network
}

func NewNetwork() *Network {
	return &Network{
		Network: network.New(),
	}
}

func (c *Network) PrintJson() {
	printJson("Network", c.Network)
}

func (c *Network) PrintBrief() {
	println("[NETWORK INFO]")
	nicMap := map[string]int{}
	for _, iface := range c.Network.PhysicalInterfaces {
		name := fmt.Sprintf("%s %s", iface.PCIe.Device, iface.Speed)
		nicMap[name]++
	}
	for name, count := range nicMap {
		fmt.Printf("%s%s * %d\n", "    ", name, count)
	}
}

func (c *Network) PrintDetail() {}
