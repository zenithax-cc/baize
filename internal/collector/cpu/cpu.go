package cpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

const (
	// CPU架构类型
	archAArch = "aarch"
	archX86   = "x86"

	// CPU电源状态
	powerStatePerformance = "Performance"
	powerStatePowerSaving = "PowerSaving"

	// 超线程状态
	htSupported         = "Supported Enabled"
	htNotSupported      = "Not Supported"
	htSupportedDisabled = "Supported Disabled"

	// CPU诊断结果
	diagnoseHealthy   = "Healthy"
	diagnoseUnhealthy = "Unhealthy"
	// CPU状态
	statusPopulatedEnabled = "Populated, Enabled"
)

var (
	// 正则表达式匹配CPU信息
	socketRegex0 = regexp.MustCompile(`^(P0|Proc 1|CPU 1|CPU01|CPU1)`)
	socketRegex1 = regexp.MustCompile(`^(P1|Proc 2|CPU 2|CPU02|CPU2)`)

	// 使用sync.Pool复用字符串构建器，减少内存分配
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// CPU厂商映射表
	vendorMap = map[string]string{
		"AuthenticAMD": "AMD",
		"GenuineIntel": "Intel",
		"0x48":         "HiSilicon",
	}
)

// 配置项，用于未来扩展
func withConfig() *CPU {
	return &CPU{
		LscpuInfo: LscpuInfo{
			PowerState:     powerStatePerformance,
			HyperThreading: htSupported,
			Diagnose:       diagnoseHealthy,
			Flags:          make([]string, 0, 16),
		},
		PhysicalCPU: make([]*SmbiosInfo, 0, 2),
	}
}

func New() *CPU {
	return withConfig()
}

func (c *CPU) Collect(ctx context.Context) error {
	errChan := make(chan error, 2)

	go func() {
		errChan <- c.collectLscpuInfo(ctx)
	}()

	go func() {
		errChan <- c.collectSmbiosInfo(ctx)
	}()

	var multiErr utils.MultiError
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			multiErr.Add(err)
		}
	}

	c.diagnose()

	return multiErr.Unwrap()
}

func (c *CPU) collectLscpuInfo(ctx context.Context) error {
	// 执行lscpu命令获取CPU信息
	op, err := utils.Run.CommandContext(ctx, "lscpu")
	if err != nil {
		return fmt.Errorf("failed to run lscpu: %w", err)
	}

	// 预定义字段解析器，避免大量的字符串比较
	type fieldParser struct {
		keys   []string
		parser func(string)
	}

	parsers := []fieldParser{
		{[]string{"Architecture"}, func(v string) { c.Architecture = v }},
		{[]string{"CPU op-mode(s)", "CPU op-modes", "CPU op-mode[s]"}, func(v string) { c.CPUOpMode = v }},
		{[]string{"Byte Order"}, func(v string) { c.ByteOrder = v }},
		{[]string{"Address sizes"}, func(v string) { c.AddressSizes = v }},
		{[]string{"CPU(s)", "CPUs", "CPU[s]"}, func(v string) { c.CPUs = v }},
		{[]string{"On-line CPU(s) list", "On-line CPUs list", "On-line CPU[s] list"}, func(v string) { c.OnlineCPUs = v }},
		{[]string{"Thread(s) per core", "Threads per core", "Thread[s] per core"}, func(v string) { c.ThreadsPerCore = v }},
		{[]string{"Core(s) per socket", "Cores per socket", "Core[s] per socket"}, func(v string) { c.CoresPerSocket = v }},
		{[]string{"Socket(s)", "Sockets", "Socket[s]"}, func(v string) { c.Sockets = v }},
		{[]string{"Vendor ID"}, func(v string) {
			if vendor, ok := vendorMap[v]; ok {
				c.VendorID = vendor
			} else {
				c.VendorID = v
			}
		}},
		{[]string{"CPU family"}, func(v string) { c.CPUFamily = v }},
		{[]string{"Model"}, func(v string) { c.CPUModel = v }},
		{[]string{"Model name"}, func(v string) { c.ModelName = v }},
		{[]string{"Stepping"}, func(v string) { c.Stepping = v }},
		{[]string{"BogoMIPS"}, func(v string) { c.BogoMIPS = v }},
		{[]string{"Virtualization"}, func(v string) { c.Virtualization = v }},
		{[]string{"L1d cache"}, func(v string) { c.L1dCache = v }},
		{[]string{"L1i cache"}, func(v string) { c.L1iCache = v }},
		{[]string{"L2 cache"}, func(v string) { c.L2Cache = v }},
		{[]string{"L3 cache"}, func(v string) { c.L3Cache = v }},
		{[]string{"Flags"}, func(v string) {
			fields := strings.Fields(v)
			if cap(c.Flags) < len(fields) {
				c.Flags = make([]string, 0, len(fields))
			} else {
				c.Flags = c.Flags[:0]
			}
			c.Flags = append(c.Flags, fields...)
		}},
	}

	// 构建快速查找表
	fieldMap := make(map[string]func(string))
	for _, parser := range parsers {
		for _, key := range parser.keys {
			fieldMap[key] = parser.parser
		}
	}
	// 解析lscpu输出，提取CPU架构、型号、核心数等信息
	scanner := bufio.NewScanner(bytes.NewReader(op))
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024) // 64KB缓冲区

	for scanner.Scan() {
		line := scanner.Text()
		if colonIndex := strings.IndexByte(line, ':'); colonIndex != -1 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])

			if parser, exists := fieldMap[key]; exists {
				parser(value)
			}
		}
	}

	// 超线程状态检查
	c.updateHyperThreadingStatus()

	return scanner.Err()

}

