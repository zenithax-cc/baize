package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/pkg"
)

// Componenters 组件接口定义
type Componenters interface {
	Collect(ctx context.Context) error
	PrintBrief()
	PrintDetail()
	PrintJSON()
	Name() string
}

// PrintType 打印类型定义
type PrintType uint8

const (
	PrintTypeBrief PrintType = iota
	PrintTypeDetail
	PrintTypeJSON
)

// ComponentType 组件类型定义
type ComponentType string

const (
	ComponentTypeProduct ComponentType = "product"
	ComponentTypeCPU     ComponentType = "cpu"
	ComponentTypeMemory  ComponentType = "memory"
	ComponentTypeRAID    ComponentType = "raid"
	ComponentTypeNetwork ComponentType = "network"
	ComponentTypeBond    ComponentType = "bond"
	ComponentTypeGPU     ComponentType = "gpu"
	ComponentTypeHealth  ComponentType = "health"
	ComponentTypeAll     ComponentType = "all"

	ComponentLen = 9
)

// FactoryFunc 工厂函数类型定义
type FactoryFunc func() Componenters
type ComponentRegistry struct {
	factories    map[ComponentType]FactoryFunc
	validTypes   map[ComponentType]struct{}
	orderedTypes []ComponentType
	mu           sync.RWMutex
}

// NewComponentRegistry 创建新的组件注册表
func NewComponentRegistry() *ComponentRegistry {
	registry := &ComponentRegistry{
		factories:    make(map[ComponentType]FactoryFunc, ComponentLen),
		validTypes:   make(map[ComponentType]struct{}, ComponentLen),
		orderedTypes: make([]ComponentType, 0, ComponentLen),
	}

	components := []struct {
		cType   ComponentType
		factory FactoryFunc
	}{
		{ComponentTypeProduct, func() Componenters { return pkg.NewProduct() }},
		{ComponentTypeCPU, func() Componenters { return pkg.NewCPU() }},
		{ComponentTypeMemory, func() Componenters { return pkg.NewMemory() }},
		{ComponentTypeRAID, func() Componenters { return pkg.NewRAID() }},
		{ComponentTypeNetwork, func() Componenters { return pkg.NewNetwork() }},
		{ComponentTypeBond, func() Componenters { return pkg.NewBond() }},
		{ComponentTypeGPU, func() Componenters { return pkg.NewGPU() }},
		{ComponentTypeHealth, func() Componenters { return pkg.NewHealth() }},
	}

	registry.orderedTypes = append(registry.orderedTypes, ComponentTypeAll)
	for _, comp := range components {
		registry.Register(comp.cType, comp.factory)
	}

	return registry
}

// Register 注册组件工厂函数 - 支持运行时注册
func (cr *ComponentRegistry) Register(cType ComponentType, factory FactoryFunc) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.factories[cType] = factory
	cr.validTypes[cType] = struct{}{}
	cr.validTypes[ComponentTypeAll] = struct{}{}

	if cType != ComponentTypeAll {
		cr.orderedTypes = append(cr.orderedTypes, cType)
	}
}

// IsValidComponent 检查组件类型是否有效
func (cr *ComponentRegistry) IsValidComponent(cType ComponentType) bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	_, exists := cr.validTypes[cType]

	return exists
}

// Create 创建组件实例
func (cr *ComponentRegistry) Create(cType ComponentType) (Componenters, error) {
	cr.mu.RLock()
	factory, exists := cr.factories[cType]
	cr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported component type: %s", cType)
	}

	return factory(), nil
}

