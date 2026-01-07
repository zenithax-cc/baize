package network

import (
	"os"
	"path/filepath"

	"github.com/zenithax-cc/baize/internal/collector/pci"
)

func collectNic() ([]PhyInterface, error) {
	nics, err := pci.GetNetworkPCIBus()
	if err != nil {
		return nil, err
	}

	phyInterfaces := make([]PhyInterface, len(nics))

	for _, nic := range nics {
		phyInterfaces = append(phyInterfaces, PhyInterface{
			DeviceName: nicName(nic),
			PCI:        *pci.New(nic),
		})
	}

	return phyInterfaces, nil
}

const sysfsBus string = "/sys/bus/pci/devices"

func nicName(addr string) string {
	dirs, err := os.ReadDir(filepath.Join(sysfsBus, addr, "net"))
	if err != nil {
		return "unkown"
	}

	return dirs[0].Name()
}
