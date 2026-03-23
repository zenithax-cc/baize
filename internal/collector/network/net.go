// Package network - net.go collects logical network interface information
// (IP addresses, MAC, speed, driver, ethtool settings) from sysfs and netlink.
package network

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const sysfsNet string = "/sys/class/net"

// skipTarget contains interface name prefixes that should be excluded from collection.
var skipTarget = []string{"lo", "loop", "bonding_master"}

// CollectNetInterfaces discovers all network interfaces under /sys/class/net
// and concurrently collects per-interface details. Interfaces matching skipTarget
// prefixes are excluded.
func CollectNetInterfaces() ([]NetInterface, error) {
	dirs, err := os.ReadDir(sysfsNet)
	if err != nil {
		return nil, fmt.Errorf("read directory %s failed: %w", sysfsNet, err)
	}

	// Pre-filter interface names to avoid spawning goroutines for skipped entries.
	names := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if !utils.HasPrefix(dir.Name(), skipTarget) {
			names = append(names, dir.Name())
		}
	}

	if len(names) == 0 {
		return nil, nil
	}

	results := make([]NetInterface, len(names))
	var wg sync.WaitGroup

	for i, name := range names {
		wg.Add(1)
		go func(idx int, ifName string) {
			defer wg.Done()
			results[idx] = collectNetInterface(ifName)
		}(i, name)
	}

	wg.Wait()
	return results, nil
}

// collectNetInterface collects all available information for a single network
// interface: sysfs attributes, ethtool driver/settings, and IPv4 addresses.
func collectNetInterface(name string) NetInterface {
	res := NetInterface{
		DeviceName: name,
	}

	// Read basic interface attributes from sysfs in a single pass.
	fieldMap := map[string]*string{
		"address":   &res.MACAddress,
		"mtu":       &res.MTU,
		"duplex":    &res.Duplex,
		"speed":     &res.Speed,
		"operstate": &res.Status,
	}

	for f, ptr := range fieldMap {
		filePath := filepath.Join(sysfsNet, name, f)
		if content, err := utils.ReadOneLineFile(filePath); err == nil {
			*ptr = content
		}
	}

	// Fetch ethtool driver info and settings concurrently.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.collectEthtoolDriver(name)
	}()

	go func() {
		defer wg.Done()
		res.collectEthtoolSetting(name)
	}()

	wg.Wait()

	// Collect IPv4 addresses only for up interfaces that are not bond slaves.
	bondSlave := filepath.Join(sysfsNet, name, "bonding_slave")
	if strings.ToLower(res.Status) == "up" && !utils.PathExists(bondSlave) {
		res.IPv4, _ = getIPv4(name)
	}

	return res
}

// getIPv4 returns all IPv4 addresses (with netmask, prefix length, and gateway)
// assigned to the named network interface.
func getIPv4(name string) ([]IPv4Address, error) {
	nf, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}

	addrs, err := nf.Addrs()
	if err != nil {
		return nil, err
	}

	res := make([]IPv4Address, 0, len(addrs))
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.To4() == nil {
			continue
		}
		maskSize, _ := ipNet.Mask.Size()
		res = append(res, IPv4Address{
			Address:   ipNet.IP.String(),
			Netmask:   calNetmask(ipNet.Mask),
			Gateway:   calGateway(ipNet.IP.To4(), ipNet.Mask),
			PrefixLen: fmt.Sprintf("%d", maskSize),
		})
	}

	return res, nil
}

// calNetmask converts a net.IPMask to its dotted-decimal string representation.
func calNetmask(mask net.IPMask) string {
	if len(mask) != 4 {
		return ""
	}

	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// calGateway computes the default gateway for a given IP and subnet mask by
// taking the network address and incrementing the host portion by 1.
// Returns an empty string for invalid inputs or /32 networks.
func calGateway(ip net.IP, mask net.IPMask) string {
	if len(mask) != 4 || ip.To4() == nil {
		return ""
	}

	// For a /32 or all-ones host portion, no gateway is derivable.
	if ip[3]&mask[3] == 255 {
		return ""
	}

	gateway := net.IP{
		ip[0] & mask[0],
		ip[1] & mask[1],
		ip[2] & mask[2],
		(ip[3] & mask[3]) + 1,
	}

	return gateway.String()
}
