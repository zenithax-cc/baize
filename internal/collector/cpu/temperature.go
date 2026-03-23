// Package cpu - temperature.go collects per-core and per-package CPU temperatures
// from vendor-specific sources: IPMI (AMD) and hwmon coretemp sysfs (Intel).
package cpu

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	ipmitool = "/usr/bin/ipmitool"
	hwmon    = "/sys/class/hwmon"
)

// collectAMDTemperature reads per-socket CPU temperatures via IPMI SDR.
// It returns a map of normalized socket ID (e.g. "0", "1") to temperature in Celsius.
func collectAMDTemperature() (map[string]int, error) {
	output := execute.ShellCommand(fmt.Sprintf("%s sdr type temperature | egrep 'CPU[0-9]+[_ ]Temp'", ipmitool))
	if output.Err != nil {
		return nil, output.Err
	}

	cpus := strings.Split(strings.TrimSpace(string(output.Stdout)), "\n")
	if len(cpus) == 0 || (len(cpus) == 1 && cpus[0] == "") {
		return nil, errors.New("amd cpu temperature not found")
	}

	res := make(map[string]int, len(cpus))
	for _, cpu := range cpus {
		parts := strings.Split(cpu, "|")
		if len(parts) != 5 {
			continue
		}

		// Extract first 4 chars of sensor name (e.g. "CPU0") as the socket key.
		name := strings.TrimSpace(parts[0])
		if len(name) < 4 {
			continue
		}
		name = name[:4]

		// Extract temperature value from the last field (first 2 chars).
		rawVal := strings.TrimSpace(parts[4])
		if len(rawVal) < 2 {
			continue
		}
		value := rawVal[:2]

		if socketID, exists := socketIDMap[name]; exists {
			n, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			res[socketID] = n
		}
	}

	return res, nil
}

// collectIntelTemperature reads per-core and per-package temperatures from the
// kernel hwmon coretemp sysfs interface (/sys/class/hwmon/hwmon*/temp*_label).
// Returns a map with two key formats:
//   - "<pid>-<pid>" for package-level (e.g. "0-0")
//   - "<pid>-<coreID>" for per-core (e.g. "0-2")
func collectIntelTemperature() (map[string]int, error) {
	hwmonDirs, err := os.ReadDir(hwmon)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not exists: %w", hwmon, err)
		}
		return nil, fmt.Errorf("read %s: %w", hwmon, err)
	}

	// Collect the hwmon directory names (e.g. "hwmon0") whose symlink targets contain "coretemp".
	coretempDirs := make([]string, 0, 2)
	for _, dir := range hwmonDirs {
		link, err := os.Readlink(filepath.Join(hwmon, dir.Name()))
		if err != nil {
			continue
		}
		if strings.Contains(link, "coretemp") {
			// Store the hwmon entry name (not the resolved link), so we can
			// correctly glob files under /sys/class/hwmon/<hwmonN>/temp*_label.
			coretempDirs = append(coretempDirs, dir.Name())
		}
	}

	if len(coretempDirs) == 0 {
		return nil, fmt.Errorf("no coretemp hwmon device found under %s", hwmon)
	}

	res := make(map[string]int)

	for _, dirName := range coretempDirs {
		dirPath := filepath.Join(hwmon, dirName)
		labels, err := filepath.Glob(filepath.Join(dirPath, "temp*_label"))
		if err != nil || len(labels) == 0 {
			continue
		}

		var pid string
		type tempEntry struct {
			id    string
			value int
		}
		tmp := make([]tempEntry, 0, len(labels))

		for _, label := range labels {
			content, err := os.ReadFile(label)
			if err != nil {
				continue
			}
			trimmed := strings.TrimSpace(string(content))

			var id string
			if bytes.HasPrefix(content, []byte("Package id")) {
				// e.g. "Package id 0" → pid = "0"
				fields := strings.Fields(trimmed)
				if len(fields) < 3 {
					continue
				}
				pid = fields[2]
				id = pid
			} else if bytes.HasPrefix(content, []byte("Core")) {
				// e.g. "Core 3" → id = "3"
				fields := strings.Fields(trimmed)
				if len(fields) < 2 {
					continue
				}
				id = fields[1]
			} else {
				continue
			}

			// Read the corresponding temperature input file (millidegrees Celsius).
			inputFile := strings.Replace(label, "_label", "_input", 1)
			inputValue, err := os.ReadFile(inputFile)
			if err != nil {
				continue
			}

			milliDeg, err := strconv.Atoi(strings.TrimSpace(string(inputValue)))
			if err != nil {
				continue
			}

			tmp = append(tmp, tempEntry{id: id, value: milliDeg / 1000})
		}

		// Build result map using "<packageID>-<coreID>" keys.
		// Package-level entry uses "<pid>-<pid>" so it can be looked up by physical ID alone.
		for _, t := range tmp {
			res[fmt.Sprintf("%s-%s", pid, t.id)] = t.value
		}
	}

	return res, nil
}
