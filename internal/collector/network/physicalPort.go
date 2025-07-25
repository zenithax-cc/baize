package network

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/pci"
	"golang.org/x/sync/errgroup"
)

type physicalInterface struct {
	networkInterface
	State            string        `json:"state,omitempty"`
	IsBondSlave      bool          `json:"is_bond_slave,omitempty"`
	LinkFailureCount string        `json:"link_failure_count,omitempty"`
	AggregatorID     string        `json:"aggregator_id,omitempty"`
	PCIe             pci.PCIe      `json:"pcie,omitempty"`
	LLDP             lldp          `json:"lldp,omitempty"`
	EthChannel       ethChannel    `json:"channel,omitempty"`
	EthRingBuffer    ethRingBuffer `json:"ring_buffer,omitempty"`
}

type lldp struct {
	RemoteID         string `json:"remote_id,omitempty"`
	TORMACAddress    string `json:"tor_mac_address,omitempty"`
	TORName          string `json:"tor_name,omitempty"`
	InterfaceName    string `json:"interface_name,omitempty"`
	ManagementIP     string `json:"management_ip,omitempty"`
	TTL              string `json:"ttl,omitempty"`
	MaxFrameSize     string `json:"max_frame_size,omitempty"`
	Vlan             string `json:"vlan,omitempty"`
	PortProtocolVlan string `json:"port_protocol_vlan,omitempty"`
}

func (n *Network) collectPhysicalInterfaces(ctx context.Context) error {
	bus, err := pci.GetNetworkControllerPCIBus()
	if err != nil {
		return fmt.Errorf("get eth pci bus failed: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 6)

	for _, addr := range bus {
		pciBus := addr
		eg.Go(
			func() error {
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					return ctx.Err()
				}
				return n.parsePhysicalInterface(ctx, pciBus)
			})
	}
	return eg.Wait()
}

func (n *Network) parsePhysicalInterface(ctx context.Context, bus string) error {
	var errs []error
	res := physicalInterface{}
	p := pci.New(bus)
	if err := p.Collect(); err != nil {
		errs = append(errs, err)
	}
	res.PCIe = *p

	net, err := findPhysicalInterfaceName(bus)
	if err != nil {
		errs = append(errs, err)
	}

	if net == "" {
		return nil
	}

	res.Name = net

	basicInfo, err := n.getBaiscInfo(ctx, net)
	if err != nil {
		errs = append(errs, err)
	}
	res.networkInterface = basicInfo

	bondSlaveDir := filepath.Join(sysNet, net, "bonding_slave")
	if utils.DirExists(bondSlaveDir) {
		if err := parseBondSlave(bondSlaveDir, &res); err != nil {
			errs = append(errs, err)
		}
	}

	if res.Status == "up" {
		if res.ethDriver.DriverName == "i40e" {
			optimizeI40e(net)
		}

		lldpInfo, err := getLLDP(net)
		if err != nil {
			errs = append(errs, err)
		}
		res.LLDP = lldpInfo
	}

	if ec, err := getEthChannel(net); err == nil {
		res.EthChannel = *ec
	} else {
		errs = append(errs, err)
	}

	if rb, err := getEthRingBuffer(net); err == nil {
		res.EthRingBuffer = *rb
	} else {
		errs = append(errs, err)
	}

	n.addToIfCache(net, res)

	n.mutex.Lock()
	n.PhysicalInterfaces = append(n.PhysicalInterfaces, res)
	n.mutex.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("error collect physical interface: %v", errs)
	}

	return nil
}

func findPhysicalInterfaceName(bus string) (string, error) {
	netPath := filepath.Join(busDevice, bus, "net")
	dirEntries, err := utils.ReadDir(netPath)
	if err != nil {
		return "", err
	}
	if len(dirEntries) > 0 {
		return dirEntries[0].Name(), nil
	}
	return "", fmt.Errorf("no physical interface name found")
}

func parseBondSlave(net string, res *physicalInterface) error {
	fieldMap := map[string]*string{
		"link_failure_count": &res.LinkFailureCount,
		"ad_aggregator_id":   &res.AggregatorID,
		"mii_status":         &res.State,
	}

	var errs []error

	for k, ptr := range fieldMap {
		inf, err := utils.ReadOneLineFile(filepath.Join(net, k))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		*ptr = inf
	}

	return utils.CombineErrors(errs)
}

func getLLDP(port string) (lldp, error) {
	res := lldp{}
	byteLLDP, err := utils.Run.Command("lldpctl", port, "-f", "keyvalue")
	if err != nil {
		return res, err
	}
	prefix := fmt.Sprintf("lldp.%s.", port)
	prefixLen := len(prefix)

	fieldMap := map[string]func(string){
		"rid":             func(v string) { res.RemoteID = v },
		"chassis.mac":     func(v string) { res.TORMACAddress = v },
		"chassis.name":    func(v string) { res.TORName = v },
		"chassis.mgmt-ip": func(v string) { res.ManagementIP = v },
		"port.ifname":     func(v string) { res.InterfaceName = v },
		"port.ttl":        func(v string) { res.TTL = v },
		"port.mfs":        func(v string) { res.MaxFrameSize = v },
		"vlan.vlan-id":    func(v string) { res.Vlan += v },
		"vlan.pvid":       func(v string) { res.Vlan += fmt.Sprintf(" pvid:%s", v) },
		"ppvid.supported": func(v string) { res.PortProtocolVlan += fmt.Sprintf("%s supported", v) },
		"ppvid.enabled":   func(v string) { res.PortProtocolVlan += fmt.Sprintf(" %s enabled", v) },
	}

	scanner := bufio.NewScanner(bytes.NewReader(byteLLDP))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, prefix) {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)[prefixLen:]
		value = strings.TrimSpace(value)

		if handler, ok := fieldMap[key]; ok {
			handler(value)
		}
	}
	return res, nil
}
