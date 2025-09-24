package pkg

import (
	"fmt"
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/memory"
)

type Memory struct {
	*memory.Memory
}

func NewMemory() *Memory {
	return &Memory{
		Memory: memory.New(),
	}
}

func (m *Memory) PrintJSON() {
	printJson("Memory", m.Memory)
}

func (m *Memory) PrintBrief() {
	var sb strings.Builder
	sb.Grow(1000)
	sb.WriteString("[MEMORY INFO]\n")

	if m.Memory == nil {
		sb.WriteString("	no memory found\n")
		println(sb.String())
		return
	}

	fileds := []string{"MemTotal", "MemAvailable", "SwapTotal", "MaximumSlots", "UsedSlots", "Diagnose", "DiagnoseDetail"}

	sb.WriteString(selectFields(m, fileds, 1, colorMap["Memory"]).String())

	memMap := map[string]int{}
	for _, mem := range m.PhysicalMemoryEntries {
		name := fmt.Sprintf("%s %s %s", mem.Manufacturer, mem.Size, mem.Speed)
		memMap[name]++
	}

	for name, num := range memMap {
		fmt.Fprintf(&sb, "%s%s * %d\n", "    	", name, num)
	}

	println(sb.String())
}

func (m *Memory) PrintDetail() {}

func (m *Memory) Name() string {
	return "Memory"
}
