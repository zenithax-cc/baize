// Package ipmi collects Baseboard Management Controller (BMC) information
// by parsing the output of `ipmitool bmc info` and `ipmitool lan print`.
package ipmi

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const ipmitool = "ipmitool"

// collectBMC populates the BMC struct with device info and LAN configuration.
// It runs two ipmitool commands concurrently:
//   - `ipmitool bmc info`  → device ID, firmware version, IPMI spec version
//   - `ipmitool lan print` → management IP, MAC address, gateway, subnet
func (m *IPMI) collectBMC(ctx context.Context) error {
	type kv = map[string]*string

	// --- bmc info ---
	infoOut := execute.CommandWithContext(ctx, ipmitool, "bmc", "info")
	if infoOut.Err == nil {
		fields := kv{
			"Device ID":         &m.BMC.DeviceID,
			"Device Revision":   &m.BMC.DeviceRevision,
			"Firmware Revision": &m.BMC.FirmwareRevision,
			"IPMI Version":      &m.BMC.IPMIVersion,
			"Manufacturer ID":   &m.BMC.ManufacturerID,
			"Product ID":        &m.BMC.ProductID,
		}
		parseIPMIKeyValue(infoOut.Stdout, ":", fields)
	}

	// --- lan print (channel 1) ---
	lanOut := execute.CommandWithContext(ctx, ipmitool, "lan", "print", "1")
	if lanOut.Err != nil {
		// Some servers expose the BMC on channel 2; fall back silently.
		lanOut = execute.CommandWithContext(ctx, ipmitool, "lan", "print")
	}
	if lanOut.Err == nil {
		fields := kv{
			"IP Address              ": &m.BMC.ManagementIP,
			"IP Address":               &m.BMC.ManagementIP,
			"MAC Address":              &m.BMC.MACAddress,
			"Subnet Mask":              &m.BMC.Subnet,
			"Default Gateway IP":       &m.BMC.Gateway,
		}
		parseIPMIKeyValue(lanOut.Stdout, ":", fields)
	}

	return nil
}

// parseIPMIKeyValue scans ipmitool line-oriented output and fills in the
// destination string pointers for each matching key.
// sep is the separator (typically ":"). Only the first match per key is used.
func parseIPMIKeyValue(data []byte, sep string, fields map[string]*string) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		k, v, ok := strings.Cut(line, sep)
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if val == "" {
			continue
		}
		if ptr, exists := fields[key]; exists && *ptr == "" {
			*ptr = val
		}
	}
}
