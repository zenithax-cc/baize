package cpu

import (
	"context"
	"errors"
	"fmt"

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
		"P0": "0", "Proc 1": "0", "CPU 1": "0", "CPU01": "0", "CPU1": "0", "Socket 1": "0",
		"P1": "1", "Proc 2": "1", "CPU 2": "1", "CPU02": "1", "CPU2": "1", "Socket 2": "1",
	}
)

func New() *CPU {
	return &CPU{
		HyperThreading: htSupported,
		PowerState:     powerStatePowerSaving,
		CPUEntries:     make([]*SMBIOSCPUEntry, 0, 2),
	}
}

func (c *CPU) Collect(ctx context.Context) error {
	errs := make([]error, 0, 4)

	if err := c.collectFromLscpu(); err != nil {
		errs = append(errs, err)
	}

	if err := c.collectFromSMBIOS(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := c.collectFromTurbostat(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := c.associateCores(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *CPU) Name() string {
	return "cpu"
}

func (c *CPU) JSON() error {
	return utils.JSONPrintln(c)
}

func (c *CPU) DetailPrintln() {
	cpu := struct {
		CPUInfo []*CPU `json:"cpu" name:"CPU INFO" output:"both"`
	}{}

	cpu.CPUInfo = append(cpu.CPUInfo, c)

	utils.SP.Print(cpu, "detail")
}

func (c *CPU) BriefPrintln() {
	cpu := struct {
		CPUInfo []*CPU `json:"cpu" name:"CPU INFO" output:"both"`
	}{}

	cpu.CPUInfo = append(cpu.CPUInfo, c)

	utils.SP.Print(cpu, "brief")
}

func (c *CPU) associateCores() error {

	var (
		err     error
		errs    []error
		tempMap map[string]int
	)

	switch c.VendorID {
	case "Intel":
		tempMap, err = collectIntelTemperature()
	case "AMD":
		tempMap, err = collectAMDTemperature()
	}

	if err != nil {
		errs = append(errs, err)
	}

	for _, entry := range c.CPUEntries {
		id, ok := socketIDMap[entry.SocketDesignation]
		if !ok {
			errs = append(errs, errors.New("socket designation not found"))
			continue
		}
		for _, thread := range c.threads {
			if temp, ok := tempMap[thread.PhysicalID+"-"+thread.CoreID]; ok {
				thread.Temperature = fmt.Sprintf("%d ℃", temp)
			}

			if temp, ok := tempMap[thread.PhysicalID]; ok {
				thread.Temperature = fmt.Sprintf("%d ℃", temp)
			}

			if thread.PhysicalID == id {
				entry.ThreadEntries = append(entry.ThreadEntries, thread)
			}
		}
	}

	return errors.Join(errs...)
}
