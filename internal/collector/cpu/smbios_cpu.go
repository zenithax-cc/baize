package cpu

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

func collectSMBIOSCPU(ctx context.Context) (SMBIOSCPU, error) {
	cpus, err := collectSMBIOSCPUEntry(ctx)
	if err != nil {
		return SMBIOSCPU{}, err
	}

	res := SMBIOSCPU{
		CPUEntries: make([]*SMBIOSCPUEntry, 0, len(cpus)),
	}

	vendor := getVendor(cpus[0].Version)
	freqs, err := collectFrequency(ctx, vendor)
	if err != nil {
		return res, err
	}

	res.BasedFreqMHz = formatMHz(freqs.basedFreq)
	res.MaxFreqMHz = formatMHz(freqs.maxFreq)
	res.MinFreqMHz = formatMHz(freqs.minFreq)
	res.Watt = fmt.Sprintf("%0.2f W", freqs.watt)
	res.PowerState = freqs.powerState

	for _, c := range cpus {
		if socketID, exists := socketIDMap[c.SocketDesignation]; exists {
			c.ThreadEntries = freqs.threadMap[socketID]
		}
	}

	res.CPUEntries = cpus

	return res, nil
}

func collectSMBIOSCPUEntry(ctx context.Context) ([]*SMBIOSCPUEntry, error) {
	d, err := smbios.New(ctx)
	if err != nil {
		return nil, err
	}

	cpus, err := smbios.GetTypeData[*smbios.Type4Processor](d, 4)
	if err != nil {
		return nil, err
	}

	res := make([]*SMBIOSCPUEntry, 0, len(cpus))

	for _, cpu := range cpus {
		c := &SMBIOSCPUEntry{
			SocketDesignation: cpu.SocketDesignation,
			ProcessorType:     cpu.ProcessorType.String(),
			Family:            cpu.GetFamily().String(),
			Manufacturer:      cpu.Manufacturer,
			Version:           cpu.Version,
			ExternalClock:     formatMHz(int(cpu.ExternalClock)),
			CurrentSpeed:      formatMHz(int(cpu.CurrentSpeed)),
			Status:            cpu.Status.String(),
			Voltage:           fmt.Sprintf("%.2f v", cpu.GetVoltage()),
			CoreCount:         strconv.Itoa(cpu.GetCoreCount()),
			CoreEnabled:       strconv.Itoa(cpu.GetCoreEnabled()),
			ThreadCount:       strconv.Itoa(cpu.GetThreadCount()),
			Characteristics:   cpu.Characteristics.StringList(),
		}
		res = append(res, c)
	}

	return res, nil
}

func formatMHz(mhz int) string {
	return strconv.Itoa(mhz) + " MHz"
}

func getVendor(version string) string {
	switch {
	case strings.HasPrefix(version, "AMD"):
		return "AMD"
	case strings.HasPrefix(version, "Intel"):
		return "Intel"
	default:
		return strings.Fields(version)[0]
	}
}
