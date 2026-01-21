package cpu

import (
	"context"
	"fmt"
	"strconv"

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

	if core, err := collectThreadSummary(ctx); err == nil {
		res.BasedFreqMHz = formatMHz(core.basedFreq)
		res.MinFreqMHz = formatMHz(core.minFreq)
		res.MaxFreqMHz = formatMHz(core.maxFreq)
		res.PowerState = core.powerState
		res.TemperatureCelsius = strconv.Itoa(core.temperature) + " °C"
		res.Watt = strconv.Itoa(core.wattage) + " W"
		for _, cpu := range cpus {
			if socketID, exists := socketIDMap[cpu.SocketDesignation]; exists {
				if thrs, exists := core.threadMap[socketID]; exists && len(thrs) > 0 {
					cpu.ThreadEntries = thrs
				}
			}
			res.CPUEntries = append(res.CPUEntries, cpu)
		}
	}

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
