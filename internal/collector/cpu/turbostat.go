package cpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

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
	turostatCMD = `turbostat -q -s Package,Core,CPU,Bzy_MHz,TSC_MHz,CoreTmp,PkgTmp,PkgWatt sleep 5`
)

func turbostat(ctx context.Context) (turbostatInfo, error) {
	res := turbostatInfo{
		pkgMap: make(map[string][]*ThreadEntry),
	}
	output, err := utils.Run.CommandContext(ctx, "bash", "-c", turostatCMD)
	if err != nil {
		return res, fmt.Errorf("turbostat command failed: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var ss [][]string
	for scanner.Scan() {
		line := scanner.Text()
		ll := strings.Fields(line)
		if len(ll) < 5 || ll[0] == "Package" {
			continue
		}
		ss = append(ss, ll)
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("error scanning turbostat output: %w", err)
	}

	coreTemp := make(map[string]string)
	for _, s := range ss {
		if s[0] == "-" && len(s) == 8 {
			res.basedFreq = utils.Atoi(s[4])
			res.temperature = s[6]
			res.wattage = s[7]
			continue
		}
		thread := &ThreadEntry{
			ProcessorID:   s[2],
			CoreID:        s[1],
			PhysicalID:    s[0],
			CoreFrequency: s[3],
		}
		coreKey := thread.PhysicalID + "_" + thread.CoreID
		if len(s) >= 6 {
			coreTemp[coreKey] = s[5]
		}

		if temp, exists := coreTemp[coreKey]; exists {
			thread.Temperature = temp
		}

		if utils.Atoi(s[3]) > res.maxFreq {
			res.maxFreq = utils.Atoi(s[3])
		}

		if utils.Atoi(s[3]) < res.minFreq || res.minFreq == 0 {
			res.minFreq = utils.Atoi(s[3])
		}

		res.pkgMap[thread.PhysicalID] = append(res.pkgMap[thread.PhysicalID], thread)
	}
	return res, nil
}
