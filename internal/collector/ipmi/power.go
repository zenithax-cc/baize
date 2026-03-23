// Package ipmi - power.go collects power supply unit (PSU) information via
// `ipmitool dcmi power reading` and `ipmitool sdr type "Power Supply"`.
package ipmi

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

// collectPowerSupplies enumerates PSUs via `ipmitool sdr type "Power Supply"`
// and attempts to enrich each entry with power readings.
//
// Example sdr output line:
//
//	PS1 Status       | 58h | ok  | 10.1 | Presence Detected
func (m *IPMI) collectPowerSupplies(ctx context.Context) error {
	out := execute.CommandWithContext(ctx, ipmitool, "sdr", "type", "Power Supply")
	if out.Err != nil {
		return out.Err
	}

	psus := parsePSUSDR(out.Stdout)
	if len(psus) == 0 {
		return nil
	}

	// Enrich with DCMI power readings if available.
	enrichPSUWithDCMI(ctx, psus)

	m.PowerSupplies = psus
	return nil
}

// parsePSUSDR parses `ipmitool sdr type "Power Supply"` output.
// Each line is pipe-delimited:
//
//	<Sensor Name> | <SDR ID> | <Status> | <Entity> | <Reading/Status Text>
func parsePSUSDR(data []byte) []*PowerSupply {
	psus := make([]*PowerSupply, 0, 4)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		status := strings.TrimSpace(parts[4])

		// Skip "absent" or "not present" PSU slots.
		lower := strings.ToLower(status)
		if strings.Contains(lower, "absent") || strings.Contains(lower, "not present") {
			continue
		}

		psus = append(psus, &PowerSupply{
			Name:   name,
			Status: status,
		})
	}
	return psus
}

// enrichPSUWithDCMI queries `ipmitool dcmi power reading` to obtain system-level
// power consumption and distributes the total across detected PSUs.
// If DCMI is unsupported, the function returns silently without modifying the PSUs.
func enrichPSUWithDCMI(ctx context.Context, psus []*PowerSupply) {
	out := execute.CommandWithContext(ctx, ipmitool, "dcmi", "power", "reading")
	if out.Err != nil || len(out.Stdout) == 0 {
		return
	}

	// Parse "Instantaneous power reading: <N> Watts" from DCMI output.
	var instantWatts string
	scanner := bufio.NewScanner(bytes.NewReader(out.Stdout))
	for scanner.Scan() {
		line := scanner.Text()
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(k), "instantaneous") {
			instantWatts = strings.TrimSpace(v)
			break
		}
	}

	if instantWatts == "" {
		return
	}

	// Annotate each PSU with the system-level instantaneous reading as a
	// best-effort output wattage (actual per-PSU metering requires vendor-specific commands).
	for _, psu := range psus {
		if psu.OutputWatts == "" {
			psu.OutputWatts = instantWatts + " (system total)"
		}
	}
}
