package memory

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const procMeminfo = "/proc/meminfo"

func collectMeminfo() (MemoryInfo, error) {
	content, err := os.Open(procMeminfo)
	if err != nil {
		if os.IsNotExist(err) {
			return MemoryInfo{}, nil
		}
		return MemoryInfo{}, err
	}
	defer content.Close()

	scanner := bufio.NewScanner(content)
	var res MemoryInfo
	fieldsMap := map[string]*string{
		"MemTotal":        &res.MemTotal,
		"MemAvailable":    &res.MemAvailable,
		"SwapTotal":       &res.SwapTotal,
		"Buffers":         &res.Buffer,
		"Cached":          &res.Cached,
		"Slab":            &res.Slab,
		"SReclaimable":    &res.SReclaimable,
		"SUnreclaim":      &res.SUnreclaim,
		"KReclaimable":    &res.KReclaimable,
		"KernelStack":     &res.KernelStack,
		"PageTables":      &res.PageTables,
		"Dirty":           &res.Dirty,
		"Writeback":       &res.Writeback,
		"HugePages_Total": &res.HPagesTotal,
		"HugePagessize":   &res.HPageSize,
		"Hugetlb":         &res.HugeTlb,
	}

	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if ptr, exists := fieldsMap[key]; exists {
			*ptr = convertUnit(value)
		}
	}

	return res, scanner.Err()
}

func convertUnit(value string) string {
	if value == "" || value == "0" {
		return "0"
	}

	num, _, ok := strings.Cut(value, " ")
	if !ok {
		return value
	}

	numUint, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return value
	}

	return utils.KGMT(numUint)
}
