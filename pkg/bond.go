package pkg

import (
	"fmt"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
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
	println("[BOND INFO]")
	if len(c.Network.BondInterfaces) == 0 {
		return
	}
	fields := []string{"Name", "Status", "Speed", "AggregatorID", "Diagnose", "DiagnoseDetail", "SlaveInterfaces"}
	var sb strings.Builder
	for _, iface := range c.Network.BondInterfaces {
		sb.WriteString(
			utils.SelectFields(iface, fields, 1).String() + "\n")
	}

	fmt.Println(sb.String())
}

func (c *Bond) PrintDetail() {}
