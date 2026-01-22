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

func collectTemperature(vendor string) (map[string]int, error) {
	switch vendor {
	case "AMD":
		return collectAMDTemperature()
	case "Intel":
		return collectIntelTemperature()
	default:
		return nil, errors.New("unknown cpu vendor")
	}
}

func collectAMDTemperature() (map[string]int, error) {
	output := execute.ShellCommand(fmt.Sprintf("%s sdr type temperature | egrep 'CPU[0-9]+[_ ]Temp'", ipmitool))
	if output.Err != nil {
		return nil, output.Err
	}

	cpus := strings.Split(string(output.Stdout), "\n")
	if len(cpus) == 0 {
		return nil, errors.New("amd cpu temperature not found")
	}

	res := make(map[string]int, len(cpus))
	for _, cpu := range cpus {
		parts := strings.Split(cpu, "|")
		if len(parts) != 5 {
			continue
		}

		name := strings.TrimSpace(parts[0])[0:4]
		value := strings.TrimSpace(parts[4])[0:2]

		println(name, value)

		if socketID, exists := socketIDMap[name]; exists {
			n, _ := strconv.Atoi(value)
			res[socketID] = n
		}
	}

	return res, nil
}

func collectIntelTemperature() (map[string]int, error) {
	hwmonDirs, err := os.ReadDir(hwmon)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not exists: %w", hwmon, err)
		}
		return nil, fmt.Errorf("read %s: %w", hwmon, err)
	}

	coretemp := make([]string, 0, 2)
	for _, dir := range hwmonDirs {
		link, err := os.Readlink(filepath.Join(hwmon, dir.Name()))
		if err != nil {
			continue
		}

		if strings.Contains(link, "coretemp") {
			coretemp = append(coretemp, link)
		}
	}

	if len(coretemp) == 0 {
		return nil, fmt.Errorf("no coretemp found")
	}

	res := make(map[string]int)

	for _, dir := range coretemp {
		labels, err := filepath.Glob(filepath.Join(hwmon, dir, "temp*_label"))
		if err != nil {
			continue
		}

		var pid string
		tmp := make([]struct {
			id    string
			value int
		}, 0, len(labels))

		for _, label := range labels {
			var id string
			content, err := os.ReadFile(label)
			if err != nil {
				continue
			}

			if bytes.HasPrefix(content, []byte("Package id")) {
				pid = strings.Fields(strings.TrimSpace(string(content)))[2]
				id = pid
			} else if bytes.HasPrefix(content, []byte("Core")) {
				id = strings.Fields(strings.TrimSpace(string(content)))[1]
			} else {
				continue
			}

			inputFile := strings.Replace(label, "label", "input", 1)
			inputValue, err := os.ReadFile(inputFile)
			if err != nil {
				continue
			}

			value, err := strconv.Atoi(strings.TrimSpace(string(inputValue)))
			if err != nil {
				continue
			}

			tmp = append(tmp, struct {
				id    string
				value int
			}{id: id, value: value / 1000})
		}

		for _, t := range tmp {
			res[fmt.Sprintf("%s-%s", pid, t.id)] = t.value
		}
	}

	return res, nil
}
