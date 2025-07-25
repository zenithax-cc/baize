package pkg

import (
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/network"
)

type Bond struct {
	*network.Network
}

func NewBond() *Bond {
	return &Bond{
		Network: network.New(),
	}
}

func (c *Bond) PrintJson() {
	printJson("Bond", c.Network.BondInterfaces)
}

func (c *Bond) PrintBrief() {
	var sb strings.Builder
	sb.Grow(1000)
	sb.WriteString("[BOND INFO]\n")

	if c == nil || c.Network.BondInterfaces == nil || len(c.Network.BondInterfaces) == 0 {
		sb.WriteString("	no bond interface found\n")
		println(sb.String())
		return
	}

	fields := []string{"Name", "Status", "Speed", "AggregatorID", "Diagnose", "DiagnoseDetail", "SlaveInterfaces"}
	for _, iface := range c.Network.BondInterfaces {
		sb.WriteString(selectFields(iface, fields, 1, colorMap["Bond"]).String() + "\n")
	}

	println(sb.String())
}

func (c *Bond) PrintDetail() {}
