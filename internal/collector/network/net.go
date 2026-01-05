package network

import (
	"fmt"
	"os"
	"strings"
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
	netInterface := NetInterface{
		DeviceName: name,
	}

	return netInterface
}
