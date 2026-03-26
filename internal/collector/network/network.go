// Package network provides functionality for collecting network interface information,
// including logical net interfaces (IP, MAC, speed), physical NIC hardware details
// (ring buffer, channel, LLDP, PCI), and bond interface configuration.
package network

import (
	"context"

	"github.com/zenithax-cc/baize/pkg/utils"
)

// New creates and returns a new Network instance with pre-allocated slices for
// physical interfaces, bond interfaces, and logical network interfaces.
func New() *Network {
	return &Network{
		PhyInterfaces:  make([]PhyInterface, 0, 8),
		BondInterfaces: make([]BondInterface, 0, 2),
		NetInterfaces:  make([]NetInterface, 0, 16),
	}
}

// Collect gathers network information from three independent sources:
//   - Physical NIC hardware data (via sysfs and ethtool/LLDP)
//   - Logical network interface data (via netlink/sysfs)
//   - Bond interface configuration (via /proc/net/bonding)
//
// Errors from individual sub-collectors are combined and returned together.
func (n *Network) Collect(ctx context.Context) error {
	var errs []error

	// Collect physical NIC details (PCI info, ring buffer, channels, LLDP).
	phys, err := collectNic()
	if err != nil {
		errs = append(errs, err)
	}
	n.PhyInterfaces = phys

	// Collect logical interface details (IP addresses, MAC, driver, speed).
	nets, err := CollectNetInterfaces()
	if err != nil {
		errs = append(errs, err)
	}
	n.NetInterfaces = nets

	// Collect bond interface configurations and slave interface states.
	bonds, err := collectBondInterfaces()
	if err != nil {
		errs = append(errs, err)
	}
	n.BondInterfaces = bonds

	return utils.CombineErrors(errs)
}

// Name returns the collector identifier used for module routing.
func (n *Network) Name() string {
	return "network"
}

// JSON serializes the Network struct to JSON and writes it to stdout.
func (n *Network) JSON() error {
	return utils.JSONPrintln(n)
}

// DetailPrintln prints full network interface details to stdout.
func (n *Network) DetailPrintln() {
	n.printInterfaces("detail")
}

// BriefPrintln prints a concise network interface summary to stdout.
func (n *Network) BriefPrintln() {
	n.printInterfaces("brief")
}

// printInterfaces renders network interface data for the given output type.
func (n *Network) printInterfaces(outputType string) {
	// Build brief view structs for each NetInterface.
	type NICBrief struct {
		Name   string `name:"Interface" output:"both" color:"DefaultGreen"`
		MAC    string `name:"MAC Address" output:"both"`
		Driver string `name:"Driver" output:"both"`
		Speed  string `name:"Speed" output:"both"`
		Duplex string `name:"Duplex" output:"detail"`
		MTU    string `name:"MTU" output:"detail"`
		Link   string `name:"Link Detected" output:"both" color:"trueGreen"`
		Status string `name:"Status" output:"both"`
		IPv4   string `name:"IPv4" output:"both" color:"DefaultGreen"`
	}

	briefs := make([]*NICBrief, 0, len(n.NetInterfaces))
	for i := range n.NetInterfaces {
		ni := &n.NetInterfaces[i]
		b := &NICBrief{
			Name:   ni.DeviceName,
			MAC:    ni.MACAddress,
			Driver: ni.Driver,
			Speed:  ni.Speed,
			Duplex: ni.Duplex,
			MTU:    ni.MTU,
			Link:   ni.LinkDetected,
			Status: ni.Status,
		}
		if len(ni.IPv4) > 0 {
			b.IPv4 = ni.IPv4[0].Address
			if ni.IPv4[0].PrefixLen != "" {
				b.IPv4 += "/" + ni.IPv4[0].PrefixLen
			}
		}
		briefs = append(briefs, b)
	}

	wrapper := struct {
		NICs []*NICBrief `name:"NETWORK INFO" output:"both"`
	}{
		NICs: briefs,
	}

	utils.PrinterInstance.Print(wrapper, "NETWORK")
}
