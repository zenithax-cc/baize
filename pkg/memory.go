package pkg

import (
	"fmt"

	"github.com/zenithax-cc/baize/common/utils"
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

func (m *Memory) PrintJson() {
	printJson("Memory", m.Memory)
}

func (m *Memory) PrintBrief() {
	fileds := []string{"MemTotal", "MemAvailable", "SwapTotal", "MaximumSlots", "UsedSlots", "Diagnose", "DiagnoseDetail"}

	sb := utils.SelectFields(m, fileds, 1)
	memMap := map[string]int{}
	for _, mem := range m.PhysicalMemoryEntries {
		name := fmt.Sprintf("%s %s %s", mem.Manufacturer, mem.Size, mem.Speed)
		memMap[name]++
	}
	for name, num := range memMap {
		fmt.Fprintf(sb, "%s%s * %d\n", "    	", name, num)
	}
	println("[MEMORY INFO]")
	fmt.Println(sb.String())
}

func (m *Memory) PrintDetail() {}
