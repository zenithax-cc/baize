// Package ipmi - sensor.go collects IPMI sensor readings via `ipmitool sensor`
// and classifies them by type (temperature, voltage, fan, current, other).
package ipmi

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

// sensorType maps ipmitool sensor type strings (lowercased) to category labels.
// ipmitool sensor list columns: Name | Value | Unit | Status | LNR | LCR | LNC | UNC | UCR | UNR
var sensorTypeKeywords = map[string]string{
	"degrees": "temperature",
	"volts":   "voltage",
	"rpm":     "fan",
	"amps":    "current",
	"watts":   "current", // power sensors grouped with current
}

// collectSensors runs `ipmitool sensor` and populates the Sensors struct.
// Sensors are bucketed into Temperature, Voltage, Fan, Current, and Other.
// Sensors with a "na" or "ns" (not available / not supported) status are skipped.
func (m *IPMI) collectSensors(ctx context.Context) error {
	out := execute.CommandWithContext(ctx, ipmitool, "sensor")
	if out.Err != nil {
		return out.Err
	}

	m.Sensors = &Sensors{
		Temperature: make([]*Sensor, 0, 16),
		Voltage:     make([]*Sensor, 0, 8),
		Fan:         make([]*Sensor, 0, 8),
		Current:     make([]*Sensor, 0, 4),
		Other:       make([]*Sensor, 0, 4),
	}

	scanner := bufio.NewScanner(bytes.NewReader(out.Stdout))
	for scanner.Scan() {
		line := scanner.Text()
		// ipmitool sensor output is pipe-delimited.
		// Format: name | value | unit | status | LNR | LCR | LNC | UNC | UCR | UNR
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		unit := strings.TrimSpace(parts[2])
		status := strings.TrimSpace(parts[3])

		// Skip sensors that are not available or not supported.
		if status == "na" || status == "ns" || value == "na" {
			continue
		}

		// Build the display value with its unit.
		displayValue := value
		if unit != "" && unit != "discrete" {
			displayValue = value + " " + unit
		}

		s := &Sensor{
			Name:   name,
			Value:  displayValue,
			Status: status,
		}

		// Extract threshold columns if present.
		if len(parts) >= 9 {
			s.LowerCritical = strings.TrimSpace(parts[6])
			s.UpperCritical = strings.TrimSpace(parts[8])
			// Normalise "na" thresholds to empty string.
			if s.LowerCritical == "na" {
				s.LowerCritical = ""
			}
			if s.UpperCritical == "na" {
				s.UpperCritical = ""
			}
		}

		// Classify the sensor by its unit keyword.
		unitLower := strings.ToLower(unit)
		category := "other"
		for kw, cat := range sensorTypeKeywords {
			if strings.Contains(unitLower, kw) {
				category = cat
				break
			}
		}

		switch category {
		case "temperature":
			m.Sensors.Temperature = append(m.Sensors.Temperature, s)
		case "voltage":
			m.Sensors.Voltage = append(m.Sensors.Voltage, s)
		case "fan":
			m.Sensors.Fan = append(m.Sensors.Fan, s)
		case "current":
			m.Sensors.Current = append(m.Sensors.Current, s)
		default:
			m.Sensors.Other = append(m.Sensors.Other, s)
		}
	}

	return scanner.Err()
}
