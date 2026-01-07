package network

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const sysfsNet string = "/sys/class/net"

func collectNetInterfaces() ([]NetInterface, error) {
	dirs, err := os.ReadDir(sysfsNet)
	if err != nil {
		return nil, fmt.Errorf("read directory %s failed: %w", sysfsNet, err)
	}

	netInterfaces := make([]NetInterface, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		dirName := dir.Name()
		if strings.HasPrefix(dirName, "lo") || strings.HasPrefix(dirName, "loop") {
			continue
		}

		netInterfaces = append(netInterfaces, collectNetInterface(dirName))
	}

	return netInterfaces, nil
}

func collectNetInterface(name string) NetInterface {
	res := NetInterface{
		DeviceName: name,
	}

	feildMap := map[string]*string{
		"address":   &res.MACAddress,
		"mtu":       &res.MTU,
		"duplex":    &res.Duplex,
		"speed":     &res.Speed,
		"operstate": &res.Status,
	}

	for name, ptr := range feildMap {
		filePath := filepath.Join(sysfsNet, name)
		if content, err := utils.ReadOneLineFile(filePath); err == nil {
			*ptr = content
		}
	}

	go res.collectEthtoolDriver(name)
	go res.collectEthtoolSetting(name)

	if strings.ToLower(res.Status) == "up" {
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
	res := make([]IPv4Address, len(addrs))
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

	gateway := make(net.IP, len(ip.To4()))

	for i := 0; i < 4; i++ {
		gateway[i] = ip[i] & mask[i]
	}

	gateway[3]++

	return gateway.String()
}
