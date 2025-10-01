package raid

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/pci"
)

// RAID/HBA卡厂商ID
type vendorID string

const (
	VendorLSI     vendorID = "1000"
	VendorHPE     vendorID = "103c"
	VendorIntel   vendorID = "8086"
	VendorAdaptec vendorID = "9005"
)

// RAID/HBA卡工具路径
const (
	smartctlPath = "/usr/sbin/smartctl"
	storcliPath  = "/usr/local/bin/storcli"
	hpssacliPath = "/usr/local/beidou/tool/hpssacli"
	arcconfPath  = "/usr/local/hwtool/tool/arcconf"
	mdadmPath    = "/usr/sbin/mdadm"
	sysBusPath   = "/sys/bus/pci/devices"
)

// 默认容量
const (
	defaultControllerNum = 2
	defaultNVMeNum       = 8
	defaultPDNum         = 16
	MaxConcurrency       = 3
)

// 控制器处理函数
type vendorHandleFunc func(context.Context, int, *controller) error

// 硬盘信息缓存
type PhysicalDriveCache struct {
	drives sync.Map
}

func (pdc *PhysicalDriveCache) Get(key string) (*physicalDrive, bool) {
	value, ok := pdc.drives.Load(key)
	if !ok {
		return nil, false
	}
	return value.(*physicalDrive), true
}

func (pdc *PhysicalDriveCache) Set(key string, value *physicalDrive) {
	pdc.drives.Store(key, value)
}

var (
	pdc = &PhysicalDriveCache{}

	vendorHandleMap = map[vendorID]vendorHandleFunc{
		VendorLSI:     lsiHandle,
		VendorHPE:     hpeHandle,
		VendorIntel:   intelHandle,
		VendorAdaptec: adaptecHandle,
	}
)

// 配置选项
type Config struct {
	ControllerCapcity int
	NVMeCapacity      int
	MaxConcurrency    int
}

func DefaultConfig() *Config {
	return &Config{
		ControllerCapcity: defaultControllerNum,
		NVMeCapacity:      defaultNVMeNum,
		MaxConcurrency:    MaxConcurrency,
	}
}

func New() *Controllers {
	return NewWithConfig(DefaultConfig())
}

func NewWithConfig(cfg *Config) *Controllers {
	return &Controllers{
		Controller: make([]*controller, 0, cfg.ControllerCapcity),
		NVMe:       make([]*nvme, 0, cfg.NVMeCapacity),
	}
}

type collectTask struct {
	name string
	fn   func(context.Context) error
}

type taskResult struct {
	name string
	err  error
}

// Collect 收集所有RAID/HBA卡信息
func (ctr *Controllers) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tasks := []collectTask{
		{name: "NVMe", fn: ctr.nvmeCollect},
		{name: "hpe", fn: ctr.vendorCollect},
	}

	return ctr.executeTasks(ctx, tasks)
}

// executeTasks 执行RAID/HBA采集任务列表
func (ctr *Controllers) executeTasks(ctx context.Context, tasks []collectTask) error {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	resultChan := make(chan taskResult, len(tasks))
	semaphore := make(chan struct{}, MaxConcurrency)

	for _, task := range tasks {
		task := task
		wg.Add(1)
		go func(t collectTask) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				resultChan <- taskResult{name: t.name, err: ctx.Err()}
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}
			err := t.fn(ctx)
			resultChan <- taskResult{name: t.name, err: err}
		}(task)
	}

	wg.Wait()
	close(resultChan)

	var multiErr utils.MultiError
	for result := range resultChan {
		if result.err != nil {
			multiErr.Add(fmt.Errorf("%s collect failed: %w", result.name, result.err))
		}
	}

	return multiErr.Unwrap()
}

// vendorCollect 收集供应商特定的raid卡信息
func (ctr *Controllers) vendorCollect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	pciBus, err := pci.GetSerialRAIDControllerPCIBus()
	if err != nil {
		return err
	}

	if len(pciBus) == 0 {
		return nil
	}

	var multiErr utils.MultiError

	// 顺序处理，避免占用过多系统资源
	busCount := len(pciBus)
	for _, bus := range pciBus {
		bus := bus

		if err := ctr.processVendor(ctx, busCount, bus); err != nil {
			multiErr.Add(err)
		}
	}

	return multiErr.Unwrap()
}

// processVendor 处理单个供应商的raid卡信息
func (ctr *Controllers) processVendor(ctx context.Context, busCount int, bus string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p := pci.New(bus)
	if err := p.Collect(); err != nil {
		return err
	}

	vendor := &controller{
		PCIe: p,
	}

	if handler, exists := vendorHandleMap[vendorID(vendor.PCIe.VendorID)]; exists {
		if err := handler(ctx, busCount, vendor); err != nil {
			return err
		}
	}

	ctr.Controller = append(ctr.Controller, vendor)

	return nil
}

// nvmeCollect 收集NVMe硬盘信息
func (ctr *Controllers) nvmeCollect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	pciBus, err := pci.GetNVMeControllerPCIBus()
	if err != nil {
		return err
	}

	if len(pciBus) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	resultChan := make(chan taskResult, len(pciBus))
	semaphore := make(chan struct{}, MaxConcurrency)

	for _, bus := range pciBus {
		bus := bus
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				resultChan <- taskResult{name: bus, err: ctx.Err()}
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}
			err := ctr.processNVMe(ctx, bus)
			resultChan <- taskResult{name: bus, err: err}
		}()
	}

	wg.Wait()
	close(resultChan)

	var multiErr utils.MultiError
	for result := range resultChan {
		if result.err != nil {
			multiErr.Add(fmt.Errorf("%s collect failed: %w", result.name, result.err))
		}
	}

	return multiErr.Unwrap()
}

const (
	NVMeInterface    = "U.2"
	NVMeMediumType   = "NVMe SSD"
	NVMeRotationRate = "Solid State Device"
	NVMeFormFactor   = "2.5 inch"
	DevicePrefix     = "/dev/"
	NVMeSubPath      = "nvme"
)

// processNVMe 处理单个NVMe硬盘信息
func (ctr *Controllers) processNVMe(ctx context.Context, bus string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p := pci.New(bus)
	if err := p.Collect(); err != nil {
		return err
	}

	nvme := &nvme{
		PCIe: p,
		physicalDrive: physicalDrive{
			RotationRate: NVMeRotationRate,
			MediumType:   NVMeMediumType,
			FormFactor:   NVMeFormFactor,
			Interface:    NVMeInterface,
		},
	}

	var multiErr utils.MultiError
	nvmePath := filepath.Join(sysBusPath, bus, NVMeSubPath)
	dirs, err := utils.ReadDir(nvmePath)
	if err != nil {
		multiErr.Add(err)
	}

	if len(dirs) == 1 {
		nvme.physicalDrive.MappingFile = DevicePrefix + dirs[0].Name()
		err := nvme.physicalDrive.getSmartctlData("nvme", "", "")
		multiErr.Add(err)

		namespacePath := filepath.Join(nvmePath, dirs[0].Name())
		namespaceDirs, err := utils.ReadDir(namespacePath)
		multiErr.Add(err)
		for _, dir := range namespaceDirs {
			name := DevicePrefix + dir.Name()
			nvme.Namespaces = append(nvme.Namespaces, name)
		}
	}

	ctr.NVMe = append(ctr.NVMe, nvme)

	return multiErr.Unwrap()
}