func (c *CPU) updateHyperThreadingStatus() {
	switch {
	case strings.HasPrefix(c.Architecture, archAArch):
		c.HyperThreading = htNotSupported
	case strings.HasPrefix(c.Architecture, archX86):
		if c.ThreadsPerCore == "1" {
			c.HyperThreading = htSupportedDisabled
		}
	}
}

func (c *CPU) collectSmbiosInfo(ctx context.Context) error {
	pkgs, err := smbios.GetTypeData[*smbios.Type4Processor](smbios.SMBIOS, 4)
	if err != nil || len(pkgs) == 0 {
		return fmt.Errorf("failed to get smbios type 4 processor: %w", err)
	}

	turbostat := NewTurbostat()
	if err := turbostat.Collect(ctx); err != nil {
		return fmt.Errorf("failed to collect turbostat: %w", err)
	}

	c.setFrequencyAndPowerInfo(turbostat)

	if cap(c.PhysicalCPU) < len(pkgs) {
		c.PhysicalCPU = make([]*SmbiosInfo, 0, len(pkgs))
	}

	for _, pkg := range pkgs {
		smbiosInfo := c.createSmbiosInfo(pkg, turbostat)
		c.PhysicalCPU = append(c.PhysicalCPU, smbiosInfo)
	}

	return nil
}

func (c *CPU) setFrequencyAndPowerInfo(threadInfo *Turbostat) {
	c.MaximumFrequency = fmt.Sprintf("%d MHz", threadInfo.maxFreq)
	c.MinimumFrequency = fmt.Sprintf("%d MHz", threadInfo.minFreq)
	c.Temperature = fmt.Sprintf("%s ℃", threadInfo.temperature)
	c.Wattage = fmt.Sprintf("%s W", threadInfo.wattage)

	if threadInfo.minFreq < threadInfo.basedFreq+50 {
		c.PowerState = powerStatePowerSaving
	}
}

func (c *CPU) createSmbiosInfo(pkg *smbios.Type4Processor, threadInfo *Turbostat) *SmbiosInfo {
	smbiosInfo := &SmbiosInfo{
		SocketDesignation: pkg.SocketDesignation,
		ProcessorType:     pkg.ProcessorType.String(),
		Family:            pkg.GetFamily().String(),
		Manufacturer:      pkg.Manufacturer,
		Version:           pkg.Version,
		ExternalClock:     fmt.Sprintf("%d MHz", pkg.ExternalClock),
		CurrentSpeed:      fmt.Sprintf("%d MHz", pkg.CurrentSpeed),
		Status:            pkg.Status.String(),
		Voltage:           fmt.Sprintf("%.2f v", pkg.GetVoltage()),
		CoreCount:         strconv.Itoa(pkg.GetCoreCount()),
		CoreEnabled:       strconv.Itoa(pkg.GetCoreEnabled()),
		ThreadCount:       strconv.Itoa(pkg.GetThreadCount()),
		Characteristics:   pkg.Characteristics.StringList(),
	}

	socketID := c.getSocketID(pkg.SocketDesignation)
	if threads, exists := threadInfo.pkgMap[socketID]; exists && len(threads) > 0 {
		smbiosInfo.ThreadEntries = threads
	}

	return smbiosInfo
}

func (c *CPU) getSocketID(socketDesignation string) string {
	if socketRegex0.MatchString(socketDesignation) {
		return "0"
	}
	if socketRegex1.MatchString(socketDesignation) {
		return "1"
	}
	return ""
}

func (c *CPU) diagnose() {
	sb := stringBuilderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		stringBuilderPool.Put(sb)
	}()

	// 检查socket数量一致性
	if socketCount := strconv.Itoa(len(c.PhysicalCPU)); c.Sockets != socketCount {
		fmt.Fprintf(sb, "Sockets: %s, PhysicalCPU: %d;", c.Sockets, len(c.PhysicalCPU))
	}

	// 检查CPU状态
	for _, pkg := range c.PhysicalCPU {
		if pkg.CoreCount != pkg.CoreEnabled {
			fmt.Fprintf(sb, "CPU:%s, CoreCount: %s, CoreEnabled: %s;",
				pkg.SocketDesignation, pkg.CoreCount, pkg.CoreEnabled)
		}

		if pkg.Status != statusPopulatedEnabled {
			fmt.Fprintf(sb, "CPU:%s, Status: %s;", pkg.SocketDesignation, pkg.Status)
		}
	}

	// 检查x86架构的超线程
	if strings.HasPrefix(c.Architecture, archX86) && c.ThreadsPerCore != "2" {
		fmt.Fprintf(sb, "Threads Per Core: %s, expected 2;", c.ThreadsPerCore)
	}

	// 设置诊断结果
	if sb.Len() > 0 {
		c.Diagnose = diagnoseUnhealthy
		c.DiagnoseDetail = sb.String()
	}
}