// GetValidTypes 获取所有有效的组件类型,返回字符串
func (cr *ComponentRegistry) GetValidTypes() string {
	cr.mu.RLock()
	types := make([]string, 0, len(cr.validTypes))
	for t := range cr.validTypes {
		types = append(types, string(t))
	}
	cr.mu.RUnlock()

	sort.Strings(types)

	strBuilder := utils.StrBuilderPool.Get().(*strings.Builder)
	defer func() {
		strBuilder.Reset()
		utils.StrBuilderPool.Put(strBuilder)
	}()

	for i, t := range types {
		if i > 0 {
			strBuilder.WriteString(", ")
		}
		strBuilder.WriteString(t)
	}

	return strBuilder.String()
}

// GetOrderedComponents 获取有序的组件类型(用于 all 模式)
func (cr *ComponentRegistry) GetOrderedComponents() []ComponentType {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	result := make([]ComponentType, len(cr.orderedTypes)-1)
	copy(result, cr.orderedTypes[1:])

	return result
}

// ComponentManager 组件管理器
type ComponentManager struct {
	registry *ComponentRegistry
	timeout  time.Duration
}

// NewComponentManager 创建新的组件管理器
func NewComponentManager(registry *ComponentRegistry, timeout time.Duration) *ComponentManager {
	return &ComponentManager{
		registry: registry,
		timeout:  600 * time.Second,
	}
}

// GetInstances 获取指定类型的组件实例
func (cm *ComponentManager) GetInstances(cType ComponentType) ([]Componenters, error) {
	if cType == ComponentTypeAll {
		return cm.createAllInstances()
	}

	instance, err := cm.registry.Create(cType)
	if err != nil {
		return nil, fmt.Errorf("failed to create component %s: %w", cType, err)
	}

	return []Componenters{instance}, nil
}

// createAllInstances 创建所有组件实例
func (cm *ComponentManager) createAllInstances() ([]Componenters, error) {
	orderedTypes := cm.registry.GetOrderedComponents()

	type result struct {
		instance Componenters
		err      error
		index    int
	}

	results := make(chan result, len(orderedTypes))
	var wg sync.WaitGroup

	for i, cType := range orderedTypes {
		wg.Add(1)
		go func(index int, cType ComponentType) {
			defer wg.Done()
			instance, err := cm.registry.Create(cType)
			results <- result{instance, err, index}
		}(i, cType)
	}

	wg.Wait()
	close(results)

	instances := make([]Componenters, len(orderedTypes))
	var multiErr utils.MultiError
	validCount := 0

	for r := range results {
		if r.err != nil {
			multiErr.Add(fmt.Errorf("failed to create component %s: %v", orderedTypes[r.index], r.err))
		} else {
			instances[r.index] = r.instance
			validCount++
		}
	}

	if validCount == 0 {
		return nil, fmt.Errorf("no valid components created: %w", multiErr.Unwrap())
	}

	filtered := make([]Componenters, 0, validCount)
	for _, inst := range instances {
		if inst != nil {
			filtered = append(filtered, inst)
		}
	}

	return filtered, multiErr.Unwrap()
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	ConcurrentLimit int
	Timeout         time.Duration
	FailFast        bool
}

// ComponentProcessor 组件处理器
type ComponentProcessor struct {
	config ProcessorConfig
}

// NewComponentProcessor 创建新的组件处理器
func NewComponentProcessor(config ProcessorConfig) *ComponentProcessor {
	if config.ConcurrentLimit <= 0 {
		config.ConcurrentLimit = 1
	}
	if config.Timeout <= 0 {
		config.Timeout = 600 * time.Second
	}

	return &ComponentProcessor{config: config}
}

