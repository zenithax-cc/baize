package cpu

import (
	"bytes"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
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

type fieldSetter func(*CPU, string)

var lscpuFieldSetters = map[string]fieldSetter{
	"Architecture":        func(info *CPU, value string) { info.Architecture = value },
	"Byte Order":          func(info *CPU, value string) { info.ByteOrder = value },
	"Address sizes":       func(info *CPU, value string) { info.AddressSizes = value },
	"CPU family":          func(info *CPU, value string) { info.CPUFamily = value },
	"Model":               func(info *CPU, value string) { info.CPUModel = value },
	"Model name":          func(info *CPU, value string) { info.ModelName = value },
	"Stepping":            func(info *CPU, value string) { info.Stepping = value },
	"BogoMIPS":            func(info *CPU, value string) { info.BogoMIPS = value },
	"Virtualization":      func(info *CPU, value string) { info.Virtualization = value },
	"L1d cache":           func(info *CPU, value string) { info.L1dCache = value },
	"L1i cache":           func(info *CPU, value string) { info.L1iCache = value },
	"L2 cache":            func(info *CPU, value string) { info.L2Cache = value },
	"L3 cache":            func(info *CPU, value string) { info.L3Cache = value },
	"CPU(s)":              func(info *CPU, value string) { info.CPUs = value },
	"On-line CPU(s) list": func(info *CPU, value string) { info.OnlineCPUs = value },
	"Thread(s) per core":  func(info *CPU, value string) { info.ThreadsPerCore = value },
	"Core(s) per socket":  func(info *CPU, value string) { info.CoresPerSocket = value },
	"Socket(s)":           func(info *CPU, value string) { info.Sockets = value },
	"Vendor ID": func(info *CPU, value string) {
		if vendor, ok := vendorMap[value]; ok {
			info.VendorID = vendor
		} else {
			info.VendorID = value
		}
	},
	"CPU op-mode(s)": func(info *CPU, value string) { info.CPUOpMode = value },
	"Flags":          func(info *CPU, value string) { info.Flags = strings.Fields(value) },
}

func (c *CPU) collectFromLscpu() error {
	output := execute.Command(lscpu)
	if output.Err != nil {
		return output.Err
	}

	scanner := utils.NewScanner(bytes.NewReader(output.Stdout))
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}

		if setter, ok := lscpuFieldSetters[k]; ok {
			setter(c, v)
		}
	}

	if c.Architecture == "aarch64" {
		c.HyperThreading = htNotSupported
	}

	if c.ThreadsPerCore == "1" && c.Architecture == "x86_64" {
		c.HyperThreading = htSupportedDisabled
	}

	return scanner.Err()
}
