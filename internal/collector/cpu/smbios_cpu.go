package cpu

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

func collectSMBIOSCPU(ctx context.Context) ([]*SmbiosCPU, error) {
	d, err := smbios.New(ctx)
	if err != nil {
		return nil, err
	}

	cpus, err := smbios.GetTypeData[*smbios.Type4Processor](d, 4)
	if err != nil {
		return nil, err
	}

	smbiosCPUs := make([]*SmbiosCPU, 0, len(cpus))
	for _, cpu := range cpus {
		c := &SmbiosCPU{
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
		smbiosCPUs = append(smbiosCPUs, c)
	}

	return smbiosCPUs, nil
}

func formatMHz(mhz int) string {
	return strconv.Itoa(mhz) + " MHz"
}
