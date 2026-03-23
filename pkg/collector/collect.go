// Package collector provides the Manager that orchestrates hardware information
// collection across all supported modules (CPU, memory, RAID, network, etc.).
package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/zenithax-cc/baize/internal/collector/cpu"
	"github.com/zenithax-cc/baize/internal/collector/gpu"
	"github.com/zenithax-cc/baize/internal/collector/health"
	"github.com/zenithax-cc/baize/internal/collector/ipmi"
	"github.com/zenithax-cc/baize/internal/collector/memory"
	"github.com/zenithax-cc/baize/internal/collector/network"
	"github.com/zenithax-cc/baize/internal/collector/product"
	"github.com/zenithax-cc/baize/internal/collector/raid"
)

// Collector defines the interface that every hardware module must implement.
type Collector interface {
	Name() string
	Collect(context.Context) error
	DetailPrintln()
	BriefPrintln()
	JSON() error
}

// moduleType is a strongly-typed string for module identifiers.
type moduleType string

// Supported module identifiers.
const (
	ModuleTypeProduct moduleType = "product"
	ModuleTypeCPU     moduleType = "cpu"
	ModuleTypeMemory  moduleType = "memory"
	ModuleTypeRAID    moduleType = "raid"
	ModuleTypeNetwork moduleType = "network"
	ModuleTypeBond    moduleType = "bond"
	ModuleTypeGPU     moduleType = "gpu"
	ModuleTypeIPMI    moduleType = "ipmi"
	moduleTypeHealth  moduleType = "health"
)

// supportedModules is the ordered registry of all available collector modules.
// Each entry pairs a module name with a freshly instantiated Collector.
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
	{ModuleTypeIPMI, ipmi.New()},
	{moduleTypeHealth, health.New()},
}

// Manager controls which modules to run and how to present their output.
type Manager struct {
	Module     string       // target module name ("all" or a specific module)
	Json       bool         // output as JSON when true
	Detail     bool         // output detailed view when true
	Log        *slog.Logger // logger for operational messages
	collectors map[string]Collector
}

// getDefaultManager returns a Manager configured to collect all modules as JSON.
func getDefaultManager() *Manager {
	return &Manager{
		Log:        slog.Default(),
		collectors: make(map[string]Collector),
		Module:     "all",
		Json:       true,
	}
}

// NewManager initialises and runs the collection pipeline.
// If m is nil, a default Manager is used.
func NewManager(m *Manager) error {
	if m == nil {
		m = getDefaultManager()
	}

	m.collectors = make(map[string]Collector)
	m.SetModule()

	return m.Collect(context.Background())
}

// SetModule populates the collectors map based on the requested Module name.
// When Module is "all", every supported module is registered.
func (m *Manager) SetModule() {
	for _, c := range supportedModules {
		if m.Module == "all" {
			m.collectors[string(c.module)] = c.collector
			continue
		}

		if string(c.module) == m.Module {
			m.collectors[string(c.module)] = c.collector
			break
		}
	}
}

// Collect runs all registered collectors concurrently, then prints their output
// sequentially in the original registration order.
// All collection errors are joined and returned; output errors are logged only.
func (m *Manager) Collect(ctx context.Context) error {
	type result struct {
		name string
		c    Collector
		err  error
	}

	resultsCh := make(chan result, len(m.collectors))
	var wg sync.WaitGroup

	// Launch each collector in its own goroutine.
	for name, c := range m.collectors {
		wg.Add(1)
		go func(n string, col Collector) {
			defer wg.Done()
			err := col.Collect(ctx)
			resultsCh <- result{name: n, c: col, err: err}
		}(name, c)
	}

	// Close the channel once all goroutines complete.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect errors and build a completed-collector map for ordered output.
	var errs []error
	done := make(map[string]Collector, len(m.collectors))
	for r := range resultsCh {
		if r.err != nil {
			m.Log.Warn("collector error", "module", r.name, "error", r.err)
			errs = append(errs, fmt.Errorf("%s: %w", r.name, r.err))
		}
		done[r.name] = r.c
	}

	// Print results in the original module registration order for consistent output.
	for _, entry := range supportedModules {
		c, ok := done[string(entry.module)]
		if !ok {
			continue
		}
		switch {
		case m.Json:
			if err := c.JSON(); err != nil {
				m.Log.Warn("json output error", "module", entry.module, "error", err)
			}
		case m.Detail:
			c.DetailPrintln()
		default:
			c.BriefPrintln()
		}
	}

	return errors.Join(errs...)
}
