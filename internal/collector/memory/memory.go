package memory

import (
	"context"

	"github.com/zenithax-cc/baize/pkg/utils"
)

func New() *Memory {
	return &Memory{}
}

func (m Memory) Collect(ctx context.Context) error {
	var errs []error
	phyMemory, err := collectPhysicalMemory(ctx)
	if err != nil {
		errs = append(errs, err)
	}
	m.PhysicalMemoryEntries = phyMemory

	sysMemory, err := collectMeminfo()
	if err != nil {
		errs = append(errs, err)
	}
	m.MemoryInfo = sysMemory

	edacMemory, err := collectEdacMemory(ctx)
	if err != nil {
		errs = append(errs, err)
	}
	m.EdacMemoryEntries = edacMemory

	return utils.CombineErrors(errs)
}
