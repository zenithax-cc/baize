package pci

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zenithax-cc/baize/common/utils"
	"golang.org/x/sync/errgroup"
)

// PCIe 表示PCIe设备的信息
type PCIe struct {
	PCIeID      string `json:"pcie_id,omitempty"`
	PCIeAddr    string `json:"pcie_address,omitempty"`
	Vendor      string `json:"vendor,omitempty"`
	VendorID    string `json:"vendor_id,omitempty"`
	Device      string `json:"device,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	SubVendor   string `json:"sub_vendor,omitempty"`
	SubVendorID string `json:"sub_vendor_id,omitempty"`
	SubDevice   string `json:"sub_device,omitempty"`
	SubDeviceID string `json:"sub_device_id,omitempty"`
	Class       string `json:"class,omitempty"`
	ClassID     string `json:"class_id,omitempty"`
	SubClass    string `json:"sub_class,omitempty"`
	SubClassID  string `json:"sub_class_id,omitempty"`
	ProgIfID    string `json:"prog_interface_id,omitempty"`
	Numa        string `json:"numa,omitempty"`
	Revision    string `json:"revision,omitempty"`
	Driver      Driver `json:"driver,omitempty"`
	Link        Link   `json:"link,omitempty"`
}

// Driver 表示PCIe设备的驱动信息
type Driver struct {
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
	SrcVersion string `json:"src_version,omitempty"`
	File       string `json:"file_name,omitempty"`
}

// Link 表示PCIe设备的链接信息
type Link struct {
	MaxSpeed  string `json:"max_link_speed,omitempty"`
	MaxWidth  string `json:"max_link_width,omitempty"`
	CurrSpeed string `json:"current_link_speed,omitempty"`
	CurrWidth string `json:"current_link_width,omitempty"`
}

const (
	modaliasMinLength = 54
	busDir            = "/sys/bus/pci/devices"
	moduleDir         = "/sys/module"

	// modalias文件中各字段的固定位置
	vendorIDPos    = 9
	vendorIDLen    = 4
	deviceIDPos    = 18
	deviceIDLen    = 4
	subVendorIDPos = 28
	subVendorIDLen = 4
	subDeviceIDPos = 38
	subDeviceIDLen = 4
	classIDPos     = 44
	classIDLen     = 2
	subClassIDPos  = 48
	subClassIDLen  = 2
	progIfIDPos    = 51
	progIfIDLen    = 2
)

// 最大并发数量 - 避免过多goroutine
const maxConcurrency = 4

// 默认上下文超时时间
const defaultTimeout = 5 * time.Second

// pcieField 表示modalias文件中PCI字段的位置和长度
type pcieField struct {
	pos    int
	length int
}

// pcieFields是modalias文件中各字段位置的映射
// 使用全局变量避免每次调用时重新创建
var pcieFields = map[string]pcieField{
	"vendorID":    {pos: vendorIDPos, length: vendorIDLen},
	"deviceID":    {pos: deviceIDPos, length: deviceIDLen},
	"subVendorID": {pos: subVendorIDPos, length: subVendorIDLen},
	"subDeviceID": {pos: subDeviceIDPos, length: subDeviceIDLen},
	"classID":     {pos: classIDPos, length: classIDLen},
	"subClassID":  {pos: subClassIDPos, length: subClassIDLen},
	"progIfID":    {pos: progIfIDPos, length: progIfIDLen},
}

// 缓存单例的PCI IDs数据
var pciids = NewPCIIDs()

// 预声明的错误
var (
	errNilPCIe = errors.New("nil pcie object")
)

// infoSource 通用信息源接口，用于统一处理不同类型的信息采集
type infoSource struct {
	fileName string       // 文件名
	handler  func(string) // 处理函数
	isLink   bool         // 是否是软链接
}

// New 创建一个新的PCIe对象
func New(addr string) *PCIe {
	return &PCIe{
		PCIeAddr: addr,
	}
}

// Collect 收集PCIe设备的所有信息
func (p *PCIe) Collect() error {
	if p == nil {
		return errNilPCIe
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// 使用errgroup并发执行所有解析函数
	group, ctx := errgroup.WithContext(ctx)

	// 定义所有解析函数
	parseFuncs := []func(context.Context) error{
		func(ctx context.Context) error { return parseModalias(ctx, p) },
		func(ctx context.Context) error { return parseDriver(ctx, p) },
		func(ctx context.Context) error { return parseLink(ctx, p) },
		func(ctx context.Context) error { return parseOther(ctx, p) },
	}

	// 启动所有解析任务
	for _, fn := range parseFuncs {
		parseFunc := fn // 创建局部变量避免闭包问题
		group.Go(func() error {
			return parseFunc(ctx)
		})
	}

	// 等待所有任务完成
	if err := group.Wait(); err != nil {
		return fmt.Errorf("collecting pcie info: %w", err)
	}

	return nil
}

// parseModalias 解析modalias文件获取设备基本信息
func parseModalias(ctx context.Context, p *PCIe) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	file := filepath.Join(busDir, p.PCIeAddr, "modalias")
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read modalias file: %w", err)
	}

	// 检查长度以确保可以安全访问所有字段
	if len(content) < modaliasMinLength {
		return fmt.Errorf("modalias content too short: expected at least %d, got %d", modaliasMinLength, len(content))
	}

	// 转为小写以进行大小写不敏感的比较
	content = bytes.ToLower(content)

	// 优化：预先分配容量，避免扩容
	fields := make(map[string]string, len(pcieFields))

	// 从modalias内容中提取字段
	for name, field := range pcieFields {
		end := field.pos + field.length
		if end > len(content) {
			break
		}
		// 直接将字节切片转换为字符串，减少内存分配
		fields[name] = string(content[field.pos:end])
	}

	// 将提取的字段设置到PCIe对象
	p.VendorID = fields["vendorID"]
	p.ClassID = fields["classID"]
	p.SubClassID = fields["subClassID"]
	p.DeviceID = fields["deviceID"]
	p.SubVendorID = fields["subVendorID"]
	p.SubDeviceID = fields["subDeviceID"]
	p.ProgIfID = fields["progIfID"]

	// 构建ID组合用于查询名称
	vendorDeviceID := p.VendorID + ":" + p.DeviceID
	subVendorDeviceID := p.SubVendorID + ":" + p.SubDeviceID

	// 使用PCI IDs数据库查询名称
	p.Vendor = pciids.getVendorNameByID(p.VendorID)
	p.Device = pciids.getDeviceNameByID(vendorDeviceID)
	p.SubVendor = pciids.getVendorNameByID(p.SubVendorID)
	p.SubDevice = pciids.getSubDeviceNameByID(vendorDeviceID, subVendorDeviceID)

	p.PCIeID = p.VendorID + ":" + p.DeviceID + ":" + p.SubVendorID + ":" + p.SubDeviceID

	return nil
}

// parseFiles 通用文件解析函数，处理一组文件信息
func parseFiles(ctx context.Context, p *PCIe, sources []infoSource) error {
	// 检查上下文状态
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 基础路径
	basePath := filepath.Join(busDir, p.PCIeAddr)

	// 创建errgroup进行并发处理
	group, ctx := errgroup.WithContext(ctx)

	// 并发限制信号量
	semaphore := make(chan struct{}, maxConcurrency)

	for _, source := range sources {
		src := source // 创建局部变量避免闭包问题
		filePath := filepath.Join(basePath, src.fileName)

		// 检查文件是否存在
		if !utils.PathExists(filePath) {
			continue
		}

		group.Go(func() error {
			// 获取信号量令牌
			select {
			case semaphore <- struct{}{}:
				// 获取到令牌，继续执行
				defer func() { <-semaphore }()
			case <-ctx.Done():
				// 上下文取消
				return ctx.Err()
			}

			var content string
			var err error

			// 根据文件类型选择读取方法
			if src.isLink {
				content, err = os.Readlink(filePath)
			} else {
				content, err = utils.ReadOneLineFile(filePath)
			}

			if err != nil {
				return fmt.Errorf("reading %s: %w", src.fileName, err)
			}

			// 处理读取到的内容
			src.handler(strings.TrimSpace(content))
			return nil
		})
	}

	// 等待所有文件读取完成
	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

// parseDriver 解析驱动相关信息
func parseDriver(ctx context.Context, p *PCIe) error {
	// 定义驱动信息源
	sources := []infoSource{
		{
			fileName: "driver",
			handler:  func(s string) { p.Driver.Name = filepath.Base(s) },
			isLink:   true, // driver是软链接
		},
		{
			fileName: "version",
			handler:  func(s string) { p.Driver.Version = s },
		},
		{
			fileName: "srcversion",
			handler:  func(s string) { p.Driver.SrcVersion = s },
		},
	}

	// 使用通用解析函数处理文件
	if err := parseFiles(ctx, p, sources); err != nil {
		return fmt.Errorf("parse driver: %w", err)
	}

	// 如果有驱动名，获取驱动文件路径
	if p.Driver.Name != "" {
		// 创建带超时的上下文
		cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		out, err := utils.Run.CommandContext(cmdCtx, "modinfo", "-n", p.Driver.Name)
		if err != nil {
			return fmt.Errorf("get driver file: %w", err)
		}

		p.Driver.File = strings.TrimSpace(string(out))
	}

	return nil
}

// parseLink 解析PCIe链接信息
func parseLink(ctx context.Context, p *PCIe) error {
	sources := []infoSource{
		{
			fileName: "max_link_speed",
			handler:  func(s string) { p.Link.MaxSpeed = s },
		},
		{
			fileName: "max_link_width",
			handler:  func(s string) { p.Link.MaxWidth = s },
		},
		{
			fileName: "current_link_speed",
			handler:  func(s string) { p.Link.CurrSpeed = s },
		},
		{
			fileName: "current_link_width",
			handler:  func(s string) { p.Link.CurrWidth = s },
		},
	}

	if err := parseFiles(ctx, p, sources); err != nil {
		return fmt.Errorf("parse link: %w", err)
	}

	return nil
}

// parseOther 解析其他PCIe信息
func parseOther(ctx context.Context, p *PCIe) error {
	sources := []infoSource{
		{
			fileName: "numa_node",
			handler:  func(s string) { p.Numa = s },
		},
		{
			fileName: "revision",
			handler:  func(s string) { p.Revision = s },
		},
	}

	if err := parseFiles(ctx, p, sources); err != nil {
		return fmt.Errorf("parse other: %w", err)
	}

	return nil
}
