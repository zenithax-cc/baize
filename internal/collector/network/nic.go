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

	phyInterfaces := make([]PhyInterface, 0, len(nics))
	var errs []error
	for _, nic := range nics {
		itf := PhyInterface{
			DeviceName: nicName(nic),
		}

		pcie := pci.New(nic)
		if err := pcie.Collect(); err != nil {
			errs = append(errs, err)
		}

		if itf.DeviceName != "unknown" {
			lldp, err := lldpNeighbors(itf.DeviceName)
			if err != nil {
				errs = append(errs, err)
			}
			itf.LLDP = lldp

			itf.RingBuffer = collectEthtoolRingBuffer(itf.DeviceName)
			itf.Channel = collectEthtoolChannel(itf.DeviceName)
		}

		phyInterfaces = append(phyInterfaces, itf)
	}

	return phyInterfaces, nil
}

const sysfsBus string = "/sys/bus/pci/devices"

func nicName(addr string) string {
	dirs, err := os.ReadDir(filepath.Join(sysfsBus, addr, "net"))
	if err != nil || len(dirs) == 0 {
		return "unkown"
	}

	return dirs[0].Name()
}
