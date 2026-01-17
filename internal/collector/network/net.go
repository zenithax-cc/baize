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

var skipTarget = []string{"lo", "loop", "bonding_master"}

func CollectNetInterfaces() ([]NetInterface, error) {
	dirs, err := os.ReadDir(sysfsNet)
	if err != nil {
		return nil, fmt.Errorf("read directory %s failed: %w", sysfsNet, err)
	}

	netInterfaces := make([]NetInterface, 0, len(dirs))
	for _, dir := range dirs {
		itfName := dir.Name()
		if utils.HasPrefix(itfName, skipTarget) {
			continue
		}

		netInterfaces = append(netInterfaces, collectNetInterface(itfName))
	}

	return netInterfaces, nil
}

func collectNetInterface(name string) NetInterface {
	res := NetInterface{
		DeviceName: name,
	}

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

	bondSlave := filepath.Join(sysfsNet, name, "bonding_slave")
	if strings.ToLower(res.Status) == "up" && !utils.PathExists(bondSlave) {
		res.IPv4, _ = getIPv4(name)
	}

	return res
}

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

func calNetmask(mask net.IPMask) string {
	if len(mask) != 4 {
		return ""
	}

	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

func calGateway(ip net.IP, mask net.IPMask) string {
	if len(mask) != 4 || ip.To4() == nil {
		return ""
	}

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
