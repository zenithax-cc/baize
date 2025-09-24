package cpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
)

type turbostatInfo struct {
	maxFreq     int
	minFreq     int
	basedFreq   int
	temperature string
	wattage     string
	pkgMap      map[string][]*ThreadEntry
}

const (
	turbostatCmd = `turbostat -q -s Package,Core,CPU,Bzy_MHz,TSC_MHz,CoreTmp,PkgTmp,PkgWatt sleep 5`

	minFieldCount    = 5
	sumFieldCount    = 8
	sumMarker        = `-`
	headerMaker      = "Package"
	initialBuffSize  = 8192
	estimatedLine    = 66
	estimatedPackage = 2

	pkgIdx     = 0
	coreIdx    = 1
	cpuIdx     = 2
	bzyMhzIdx  = 3
	tscMhzIdx  = 4
	coreTmpIdx = 5
	pkgTmpIdx  = 6
	pkgWattIdx = 7
)

var (
	turbostatPool = sync.Pool{
		New: func() interface{} {
			return &turbostatInfo{
				pkgMap: make(map[string][]*ThreadEntry, estimatedPackage),
			}
		},
	}

	threadPool = sync.Pool{
		New: func() interface{} {
			return &ThreadEntry{}
		},
	}

	linesPool = sync.Pool{
		New: func() interface{} {
			return make([][]string, 0, estimatedLine)
		},
	}

	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

func newTurbostat() *turbostatInfo {
	info := turbostatPool.Get().(*turbostatInfo)

	info.maxFreq = 0
	info.minFreq = 0
	info.basedFreq = 0
	info.temperature = ""
	info.wattage = ""

	for k := range info.pkgMap {
		for _, entry := range info.pkgMap[k] {
			threadPool.Put(entry)
		}
		delete(info.pkgMap, k)
	}

	return info
}

func (t *turbostatInfo) Release() {
	turbostatPool.Put(t)
}

type turboError struct {
	Operation string
	Err       error
	context   string
}

func (e *turboError) Error() string {
	return fmt.Sprintf("turbostat %s error: %v, context: %s", e.Operation, e.Err, e.context)
}

func (e *turboError) Unwrap() error {
	return e.Err
}

func turbostat(ctx context.Context) (*turbostatInfo, error) {

	output, err := execCmd(ctx)
	if err != nil {
		return nil, err
	}

	info, err := parseTurbostat(output)
	if err != nil {
		if info != nil {
			info.Release()
		}
		return nil, err
	}

	if err := validateData(info); err != nil {
		info.Release()
		return nil, err
	}

	return info, nil
}

func execCmd(ctx context.Context) ([]byte, error) {
	op, err := utils.Run.CommandContext(ctx, "bash", "-c", turbostatCmd)
	if err != nil {
		return nil, &turboError{
			Operation: "command_execution",
			Err:       err,
			context:   "turbostat command execution failed",
		}
	}
	return op, nil
}

func parseTurbostat(output []byte) (*turbostatInfo, error) {
	info := newTurbostat()

	lines, err := preprocessLines(output)
	if err != nil {
		return info, err
	}
	defer func() {
		lines = lines[:0]
		linesPool.Put(lines)
	}()

	if err := processData(lines, info); err != nil {
		return info, err
	}

	return info, nil
}

func preprocessLines(output []byte) ([][]string, error) {
	lines := linesPool.Get().([][]string)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		fields := strings.Fields(line)

		if len(fields) < minFieldCount || fields[pkgIdx] == headerMaker {
			continue
		}

		lines = append(lines, fields)
	}

	if err := scanner.Err(); err != nil {
		return nil, &turboError{
			Operation: "line_scanning",
			Err:       err,
			context:   fmt.Sprintf("error scanning turbostat output lines: %d", lineNum),
		}
	}

	return lines, nil
}

func processData(lines [][]string, info *turbostatInfo) error {
	freqCache := make(map[string]int, len(lines))

	builder := builderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		builderPool.Put(builder)
	}()

	coreTemps := make(map[string]string, len(lines)/2)

	for lineIdx, fields := range lines {
		if fields[pkgIdx] == sumMarker && len(fields) == sumFieldCount {
			if err := processSummaryLine(fields, info, lineIdx); err != nil {
				return err
			}
			continue
		}

		freqStr := fields[bzyMhzIdx]
		if _, exists := freqCache[freqStr]; !exists {
			freq, err := fastAtoi(freqStr)
			if err != nil {
				return &turboError{
					Operation: "frequency_parsing",
					Err:       err,
					context:   fmt.Sprintf("error parsing Bzy_MHz frequency at line %d", lineIdx),
				}
			}
			freqCache[freqStr] = freq
		}
	}

	for lineIdx, fields := range lines {
		if fields[pkgIdx] == sumMarker {
			continue
		}

		if err := processThreadLine(fields, info, freqCache, coreTemps, builder, lineIdx); err != nil {
			return err
		}
	}

	return nil
}

func processSummaryLine(fields []string, info *turbostatInfo, lineIdx int) error {
	freq, err := fastAtoi(fields[bzyMhzIdx])
	if err != nil {
		return &turboError{
			Operation: "frequency_parsing",
			Err:       err,
			context:   fmt.Sprintf("error parsing TSC_MHz frequency at line %d", lineIdx),
		}
	}

	info.basedFreq = freq
	info.temperature = fields[coreTmpIdx]
	info.wattage = fields[pkgWattIdx]

	return nil
}

func processThreadLine(fields []string, info *turbostatInfo, freqCache map[string]int, coreTemps map[string]string, builder *strings.Builder, lineIdx int) error {
	thread := threadPool.Get().(*ThreadEntry)
	*thread = ThreadEntry{
		ProcessorID:   fields[cpuIdx],
		CoreID:        fields[coreIdx],
		PhysicalID:    fields[pkgIdx],
		CoreFrequency: fields[bzyMhzIdx],
	}

	builder.Reset()
	builder.WriteString(thread.PhysicalID)
	builder.WriteByte('_')
	builder.WriteString(thread.CoreID)
	coreKey := builder.String()

	if len(fields) > coreTmpIdx && fields[coreTmpIdx] != "" {
		coreTemps[coreKey] = fields[coreTmpIdx]
	}

	if temp, exists := coreTemps[coreKey]; exists {
		thread.Temperature = temp
	}

	freq := freqCache[fields[bzyMhzIdx]]
	if info.maxFreq < freq {
		info.maxFreq = freq
	}
	if info.minFreq == 0 || info.minFreq > freq {
		info.minFreq = freq
	}

	info.pkgMap[thread.PhysicalID] = append(info.pkgMap[thread.PhysicalID], thread)

	return nil

}

func fastAtoi(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}

	if len(s) == 1 {
		c := s[0]
		if c >= '0' && c <= '9' {
			return int(c - '0'), nil
		}
		return 0, fmt.Errorf("invalid character: %c", c)
	}

	return strconv.Atoi(s)
}

func validateData(info *turbostatInfo) error {

	if info.maxFreq <= 0 || info.minFreq <= 0 {
		return &turboError{
			Operation: "validation",
			Err:       fmt.Errorf("invalid frequency range: max=%d, min=%d", info.maxFreq, info.minFreq),
		}
	}

	if len(info.pkgMap) == 0 {
		return &turboError{
			Operation: "validation",
			Err:       fmt.Errorf("no CPU package data found"),
		}
	}

	return nil
}
