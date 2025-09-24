package pkg

import (
	"fmt"
	"strings"

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

func (c *Network) PrintJSON() {
	printJson("Network", c.Network)
}

func (c *Network) PrintBrief() {
	var sb strings.Builder
	sb.Grow(1000)
	sb.WriteString("[NETWORK INFO]\n")

	if c == nil || c.Network.PhysicalInterfaces == nil || len(c.Network.PhysicalInterfaces) == 0 {
		sb.WriteString("	no physical network interface found\n")
		println(sb.String())
		return
	}

	nicMap := map[string]int{}
	for _, iface := range c.Network.PhysicalInterfaces {
		nicMap[iface.PCIe.Device]++
	}

	for name, count := range nicMap {
		sb.WriteString(printSeparator(name, fmt.Sprintf("* %d", count), false, 1))
	}

	println(sb.String())
}

func (c *Network) PrintDetail() {}

func (c *Network) Name() string {
	return "Network"
}
