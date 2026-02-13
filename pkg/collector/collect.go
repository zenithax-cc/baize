package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/zenithax-cc/baize/internal/collector/cpu"
	"github.com/zenithax-cc/baize/internal/collector/gpu"
	"github.com/zenithax-cc/baize/internal/collector/health"
	"github.com/zenithax-cc/baize/internal/collector/memory"
	"github.com/zenithax-cc/baize/internal/collector/network"
	"github.com/zenithax-cc/baize/internal/collector/product"
	"github.com/zenithax-cc/baize/internal/collector/raid"
)

type Collector interface {
	//	Name() string
	Collect(context.Context) error
	// Print()
	// ToJSON()
}

type moduleType string

const (
	ModuleTypeProduct moduleType = "product"
	ModuleTypeCPU     moduleType = "cpu"
	ModuleTypeMemory  moduleType = "memory"
	ModuleTypeRAID    moduleType = "raid"
	ModuleTypeNetwork moduleType = "network"
	ModuleTypeBond    moduleType = "bond"
	ModuleTypeGPU     moduleType = "gpu"
	moduleTypeHealth  moduleType = "health"
)

var supportedModules = []struct {
	module    moduleType
	collector Collector
}{
	{ModuleTypeProduct, product.New()},
	{ModuleTypeCPU, cpu.New()},
	{ModuleTypeMemory, memory.New()},
	{ModuleTypeRAID, raid.New()},
	{ModuleTypeNetwork, network.New()},
	{ModuleTypeBond, network.New()},
	{ModuleTypeGPU, gpu.New()},
	{moduleTypeHealth, health.New()},
}

type Manager struct {
	module     string
	json       bool
	detail     bool
	log        *slog.Logger
	collectors map[string]Collector
}

func NewManager() error {
	m := &Manager{
		log:        slog.Default(),
		collectors: make(map[string]Collector),
		module:     "all",
		json:       true,
	}

	m.SetModule()

	return m.Collect(context.Background())

}

func (m *Manager) SetModule() {
	for _, c := range supportedModules {
		if m.module == "all" {
			m.collectors[string(c.module)] = c.collector
			continue
		}

		if string(c.module) == m.module {
			m.collectors[string(c.module)] = c.collector
			break
		}
	}
}

func (m *Manager) Collect(ctx context.Context) error {
	var errs []error
	for _, c := range m.collectors {
		if err := c.Collect(ctx); err != nil {
			errs = append(errs, err)
		}
		ToJSON(c)
	}

	return errors.Join(errs...)
}

func ToJSON(text any) error {
	j, err := json.MarshalIndent(text, "  ", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	fmt.Println(string(j))
	return nil
}
