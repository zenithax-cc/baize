package cpu

import (
	"context"

	"github.com/zenithax-cc/baize/pkg/utils"
)

const (
	archARM = "aarch"
	archX86 = "x86"

	powerStatePerformance = "Performance"
	powerStatePowerSaving = "PowerSaving"

	htSupported         = "Supported Enabled"
	htNotSupported      = "Not Supported"
	htSupportedDisabled = "Supported Disabled"

	diagnoseHealthy   = "Healthy"
	diagnoseUnhealthy = "Unhealthy"

	statusPopulatedEnabled = "Populated, Enabled"
)

var (
	socketIDMap = map[string]string{
		"P0": "0", "Proc 1": "0", "CPU 1": "0", "CPU01": "0", "CPU1": "0",
		"P1": "1", "Proc 2": "1", "CPU 2": "1", "CPU02": "1", "CPU2": "1",
	}
)

func New() *CPU {
	return &CPU{
		SummaryCPU: SummaryCPU{
			HyperThreading: htSupported,
		},
		SMBIOSCPU: SMBIOSCPU{
			PowerState: powerStatePowerSaving,
			CPUEntries: make([]*SMBIOSCPUEntry, 0, 2),
		},
	}
}

func (c *CPU) Collect(ctx context.Context) error {
	errs := make([]error, 0, 2)
	lscpuInfo, err := collectSummaryCPU()
	if err != nil {
		errs = append(errs, err)
	}
	c.SummaryCPU = lscpuInfo

	smbiosCPU, err := collectSMBIOSCPU(ctx)
	if err != nil {
		errs = append(errs, err)
	}
	c.SMBIOSCPU = smbiosCPU

	return utils.CombineErrors(errs)
}
