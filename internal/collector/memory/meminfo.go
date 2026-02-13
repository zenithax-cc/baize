package memory

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const procMeminfo = "/proc/meminfo"

func (m *Memory) collectFromMeminfo(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.Open(procMeminfo)
	if err != nil {
		return err
	}
	defer file.Close()

	fieldsMap := map[string]*string{
		"MemTotal":        &m.MemTotal,
		"MemFree":         &m.MemFree,
		"MemAvailable":    &m.MemAvailable,
		"SwapCached":      &m.SwapCached,
		"SwapTotal":       &m.SwapTotal,
		"SwapFree":        &m.SwapFree,
		"Buffers":         &m.Buffer,
		"Cached":          &m.Cached,
		"Slab":            &m.Slab,
		"SReclaimable":    &m.SReclaimable,
		"SUnreclaim":      &m.SUnreclaim,
		"KReclaimable":    &m.KReclaimable,
		"KernelStack":     &m.KernelStack,
		"PageTables":      &m.PageTables,
		"Dirty":           &m.Dirty,
		"Writeback":       &m.Writeback,
		"HugePages_Total": &m.HPagesTotal,
		"HugePagessize":   &m.HPageSize,
		"Hugetlb":         &m.HugeTlb,
	}
	scanner := utils.NewScanner(file)
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}

		if ptr, exists := fieldsMap[k]; exists {
			*ptr = convertUnit(v)
		}
	}

	return scanner.Err()
}

func convertUnit(value string) string {
	if value == "" || value == "0" {
		return "0"
	}

	num, _, ok := strings.Cut(value, " ")
	if !ok {
		return value
	}

	numUint, err := strconv.ParseFloat(num, 64)
	if err != nil {
		fmt.Printf("meminfo parseuint: %v", err)
		return value
	}

	return utils.KGMT(numUint*1024, true)
}