func (cp *ComponentProcessor) Process(ctx context.Context, instances []Componenters, printType PrintType) error {
	if len(instances) == 0 {
		return fmt.Errorf("no instances to process")
	}

	ctx, cancel := context.WithTimeout(ctx, cp.config.Timeout)
	defer cancel()

	semaphore := make(chan struct{}, cp.config.ConcurrentLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var multiErr utils.MultiError

	printFuncs := map[PrintType]func(Componenters){
		PrintTypeBrief:  func(c Componenters) { c.PrintBrief() },
		PrintTypeDetail: func(c Componenters) { c.PrintDetail() },
		PrintTypeJSON:   func(c Componenters) { c.PrintJSON() },
	}

	printFunc, exists := printFuncs[printType]
	if !exists {
		return fmt.Errorf("invalid print type: %v", printType)
	}

	for _, instance := range instances {
		wg.Add(1)
		go func(c Componenters) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				multiErr.Add(fmt.Errorf("context cancelled for %s: %v", c.Name(), ctx.Err()))
				mu.Unlock()
				return
			}
			if err := c.Collect(ctx); err != nil {
				mu.Lock()
				multiErr.Add(fmt.Errorf("failed to collect data for %s: %v", c.Name(), err))
				mu.Unlock()

				if cp.config.FailFast {
					cancel()
					return
				}
			}
			printFunc(c)
		}(instance)
	}

	wg.Wait()

	return multiErr.Unwrap()
}

type CLIConfig struct {
	Mode       string
	ShowDetail bool
	JOSNOutput bool
	Concurrent int
	Timeout    time.Duration
	FailFast   bool
}

// parseCLIFlags 解析命令行参数
func parseCLIFlags(registry *ComponentRegistry) (*CLIConfig, error) {
	config := &CLIConfig{}
	flag.StringVar(&config.Mode, "m", "all",
		fmt.Sprintf("Query mode information, supported values: %s", registry.GetValidTypes()))
	flag.BoolVar(&config.ShowDetail, "d", false, "Show detailed information")
	flag.BoolVar(&config.JOSNOutput, "j", false, "Output in JSON format")
	flag.IntVar(&config.Concurrent, "c", 1, "Number of concurrent workers (default: 1)")
	flag.DurationVar(&config.Timeout, "t", 600*time.Second, "Processing timeout (default: 30s)")
	flag.BoolVar(&config.FailFast, "f", false, "Fail fast on first error")

	flag.Parse()

	if config.Concurrent < 1 {
		config.Concurrent = 1
	}
	if config.Timeout < time.Second {
		config.Timeout = time.Second
	}

	return config, nil
}

// determineOutputType 确定输出类型 - 优先级处理
func determineOutputType(config *CLIConfig) PrintType {
	switch {
	case config.JOSNOutput:
		return PrintTypeJSON
	case config.ShowDetail:
		return PrintTypeDetail
	default:
		return PrintTypeBrief
	}
}

type Application struct {
	registry *ComponentRegistry
	manager  *ComponentManager
}

func NewApplication() *Application {
	registry := NewComponentRegistry()
	manager := NewComponentManager(registry, 600*time.Second)

	return &Application{
		registry: registry,
		manager:  manager,
	}
}

// Run 运行应用程序
func (app *Application) Run() error {
	// 解析命令行参数
	config, err := parseCLIFlags(app.registry)
	if err != nil {
		return fmt.Errorf("parse CLI flags failed: %w", err)
	}

	// 验证模式
	ctype := ComponentType(config.Mode)
	if !app.registry.IsValidComponent(ctype) {
		fmt.Printf("Invalid component type: %s\n", config.Mode)
		flag.Usage()
		return fmt.Errorf("invalid component type: %s", config.Mode)
	}

	// 获取实例
	instances, err := app.manager.GetInstances(ctype)
	if err != nil {
		return fmt.Errorf("get instances failed: %w", err)
	}

	if len(instances) == 0 {
		return fmt.Errorf("no instances found for type: %s", ctype)
	}

	// 配置处理器
	ProcessorConfig := ProcessorConfig{
		ConcurrentLimit: config.Concurrent,
		Timeout:         config.Timeout,
		FailFast:        config.FailFast,
	}

	processor := NewComponentProcessor(ProcessorConfig)
	outputType := determineOutputType(config)

	// 处理组件
	ctx := context.Background()
	return processor.Process(ctx, instances, outputType)
}

func main() {
	app := NewApplication()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
