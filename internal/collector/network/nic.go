// Package network - nic.go collects physical NIC hardware information
// (PCI metadata, LLDP neighbors, ring buffer, and channel settings)
// for each network NIC found on the PCI bus.
package network

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/zenithax-cc/baize/internal/collector/pci"
)

// collectNic discovers all network PCI devices and concurrently collects
// physical interface details (PCI info, LLDP, ring buffer, channels) for each.
func collectNic() ([]PhyInterface, error) {
	nics, err := pci.GetNetworkPCIBus()
	if err != nil {
		return nil, err
	}

	if len(nics) == 0 {
		return nil, nil
	}

	results := make([]PhyInterface, len(nics))
	errsCh := make(chan error, len(nics))

	var wg sync.WaitGroup
	for i, nic := range nics {
		wg.Add(1)
		go func(idx int, addr string) {
			defer wg.Done()

			itf := PhyInterface{
				DeviceName: nicName(addr),
			}

			// Collect PCI device metadata (vendor, device ID, subsystem, etc.).
			pcie := pci.New(addr)
			if err := pcie.Collect(); err != nil {
				errsCh <- err
			} else {
				itf.PCI = *pcie
			}

			// Only collect ethtool/LLDP data when the device name is resolved.
			if itf.DeviceName != "unknown" {
				var innerWg sync.WaitGroup
				innerWg.Add(3)

				go func() {
					defer innerWg.Done()
					lldp, err := lldpNeighbors(itf.DeviceName)
					if err != nil {
						errsCh <- err
					} else {
						itf.LLDP = lldp
					}
				}()

				go func() {
					defer innerWg.Done()
					itf.RingBuffer = collectEthtoolRingBuffer(itf.DeviceName)
				}()

				go func() {
					defer innerWg.Done()
					itf.Channel = collectEthtoolChannel(itf.DeviceName)
				}()

				innerWg.Wait()
			}

			results[idx] = itf
		}(i, nic)
	}

	wg.Wait()
	close(errsCh)

	var errs []error
	for e := range errsCh {
		errs = append(errs, e)
	}

	return results, combineErrs(errs)
}

const sysfsBus string = "/sys/bus/pci/devices"

// nicName resolves the kernel network interface name from a PCI device address
// by reading /sys/bus/pci/devices/<addr>/net/ directory.
// Returns "unknown" if the name cannot be determined.
func nicName(addr string) string {
	dirs, err := os.ReadDir(filepath.Join(sysfsBus, addr, "net"))
	if err != nil || len(dirs) == 0 {
		return "unknown"
	}

	return dirs[0].Name()
}

// combineErrs joins a slice of errors into a single error, returning nil if empty.
func combineErrs(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	combined := errs[0]
	for _, e := range errs[1:] {
		combined = fmt.Errorf("%w; %v", combined, e)
	}
	return combined
}
