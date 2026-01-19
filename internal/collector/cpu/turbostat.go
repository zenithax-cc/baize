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
	headerFlag = "Package"
)

type turbostatResult struct {
	maxFreq     int
	minFreq     int
	basedFreq   int
	temperature string
	wattage     string
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
	res := &turbostatResult{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasSuffix(line, timeSuffix) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == headerFlag {
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

		getFloatValue := func(key string) float64 {
			v, _ := strconv.ParseFloat(getValue(key), 64)
			return v
		}

		getIntValue := func(key string) int {
			v, _ := strconv.Atoi(getValue(key))
			return v
		}

		pkg := getValue("Package")
		core := getValue("Core")
		cpu := getValue("CPU")

		if pkg == "-" || core == "-" || cpu == "-" {

		}

	}

	return res, nil
}
