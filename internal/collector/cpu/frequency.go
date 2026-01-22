package cpu

import (
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

type coreFrequency struct {
	minFreq    int
	maxFreq    int
	basedFreq  int
	watt       float64
	powerState string
	threadMap  map[string][]*ThreadEntry
}

func collectFrequency(ctx context.Context, vendor string) (*coreFrequency, error) {
	c := &coreFrequency{
		powerState: powerStatePowerSaving,
		threadMap:  make(map[string][]*ThreadEntry),
	}

	tempMap, err := collectTemperature(vendor)
	if err != nil {
		return nil, err
	}

	if err := c.turbostat(ctx, vendor, tempMap); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *coreFrequency) turbostat(ctx context.Context, vendor string, tempMap map[string]int) error {
	output := execute.CommandWithContext(ctx, turbostat, "-q", "sleep", "5")
	if output.AsError() != nil {
		return output.Err
	}

	lines := strings.Split(string(output.Stderr), "\n")

	if len(lines) < 3 {
		return errors.New("turbostat output is too short")
	}
	headers := strings.Fields(lines[1])
	headerIndex := make(map[string]int)

	for i, header := range headers {
		headerIndex[header] = i
	}

	summary := strings.Fields(lines[2])
	if len(summary) != len(headers) {
		return errors.New("turbostat summary line does not match headers")
	}

	c.basedFreq = getIntValue(tscMHz, summary, headerIndex)
	c.minFreq = getIntValue(bzyMHz, summary, headerIndex)
	c.maxFreq = getIntValue(bzyMHz, summary, headerIndex)
	c.watt = getFloatValue(pkgWatt, summary, headerIndex)

	for _, line := range lines[3:] {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		pkgVal := fields[headerIndex[pkg]]
		coreVal := fields[headerIndex[core]]
		threadVal := fields[headerIndex[cpu]]
		coreFreq := getIntValue(bzyMHz, fields, headerIndex)

		if coreFreq > c.maxFreq {
			c.maxFreq = coreFreq
		}

		if coreFreq < c.minFreq {
			c.minFreq = coreFreq
		}

		thread := &ThreadEntry{
			PhysicalID:    pkgVal,
			CoreID:        coreVal,
			ProcessorID:   threadVal,
			CoreFrequency: formatMHz(coreFreq),
		}

		parseCoreTemperature(thread, tempMap, vendor)

		c.threadMap[pkgVal] = append(c.threadMap[pkgVal], thread)
	}

	if c.minFreq-50 > c.basedFreq {
		c.powerState = powerStatePerformance
	}

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
