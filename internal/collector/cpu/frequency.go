// Package cpu - frequency.go collects per-thread CPU frequency and power metrics
// using the turbostat tool and parses its columnar stderr output.
package cpu

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	turbostat = "/usr/sbin/turbostat"

	pkg     = "Package"
	core    = "Core"
	cpu     = "CPU"
	bzyMHz  = "Bzy_MHz"
	tscMHz  = "TSC_MHz"
	coreTmp = "CoreTmp"
	pkgWatt = "PkgWatt"
)

// collectFromTurbostat runs turbostat with a 1-second sampling interval to collect
// per-thread CPU frequency, package temperature, and power consumption.
// turbostat writes its output to stderr; stdout is discarded.
func (c *CPU) collectFromTurbostat(ctx context.Context) error {
	// Use a 1-second sample instead of 5 seconds to reduce collection latency
	// while still providing a representative frequency snapshot.
	output := execute.CommandWithContext(ctx, turbostat, "-q", "sleep", "1")
	if output.Err != nil {
		return output.Err
	}

	lines := bytes.Split(output.Stderr, []byte("\n"))
	if len(lines) < 3 {
		return errors.New("turbostat output is too short")
	}

	// Line 1 (index 1) is the column header; line 2 (index 2) is the system summary.
	headers := strings.Fields(string(lines[1]))
	if len(headers) == 0 {
		return errors.New("turbostat header line is empty")
	}

	headerIndex := make(map[string]int, len(headers))
	for i, header := range headers {
		headerIndex[header] = i
	}

	summaryLine := strings.Fields(string(lines[2]))
	if len(summaryLine) != len(headers) {
		return errors.New("turbostat summary line does not match headers")
	}

	baseFreq := getIntValue(tscMHz, summaryLine, headerIndex)
	minFreq := getIntValue(bzyMHz, summaryLine, headerIndex)
	maxFreq := minFreq

	// Populate package-level temperature and power from the summary line.
	if idx, ok := headerIndex[coreTmp]; ok && idx < len(summaryLine) {
		c.TemperatureCelsius = summaryLine[idx] + " °C"
	}
	if idx, ok := headerIndex[pkgWatt]; ok && idx < len(summaryLine) {
		c.Watt = summaryLine[idx] + " W"
	}

	// Cache header indices used in the inner loop to avoid repeated map lookups.
	pkgIdx, hasPkg := headerIndex[pkg]
	coreIdx, hasCore := headerIndex[core]
	cpuIdx, hasCPU := headerIndex[cpu]
	bzyIdx, hasBzy := headerIndex[bzyMHz]

	// Pre-allocate thread slice assuming remaining lines are all thread rows.
	c.threads = make([]*ThreadEntry, 0, len(lines)-3)

	for _, line := range lines[3:] {
		parts := strings.Fields(string(line))
		if len(parts) == 0 {
			continue
		}

		var pkgVal, coreVal, threadVal string
		if hasPkg && pkgIdx < len(parts) {
			pkgVal = parts[pkgIdx]
		}
		if hasCore && coreIdx < len(parts) {
			coreVal = parts[coreIdx]
		}
		if hasCPU && cpuIdx < len(parts) {
			threadVal = parts[cpuIdx]
		}

		var coreFreq int
		if hasBzy && bzyIdx < len(parts) {
			coreFreq, _ = strconv.Atoi(parts[bzyIdx])
		}

		if coreFreq > maxFreq {
			maxFreq = coreFreq
		}
		if coreFreq > 0 && coreFreq < minFreq {
			minFreq = coreFreq
		}

		c.threads = append(c.threads, &ThreadEntry{
			PhysicalID:    pkgVal,
			CoreID:        coreVal,
			ProcessorID:   threadVal,
			CoreFrequency: formatMHz(coreFreq),
		})
	}

	// If the minimum busy frequency is notably above the base (TSC) frequency,
	// the CPU is running in performance governor mode.
	if minFreq-50 > baseFreq {
		c.PowerState = powerStatePerformance
	}

	c.MaxFreqMHz = formatMHz(maxFreq)
	c.MinFreqMHz = formatMHz(minFreq)
	c.BasedFreqMHz = formatMHz(baseFreq)

	return nil
}

// getIntValue safely retrieves an integer value from a parsed turbostat line
// using the pre-built header index map. Returns -1 if the key is not found.
func getIntValue(key string, header []string, headerIndex map[string]int) int {
	if index, ok := headerIndex[key]; ok && index < len(header) {
		v, _ := strconv.Atoi(header[index])
		return v
	}
	return -1
}

// getFloatValue safely retrieves a float64 value from a parsed turbostat line
// using the pre-built header index map. Returns -1 if the key is not found.
func getFloatValue(key string, header []string, headerIndex map[string]int) float64 {
	if index, ok := headerIndex[key]; ok && index < len(header) {
		v, _ := strconv.ParseFloat(header[index], 64)
		return v
	}
	return -1
}
