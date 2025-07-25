package pkg

import (
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/gpu"
)

type GPU struct {
	*gpu.GPU
}

func NewGPU() *GPU {
	return &GPU{
		GPU: gpu.New(),
	}
}

func (g GPU) PrintJson() {
	printJson("GPU", g.GPU)
}

func (g GPU) PrintBrief() {
	if g.GPU == nil {
		fmt.Println("GPU information is not collected yet")
		return
	}
	fields := []string{"Device", "Vendor", "PCIeID"}
	fmt.Println("[GPU INFO]")
	for _, card := range g.GraphicsCards {
		sb := selectFields(card.PCIe, fields, 1, nil)
		fmt.Fprintf(sb, "%s%-*s: %v\n", "    ", 36, "IsOnBoard", card.IsOnBoard)
		fmt.Println(sb.String())
	}
}

func (g GPU) PrintDetail() {
	if g.GPU == nil {
		fmt.Println("GPU information is not collected yet")
		return
	}
	fields := []string{"Device", "Vendor", "PCIeID"}
	for _, card := range g.GraphicsCards {
		sb := selectFields(card.PCIe, fields, 1, nil)
		fmt.Fprintf(sb, "%s%-*s: %v\n", "    ", 36, "IsOnBoard", card.IsOnBoard)
		fmt.Println(sb.String())
	}
}
