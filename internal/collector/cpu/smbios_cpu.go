package cpu

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

func (c *CPU) collectFromSMBIOS(ctx context.Context) error {
	cpus, err := smbios.GetTypeData[*smbios.Type4Processor](4)
	if err != nil {
		return err
	}

	for _, cpu := range cpus {
		c.CPUEntries = append(c.CPUEntries, &SMBIOSCPUEntry{
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
		})
	}

	return nil
}

func formatMHz(mhz int) string {
	return strconv.Itoa(mhz) + " MHz"
}
