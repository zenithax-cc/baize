package cpu

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	lscpu = "/usr/bin/lscpu"
)

var (
	vendorMap = map[string]string{
		"AuthenticAMD": "AMD",
		"GenuineIntel": "Intel",
		"0x48":         "HiSilicon",
	}
)

type fieldSetter func(*LscpuInfo, string)

var lscpuFieldSetters = map[string]fieldSetter{
	"Architecture":   func(info *LscpuInfo, value string) { info.Architecture = value },
	"Byte Order":     func(info *LscpuInfo, value string) { info.ByteOrder = value },
	"Address sizes":  func(info *LscpuInfo, value string) { info.AddressSizes = value },
	"CPU family":     func(info *LscpuInfo, value string) { info.CPUFamily = value },
	"Model":          func(info *LscpuInfo, value string) { info.CPUModel = value },
	"Model name":     func(info *LscpuInfo, value string) { info.ModelName = value },
	"Stepping":       func(info *LscpuInfo, value string) { info.Stepping = value },
	"BogoMIPS":       func(info *LscpuInfo, value string) { info.BogoMIPS = value },
	"Virtualization": func(info *LscpuInfo, value string) { info.Virtualization = value },
	"L1d cache":      func(info *LscpuInfo, value string) { info.L1dCache = value },
	"L1i cache":      func(info *LscpuInfo, value string) { info.L1iCache = value },
	"L2 cache":       func(info *LscpuInfo, value string) { info.L2Cache = value },
	"L3 cache":       func(info *LscpuInfo, value string) { info.L3Cache = value },

	"CPU(s)":              func(info *LscpuInfo, value string) { info.CPUs = value },
	"On-line CPU(s) list": func(info *LscpuInfo, value string) { info.OnlineCPUs = value },
	"Thread(s) per core":  func(info *LscpuInfo, value string) { info.ThreadsPerCore = value },
	"Core(s) per socket":  func(info *LscpuInfo, value string) { info.CoresPerSocket = value },
	"Socket(s)":           func(info *LscpuInfo, value string) { info.Sockets = value },
	"Vendor ID": func(info *LscpuInfo, value string) {
		if vendor, ok := vendorMap[value]; ok {
			info.VendorID = vendor
		} else {
			info.VendorID = value
		}
	},
	"CPU op-mode(s)": func(info *LscpuInfo, value string) { info.CPUOpMode = value },
	"Flags":          func(info *LscpuInfo, value string) { info.Flags = strings.Fields(value) },
}

func collectLscpuInfo() (LscpuInfo, error) {
	output := execute.Command(lscpu)
	if output.AsError() != nil {
		return LscpuInfo{}, output.Err
	}

	res := LscpuInfo{}
	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if setter, ok := lscpuFieldSetters[key]; ok {
			setter(&res, value)
		}
	}

	return res, nil
}
