package cpu

import (
	"context"
	"errors"
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
	minFreq     int
	maxFreq     int
	basedFreq   int
	powerState  string
	coreFreqMap map[string]int
}

func collectThreadSummary(ctx context.Context) (*coreFrequency, error) {
	c := &coreFrequency{
		powerState: powerStatePowerSaving,
	}

	if err := c.turbostat(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *coreFrequency) turbostat(ctx context.Context) error {
	output := execute.CommandWithContext(ctx, turbostat, "-q", "sleep", "5")
	if output.AsError() != nil {
		return output.Err
	}

	lines := strings.Split(string(output.Stdout), "\n")
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

	for _, line := range lines[3:] {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		pkgVal := fields[headerIndex[pkg]]
		core := fields[headerIndex[pkg]]
		cpu := fields[headerIndex[pkg]]
		coreFreq := getIntValue(bzyMHz, fields, headerIndex)

		if coreFreq > c.maxFreq {
			c.maxFreq = coreFreq
		}

		if coreFreq < c.minFreq {
			c.minFreq = coreFreq
		}

		key := pkgVal + "-" + core + "-" + cpu
		c.coreFreqMap[key] = coreFreq
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
