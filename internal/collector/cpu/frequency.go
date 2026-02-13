package cpu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	turbostat = "/usr/sbin/turbostat"

	timeSuffix = "sec"
	pkg        = "Package"
	die        = "Die"
	core       = "Core"
	cpu        = "CPU"
	bzyMHz     = "Bzy_MHz"
	tscMHz     = "TSC_MHz"
	coreTmp    = "CoreTmp"
	pkgTmp     = "PkgTmp"
	pkgWatt    = "PkgWatt"
	corWatt    = "CorWatt"
)

func (c *CPU) collectFromTurbostat(ctx context.Context) error {
	output := execute.CommandWithContext(ctx, turbostat, "-q", "sleep", "5")
	if output.Err != nil {
		return output.Err
	}

	lines := bytes.Split(output.Stderr, []byte("\n"))
	if len(lines) < 3 {
		return errors.New("turbostat output is too short")
	}

	headers := strings.Fields(string(lines[1]))
	headerIndex := make(map[string]int)
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
	c.TemperatureCelsius = fmt.Sprintf("%d °C", getIntValue(coreTmp, summaryLine, headerIndex))
	c.Watt = summaryLine[headerIndex[pkgWatt]] + " W"

	for _, line := range lines[3:] {
		parts := strings.Fields(string(line))
		if len(parts) == 0 {
			continue
		}

		var pkgVal, coreVal, threadVal string
		if pkgIndex, ok := headerIndex[pkg]; ok {
			pkgVal = parts[pkgIndex]
		}

		if coreIndex, ok := headerIndex[core]; ok {
			coreVal = parts[coreIndex]
		}

		if cpuIndex, ok := headerIndex[cpu]; ok {
			threadVal = parts[cpuIndex]
		}

		coreFreq := getIntValue(bzyMHz, parts, headerIndex)

		if coreFreq > maxFreq {
			maxFreq = coreFreq
		}

		if coreFreq < minFreq {
			minFreq = coreFreq
		}

		c.threads = append(c.threads, &ThreadEntry{
			PhysicalID:    pkgVal,
			CoreID:        coreVal,
			ProcessorID:   threadVal,
			CoreFrequency: formatMHz(coreFreq),
		})
	}

	if minFreq-50 > baseFreq {
		c.PowerState = powerStatePerformance
	}

	c.MaxFreqMHz = formatMHz(maxFreq)
	c.MinFreqMHz = formatMHz(minFreq)
	c.BasedFreqMHz = formatMHz(baseFreq)

	return nil
}

func getIntValue(key string, header []string, headerIndex map[string]int) int {
	if index, ok := headerIndex[key]; ok && index < len(header) {
		v, _ := strconv.Atoi(header[index])
		return v
	}

	return -1
}

func getFloatValue(key string, header []string, headerIndex map[string]int) float64 {
	if index, ok := headerIndex[key]; ok && index < len(header) {
		v, _ := strconv.ParseFloat(header[index], 64)
		return v
	}

	return -1
}

func parseCoreTemperature(thread *ThreadEntry, tempMap map[string]int, vendor string) {
	var key string
	if vendor == "Intel" {
		key = fmt.Sprintf("%s-%s", thread.PhysicalID, thread.CoreID)
	} else {
		key = thread.PhysicalID
	}

	if temp, ok := tempMap[key]; ok {
		thread.Temperature = fmt.Sprintf("%d °C", temp)
	}
}
