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

			devName := nicName(addr)

			// Collect PCI device metadata (vendor, device ID, subsystem, etc.).
			var pcie pci.PCI
			pcie_dev := pci.New(addr)
			if err := pcie_dev.Collect(); err != nil {
				errsCh <- err
			} else {
				pcie = *pcie_dev
			}

			// Collect ethtool/LLDP data concurrently; store in local vars to
			// avoid data races on the shared itf struct.
			var lldpData LLDP
			var ringBuf RingBuffer
			var channel Channel

			// Only collect ethtool/LLDP data when the device name is resolved.
			if devName != "unknown" {
				var innerWg sync.WaitGroup
				innerWg.Add(3)

				go func() {
					defer innerWg.Done()
					l, err := lldpNeighbors(devName)
					if err != nil {
						errsCh <- err
					} else {
						lldpData = l
					}
				}()

				go func() {
					defer innerWg.Done()
					ringBuf = collectEthtoolRingBuffer(devName)
				}()

				go func() {
					defer innerWg.Done()
					channel = collectEthtoolChannel(devName)
				}()

				innerWg.Wait()
			}

			results[idx] = PhyInterface{
				DeviceName: devName,
				PCI:        pcie,
				LLDP:       lldpData,
				RingBuffer: ringBuf,
				Channel:    channel,
			}
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
