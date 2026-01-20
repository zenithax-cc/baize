package cpu

import (
	"bufio"
	"bytes"
	"context"
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

type turbostatResult struct {
	maxFreq     int
	minFreq     int
	basedFreq   int
	temperature string
	wattage     string
	powerState  string
	pkgMap      map[string][]*ThreadEntry
}

func collectTurbostat(ctx context.Context) (*turbostatResult, error) {
	output := execute.CommandWithContext(ctx, turbostat, "-q", "sleep", "5")
	if output.AsError() != nil {
		return nil, output.Err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	headerIndex := make(map[string]int)
	var headers []string
	res := &turbostatResult{
		powerState: powerStatePowerSaving,
		pkgMap:     make(map[string][]*ThreadEntry),
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasSuffix(line, timeSuffix) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == pkg || fields[0] == core || fields[0] == die {
			headers = fields
			for i, header := range headers {
				headerIndex[header] = i
			}
			continue
		}

		if len(headers) == 0 {
			continue
		}

		getValue := func(key string) string {
			if index, ok := headerIndex[key]; ok && index < len(fields) {
				return fields[index]
			}
			return ""
		}

		getIntValue := func(key string) int {
			v, _ := strconv.Atoi(getValue(key))
			return v
		}

		minFreq := func(key string) {
			if v := getIntValue(key); v > 0 {
				if v < res.minFreq {
					res.minFreq = v
				}
			}
		}

		pkgVal := getValue(pkg)
		coreVal := getValue(core)
		cpuVal := getValue(cpu)

		if pkgVal == "-" || coreVal == "-" || cpuVal == "-" {
			res.basedFreq = getIntValue(tscMHz)
			res.temperature = getValue(pkgTmp) + " ℃"
			res.wattage = getValue(pkgWatt) + " W"
			continue
		}

		if pkgVal == "" {
			pkgVal = "0"
		}

		thr := &ThreadEntry{
			ProcessorID:   cpuVal,
			PhysicalID:    pkgVal,
			CoreID:        coreVal,
			CoreFrequency: getValue(bzyMHz),
			Temperature:   getValue(coreTmp),
		}

		minFreq(bzyMHz)

		res.pkgMap[pkgVal] = append(res.pkgMap[pkgVal], thr)
	}

	if res.minFreq-50 > res.basedFreq {
		res.powerState = powerStatePerformance
	}

	return res, scanner.Err()
}
