package cpu

import (
	"context"
	"sync"
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
	var wg sync.WaitGroup
	wg.Add(2)

	return nil
}
