package pci

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

// DeviceClasses represents the type of PCI device class.
type DeviceClasses int

const (
	ClassRAID    DeviceClasses = 1
	ClassNvme    DeviceClasses = 2
	ClassNetwork DeviceClasses = 3
	ClassDisplay DeviceClasses = 4
)

func (classes DeviceClasses) String() string {
	switch classes {
	case ClassRAID:
		return "RAID Controller"
	case ClassNvme:
		return "NVMe Controller"
	case ClassNetwork:
		return "Network Controller"
	case ClassDisplay:
		return "Display Controller"
	default:
		return fmt.Sprintf("Unknown(%d)", classes)
	}
}

// devicePatterns maps device types to PCI class code patterns.
// Reference: https://pci-ids.ucw.cz/read/PD
var devicePatterns = map[DeviceClasses]*regexp.Regexp{
	ClassRAID:    regexp.MustCompile(`\[010[47]\]`),  // RAID & SAS
	ClassNvme:    regexp.MustCompile(`\[0108\]`),     // NVMe
	ClassNetwork: regexp.MustCompile(`\[020[0-8]\]`), // Ethernet, etc.
	ClassDisplay: regexp.MustCompile(`\[030[0-2]\]`), // VGA,XGA,3D
}

// getPCIBus returns the PCI bus addresses of devices matching the given type.
func getPCIBus(class DeviceClasses) ([]string, error) {
	pattern, ok := devicePatterns[class]
	if !ok {
		return nil, fmt.Errorf("unsupported device type: %s", class)
	}

	res := execute.Command("lspci", "-Dnn")

	if res.Err != nil {
		return nil, fmt.Errorf("lspci failed: %w", res.Err)
	}

	return filterPCIBus(res.Stdout, pattern)
}

// filterDevices filters lspci output by the given pattern.
func filterPCIBus(data []byte, pattern *regexp.Regexp) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var res []string
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if pattern.MatchString(line) {
			if busAddr := extractBusAddress(line); busAddr != "" {
				res = append(res, busAddr)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("error scanning lspci output: %w", err)
	}

	return res, nil
}

// extractBusAddress extracts the PCI bus address from an lspci output line.
// Example input: "0000:00:1f.2 SATA controller [0106]: Intel..."
// Example output: "0000:00:1f.2"
func extractBusAddress(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	if idx := strings.IndexByte(line, ' '); idx > 0 {
		return line[:idx]
	}

	return line
}

// GetSerialRAIDPCIBus returns the PCI bus addresses of RAID and SAS controllers.
func GetSerialRAIDPCIBus() ([]string, error) {
	return getPCIBus(ClassRAID)
}

// GetNVMePCIBus returns the PCI bus addresses of NVMe controllers.
func GetNVMePCIBus() ([]string, error) {
	return getPCIBus(ClassNvme)
}

// GetNetworkPCIBus returns the PCI bus addresses of network controllers.
func GetNetworkPCIBus() ([]string, error) {
	return getPCIBus(ClassNetwork)
}

// GetDisplayPCIBus returns the PCI bus addresses of display controllers.
func GetDisplayPCIBus() ([]string, error) {
	return getPCIBus(ClassDisplay)
}
