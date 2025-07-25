package pkg

import (
	"strings"

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
	var sb strings.Builder
	sb.Grow(1000)
	sb.WriteString("[CPU INFO]\n")

	if c == nil {
		sb.WriteString("	no cpu info found\n")
		println(sb.String())
		return
	}

	fields := []string{"ModelName", "VendorID", "Architecture", "Sockets",
		"CPUs", "HyperThreading", "PowerState", "MaximumFrequency", "MinimumFrequency",
		"Temperature", "Wattage", "Diagnose", "DiagnoseDetail"}

	sb.WriteString(selectFields(c.CPU, fields, 1, colorMap["CPU"]).String() + "\n")

	println(sb.String())
}

func (c *CPU) PrintDetail() {}
