package cpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

type Turbostat struct {
	maxFreq     int
	minFreq     int
	basedFreq   int
	temperature string
	wattage     string
	pkgMap      map[string][]*ThreadEntry
}

const (
	turbostatCmd = `turbostat -q -s Package,Core,CPU,Bzy_MHz,TSC_MHz,CoreTmp,PkgTmp,PkgWatt sleep 5`

	minFieldCount = 4
	maxFieldCount = 8
	sumMarker     = `-`

	pkgKey     = "Package"
	coreKey    = "Core"
	cpuKey     = "CPU"
	bzyMhzKey  = "Bzy_MHz"
	tscMhzKey  = "TSC_MHz"
	coreTmpKey = "CoreTmp"
	pkgTmpKey  = "PkgTmp"
	pkgWattKey = "PkgWatt"
)

func NewTurbostat() *Turbostat {
	return &Turbostat{
		pkgMap: make(map[string][]*ThreadEntry),
	}
}

func (t *Turbostat) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	op, err := utils.Run.CommandContext(ctx, turbostatCmd)
	if err != nil {
		return fmt.Errorf("turbostat command failed: %w", err)
	}

	if err := t.processTurbostat(op); err != nil {
		return fmt.Errorf("failed to process turbostat output: %w", err)
	}

	return nil
}

func (t *Turbostat) processTurbostat(output []byte) error {

	scanner := bufio.NewScanner(bytes.NewReader(output))
	lineMap := map[string]string{
		pkgKey:     "0",
		coreKey:    "-",
		cpuKey:     "-",
		bzyMhzKey:  "-",
		tscMhzKey:  "-",
		coreTmpKey: "-",
		pkgTmpKey:  "-",
		pkgWattKey: "-",
	}
	keys := make([]string, 0, len(lineMap))

	result := make([]map[string]string, 192)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) < minFieldCount {
			continue
		}

		fields := strings.Fields(line)
		if strings.HasPrefix(line, pkgKey) || strings.HasPrefix(line, coreKey) {
			keys = fields
			continue
		}

		for i, field := range fields {
			lineMap[keys[i]] = field
		}

		result = append(result, lineMap)
	}

	var multiErr utils.MultiError
	if err := scanner.Err(); err != nil {
		multiErr.Add(fmt.Errorf("read turbostat output failed: %w", err))
	}

	if err := t.parseTurbostatLines(result); err != nil {
		multiErr.Add(fmt.Errorf("parse turbostat lines failed: %w", err))
	}

	return multiErr.Unwrap()
}

func (t *Turbostat) parseTurbostatLines(lines []map[string]string) error {
	var multiErr utils.MultiError

	for _, line := range lines {
		if line[pkgKey] == sumMarker || line[coreKey] == sumMarker {
			if err := t.processSummaryLine(line); err != nil {
				multiErr.Add(err)
			}
			continue
		}

		thread := &ThreadEntry{
			ProcessorID:   line[cpuKey],
			CoreID:        line[coreKey],
			PhysicalID:    line[pkgKey],
			CoreFrequency: line[bzyMhzKey],
			Temperature:   line[coreTmpKey],
		}

		if err := t.GetMaxAndMinFreq(line); err != nil {
			multiErr.Add(err)
		}

		key := fmt.Sprintf("%s_%s", thread.PhysicalID, thread.CoreID)
		t.pkgMap[key] = append(t.pkgMap[thread.PhysicalID], thread)
	}

	return multiErr.Unwrap()
}

func (t *Turbostat) processSummaryLine(line map[string]string) error {
	t.temperature = line[pkgTmpKey]
	t.wattage = line[pkgWattKey]
	basedFreq, err := strconv.Atoi(line[bzyMhzKey])
	if err != nil {
		return fmt.Errorf("convert based frequency failed: %w", err)
	}
	t.basedFreq = basedFreq

	return nil
}

func (t *Turbostat) GetMaxAndMinFreq(line map[string]string) error {
	freq, err := strconv.Atoi(line[bzyMhzKey])
	if err != nil {
		return fmt.Errorf("convert frequency failed: %w", err)
	}

	if freq > t.maxFreq {
		t.maxFreq = freq
	}

	if freq < t.minFreq {
		t.minFreq = freq
	}

	return nil
}
