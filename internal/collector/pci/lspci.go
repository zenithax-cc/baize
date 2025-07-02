package pci

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

// PCIDeviceType is the type of PCI device.
type PCIDeviceType int

const (
	SerialRAIDController PCIDeviceType = iota
	NVMeController
	NetworkController
	DisplayController
)

// typeMap is a map of device types to regex patterns.
var typeMap = map[PCIDeviceType]string{
	SerialRAIDController: `\[010[47]\]`,
	NVMeController:       `\[0108\]`,
	NetworkController:    `\[020[0-8]\]`,
	DisplayController:    `\[030[0-2]\]`,
}

// getPCIID returns the PCI IDs of devices with the given type.
func getPCIID(id PCIDeviceType) ([]string, error) {
	if _, ok := typeMap[id]; !ok {
		return nil, fmt.Errorf("invalid device type %v", id)
	}
	cmd := fmt.Sprintf("lspci -Dnn | egrep '%s'", typeMap[id])
	pciInfo, err := utils.Run.Command("bash", "-c", cmd)
	if err != nil {
		return nil, err
	}
	return parseLspci(pciInfo)
}

// parseLspci parses the output of lspci and returns a slice of PCI IDs.
func parseLspci(b []byte) ([]string, error) {
	if len(b) == 0 {
		return nil, nil
	}

	lineCount := bytes.Count(b, []byte{'\n'}) + 1

	res := make([]string, 0, lineCount)
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		if id := parseLine(scanner.Text()); id != "" {
			res = append(res, id)
		}
	}

	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("error scanning lspci output: %w", err)
	}

	return res, nil
}

// parseLine parses a line of lspci output and returns the PCI ID.
func parseLine(l string) string {

	if l = strings.TrimSpace(l); l == "" {
		return ""
	}

	if idx := strings.IndexByte(l, ' '); idx > 0 {
		return l[:idx]
	} else if idx == -1 && len(l) > 0 {
		return l
	}

	return ""
}

// GetSerialRAIDControllerPCIID returns the PCI IDs of RAID bus controller and Serial Attached SCSI controller
func GetSerialRAIDControllerPCIBus() ([]string, error) {
	return getPCIID(SerialRAIDController)
}

// GetNVMeControllerPCIID returns the PCI IDs of NVMe controller
func GetNVMeControllerPCIBus() ([]string, error) {
	return getPCIID(NVMeController)
}

// GetNetworkControllerPCIID returns the PCI IDs of network controller
func GetNetworkControllerPCIBus() ([]string, error) {
	return getPCIID(NetworkController)
}

// GetDisplayControllerPCIID returns the PCI IDs of display controller
func GetDisplayControllerPCIBus() ([]string, error) {
	return getPCIID(DisplayController)
}
