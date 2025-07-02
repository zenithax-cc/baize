package pkg

import (
	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/cpu"
)

type CPU struct {
	*cpu.CPU
}

func NewCPU() *CPU {
	return &CPU{
		CPU: cpu.New(),
	}
}

func (c *CPU) Collect() error {
	return c.CPU.Collect()
}

func (c *CPU) PrintJson() {
	printJson("CPU", c.CPU)
}

func (c *CPU) PrintBrief() {
	fields := []string{"ModelName", "VendorID", "Architecture", "Sockets",
		"CPUs", "HyperThreading", "PowerState", "MaximumFrequency", "MinimumFrequency",
		"Temperature", "Wattage", "Diagnose", "DiagnoseDetail"}

	println("[CPU INFO]")
	sb := utils.SelectFields(c.LscpuInfo, fields, 1)
	println(sb.String())
}

func (c *CPU) PrintDetail() {}
