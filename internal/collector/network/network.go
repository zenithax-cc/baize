package network

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/zenithax-cc/baize/common/utils"
	"golang.org/x/sync/errgroup"
)

const (
	sysNet         = "/sys/class/net"
	busDevice      = "/sys/bus/pci/devices"
	collectTimeout = 10 * time.Second
	maxConcurrency = 4
)

type Network struct {
	PhysicalInterfaces []physicalInterface `json:"physical_interfaces,omitempty"`
	BondInterfaces     []bondInterface     `json:"bond_interfaces,omitempty"`
	OtherInterfaces    []networkInterface  `json:"other_interfaces,omitempty"`

	ifCache map[string]interface{}
	mutex   sync.RWMutex
}

type networkInterface struct {
	Name       string `json:"device_name,omitempty"`
	Status     string `json:"status,omitempty"`
	Speed      string `json:"speed,omitempty"`
	Duplex     string `json:"duplex,omitempty"`
	Mtu        string `json:"mtu,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
	ethDriver
	IPv4 ipv4 `json:"ipv4,omitempty"`
}
type ipv4 struct {
	IPAddress string `json:"ipv4_address,omitempty"`
	Netmask   string `json:"netmask,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
	PrefixLen string `json:"prefix_length,omitempty"`
}

type bondInterface struct {
	networkInterface
	Mode            string              `json:"bond_mode,omitempty"`
	HashPolicy      string              `json:"bond_hash_policy,omitempty"`
	PollingInterval string              `json:"bond_polling_interval,omitempty"`
	LACPRate        string              `json:"bond_lacp_rate,omitempty"`
	LACPActive      string              `json:"bond_lacp_active,omitempty"`
	AggregatorID    string              `json:"aggregator_id,omitempty"`
	NumberOfPorts   string              `json:"number_of_ports,omitempty"`
	Diagnose        string              `json:"diagnose,omitempty"`
	DiagnoseDetail  string              `json:"diagnose_detail,omitempty"`
	SlaveInterfaces []physicalInterface `json:"slave_interfaces,omitempty"`
}

func New() *Network {
	return &Network{
		PhysicalInterfaces: make([]physicalInterface, 0, 6),
		BondInterfaces:     make([]bondInterface, 0, 2),
		OtherInterfaces:    make([]networkInterface, 0, 2),
		ifCache:            make(map[string]interface{}),
	}
}

func (n *Network) Collect(ctx context.Context) error {
	var errs []error

	if err := n.collectPhysicalInterfaces(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := n.collectBondInterfaces(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := n.collectOtherInterfaces(ctx); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("error collect network information: %v", errs)
	}

	return nil
}

// getBaiscInfo collect basic information of a network interface.
func (n *Network) getBaiscInfo(ctx context.Context, port string) (networkInterface, error) {
	select {
	case <-ctx.Done():
		return networkInterface{}, ctx.Err()
	default:
	}

	dirPort := filepath.Join(sysNet, port)
	res := networkInterface{
		Name: port,
	}

	fileMap := map[string]*string{
		"address":   &res.MACAddress,
		"mtu":       &res.Mtu,
		"duplex":    &res.Duplex,
		"speed":     &res.Speed,
		"operstate": &res.Status,
	}

	for fileName, ptr := range fileMap {
		filePath := filepath.Join(dirPort, fileName)
		if content, err := utils.ReadOneLineFile(filePath); err == nil {
			*ptr = content
		}
	}

	if driver, err := getEthDriver(port); err == nil {
		res.ethDriver = *driver
	}

	if res.Status == "up" {
		if ipv4, err := getIPv4(port); err == nil {
			res.IPv4 = ipv4
		}
	}

	return res, nil
}

// collectOtherInterfaces collect other network interface information, excluding physical ports and bond ports.
func (n *Network) collectOtherInterfaces(ctx context.Context) error {
	dirEntries, err := utils.ReadDir(sysNet)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxConcurrency)
	for _, entry := range dirEntries {
		port := entry.Name()
		if n.proccessedName(port) || !entry.IsDir() {
			continue
		}
		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}
			if intf, err := n.getBaiscInfo(ctx, port); err == nil {
				n.mutex.Lock()
				n.ifCache[port] = intf
				n.OtherInterfaces = append(n.OtherInterfaces, intf)
				n.mutex.Unlock()
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("collect other interfaces failed: %w", err)
	}
	return nil
}

// proccessedName check if a network interface has been processed.
func (n *Network) proccessedName(name string) bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	if _, exists := n.ifCache[name]; exists {
		return true
	}
	return false
}

// addToIfCache add a network interface to the cache.
func (n *Network) addToIfCache(name string, intf interface{}) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.ifCache[name] = intf
}

// buildPhysicalMap build a map of physical interfaces.
func (n *Network) buildPhysicalMap() map[string]physicalInterface {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	res := make(map[string]physicalInterface)
	for name, intf := range n.ifCache {
		if phys, ok := intf.(physicalInterface); ok {
			res[name] = phys
		}
	}
	return res
}

// getIPv4 get IPv4 information of a network interface.
func getIPv4(name string) (ipv4, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ipv4{}, fmt.Errorf("failed to get %s info: %w", name, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return ipv4{}, fmt.Errorf("failed to get %s info: %w", name, err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.To4() == nil {
			continue
		}
		maskSize, _ := ipNet.Mask.Size()
		return ipv4{
			IPAddress: ipNet.IP.String(),
			Netmask:   calNetmask(ipNet.Mask),
			Gateway:   calGateway(ipNet.IP.To4(), ipNet.Mask),
			PrefixLen: fmt.Sprintf("%d", maskSize),
		}, nil
	}
	return ipv4{}, fmt.Errorf("no ipv4 address found for %s", name)
}

// calNetmask calculate netmask of an IP address.
func calNetmask(mask net.IPMask) string {
	if len(mask) != 4 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// calGateway calculate gateway of an IP address.
func calGateway(ip net.IP, mask net.IPMask) string {
	if len(mask) != 4 || ip.To4() == nil {
		return ""
	}
	gateway := make(net.IP, len(ip.To4()))
	for i := 0; i < 4; i++ {
		gateway[i] = ip[i] & mask[i]
	}
	gateway[3]++
	return gateway.String()
}
