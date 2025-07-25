package cpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

type LscpuInfo struct {
	Architecture     string   `json:"architecture,omitempty"`
	CPUOpMode        string   `json:"cpu_op_mode,omitempty"`
	AddressSizes     string   `json:"address_sizes,omitempty"`
	ByteOrder        string   `json:"byte_order,omitempty"`
	CPUs             string   `json:"cpus,omitempty"`
	OnlineCPUs       string   `json:"online_cpus,omitempty"`
	VendorID         string   `json:"vendor_id,omitempty"`
	ModelName        string   `json:"model_name,omitempty"`
	CPUFamily        string   `json:"cpu_family,omitempty"`
	CPUModel         string   `json:"cpu_model,omitempty"`
	ThreadsPerCore   string   `json:"threads_per_core,omitempty"`
	CoresPerSocket   string   `json:"cores_per_socket,omitempty"`
	Sockets          string   `json:"sockets,omitempty"`
	Stepping         string   `json:"stepping,omitempty"`
	BogoMIPS         string   `json:"bogomips,omitempty"`
	Virtualization   string   `json:"virtualization,omitempty"`
	L1dCache         string   `json:"l1d_cache,omitempty"`
	L1iCache         string   `json:"l1i_cache,omitempty"`
	L2Cache          string   `json:"l2_cache,omitempty"`
	L3Cache          string   `json:"l3_cache,omitempty"`
	MaximumFrequency string   `json:"maximum_frequency,omitempty"`
	MinimumFrequency string   `json:"minimum_frequency,omitempty"`
	PowerState       string   `json:"power_state,omitempty"`
	HyperThreading   string   `json:"hyper_threading,omitempty"`
	Temperature      string   `json:"temperature,omitempty"`
	Wattage          string   `json:"wattage,omitempty"`
	Diagnose         string   `json:"diagnose,omitempty"`
	DiagnoseDetail   string   `json:"diagnose_detail,omitempty"`
	Flags            []string `json:"flags,omitempty"`
}

type SmbiosInfo struct {
	SocketDesignation string         `json:"socket_designation,omitempty"`
	ProcessorType     string         `json:"processor_type,omitempty"`
	Family            string         `json:"family,omitempty"`
	Manufacturer      string         `json:"manufacturer,omitempty"`
	Version           string         `json:"version,omitempty"`
	ExternalClock     string         `json:"external_clock,omitempty"`
	CurrentSpeed      string         `json:"current_speed,omitempty"`
	Status            string         `json:"status,omitempty"`
	Voltage           string         `json:"voltage,omitempty"`
	CoreCount         string         `json:"core_count,omitempty"`
	CoreEnabled       string         `json:"core_enabled,omitempty"`
	ThreadCount       string         `json:"threads_count,omitempty"`
	Characteristics   []string       `json:"characteristics,omitempty"`
	ThreadEntries     []*ThreadEntry `json:"thread_entries,omitempty"`
}

type ThreadEntry struct {
	ProcessorID   string `json:"processor_id,omitempty"`
	CoreID        string `json:"core_id,omitempty"`
	PhysicalID    string `json:"physical_id,omitempty"`
	CoreFrequency string `json:"core_frequency,omitempty"`
	Temperature   string `json:"temperature,omitempty"`
}

type CPU struct {
	LscpuInfo
	PhysicalCPU []*SmbiosInfo `json:"physical_cpu,omitempty"`
}

var vendorMap = map[string]string{
	"AuthenticAMD": "AMD",
	"GenuineIntel": "Intel",
	"0x48":         "HiSilicon",
}

func New() *CPU {
	return &CPU{
		LscpuInfo: LscpuInfo{
			PowerState:     "Performance",
			HyperThreading: "Supported Enabled",
			Diagnose:       "Healthy",
			Flags:          make([]string, 0, 16),
		},
		PhysicalCPU: make([]*SmbiosInfo, 0, 2),
	}
}

const (
	timeOut = 10 * time.Second
)

func (c *CPU) Collect() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()
	var errs []error

	if err := c.lscpu(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := c.smbiosCPU(ctx); err != nil {
		errs = append(errs, err)
	}

	c.diagnose()

	if len(errs) > 0 {
		return fmt.Errorf("failed to collect CPU information: %v", errs)
	}
	return nil
}

func (c *CPU) lscpu(ctx context.Context) error {
	output, err := utils.Run.CommandContext(ctx, "lscpu")
	if err != nil {
		return fmt.Errorf("lscpu command failed: %w", err)
	}

	const (
		architecture   = "Architecture"
		opModes        = "CPU op-modes"
		byteOrder      = "Byte Order"
		addressSizes   = "Address sizes"
		cpus           = "CPUs"
		onlineCPUsList = "On-line CPUs list"
		threadsPerCore = "Threads per core"
		coresPerSocket = "Cores per socket"
		sockets        = "Sockets"
		numaNodes      = "NUMA nodes"
		vendorID       = "Vendor ID"
		cpuFamily      = "CPU family"
		model          = "Model"
		modelName      = "Model name"
		stepping       = "Stepping"
		bogoMIPS       = "BogoMIPS"
		virtualization = "Virtualization"
		l1dCache       = "L1d cache"
		l1iCache       = "L1i cache"
		l2Cache        = "L2 cache"
		l3Cache        = "L3 cache"
		flags          = "Flags"
	)

	aliasMap := map[string]string{
		"CPU op-mode(s)":      opModes,
		"CPU op-mode[s]":      opModes,
		"CPU(s)":              cpus,
		"CPU[s]":              cpus,
		"On-line CPU(s) list": onlineCPUsList,
		"On-line CPU[s] list": onlineCPUsList,
		"Thread(s) per core":  threadsPerCore,
		"Thread[s] per core":  threadsPerCore,
		"Core(s) per socket":  coresPerSocket,
		"Core[s] per socket":  coresPerSocket,
		"Socket(s)":           sockets,
		"Socket[s]":           sockets,
		"NUMA node(s)":        numaNodes,
		"NUMA node[s]":        numaNodes,
	}

	fieldMap := map[string]func(string){
		architecture:   func(v string) { c.Architecture = v },
		opModes:        func(v string) { c.CPUOpMode = v },
		byteOrder:      func(v string) { c.ByteOrder = v },
		addressSizes:   func(v string) { c.AddressSizes = v },
		cpus:           func(v string) { c.CPUs = v },
		onlineCPUsList: func(v string) { c.OnlineCPUs = v },
		threadsPerCore: func(v string) { c.ThreadsPerCore = v },
		coresPerSocket: func(v string) { c.CoresPerSocket = v },
		sockets:        func(v string) { c.Sockets = v },
		vendorID: func(v string) {
			if vendor, ok := vendorMap[v]; ok {
				c.VendorID = vendor
			} else {
				c.VendorID = v
			}
		},
		cpuFamily:      func(v string) { c.CPUFamily = v },
		model:          func(v string) { c.CPUModel = v },
		modelName:      func(v string) { c.ModelName = v },
		stepping:       func(v string) { c.Stepping = v },
		bogoMIPS:       func(v string) { c.BogoMIPS = v },
		virtualization: func(v string) { c.Virtualization = v },
		l1dCache:       func(v string) { c.L1dCache = v },
		l1iCache:       func(v string) { c.L1iCache = v },
		l2Cache:        func(v string) { c.L2Cache = v },
		l3Cache:        func(v string) { c.L3Cache = v },
		flags:          func(v string) { c.Flags = strings.Fields(v) },
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		key, value, found := utils.Cut(line, ":")
		if !found {
			continue
		}

		if alias, ok := aliasMap[key]; ok {
			key = alias
		}
		if fieldFunc, ok := fieldMap[key]; ok {
			fieldFunc(value)
		}
	}

	switch {
	case strings.HasPrefix(c.Architecture, "aarch"):
		c.HyperThreading = "Not Supported"
	case strings.HasPrefix(c.Architecture, "x86"):
		if c.ThreadsPerCore == "1" {
			c.HyperThreading = "Supported Disabled"
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading lscpu output: %w", err)
	}

	return nil
}

func (c *CPU) smbiosCPU(ctx context.Context) error {
	pkgs, err := smbios.GetTypeData[*smbios.Type4Processor](smbios.SMBIOS, 4)
	if len(pkgs) == 0 {
		return fmt.Errorf("no processor information found in SMBIOS : %v", err)
	}

	socketRegex0 := regexp.MustCompile(`^(P0|Proc 1|CPU 1|CPU01|CPU1)$`)
	socketRegex1 := regexp.MustCompile(`^(P1|Proc 2|CPU 2|CPU02|CPU2)$`)

	threadInfo, err := turbostat(ctx)
	if err != nil {
		return err
	}
	c.MaximumFrequency = fmt.Sprintf("%d MHz", threadInfo.maxFreq)
	c.MinimumFrequency = fmt.Sprintf("%d MHz", threadInfo.minFreq)
	c.Temperature = threadInfo.temperature
	c.Wattage = threadInfo.wattage
	if threadInfo.minFreq < threadInfo.basedFreq+50 {
		c.PowerState = "PowerSave"
	}

	for _, pkg := range pkgs {
		var socketID string
		res := &SmbiosInfo{
			SocketDesignation: pkg.SocketDesignation,
			ProcessorType:     pkg.ProcessorType.String(),
			Family:            pkg.GetFamily().String(),
			Manufacturer:      pkg.Manufacturer,
			Version:           pkg.Version,
			ExternalClock:     fmt.Sprintf("%d MHz", pkg.ExternalClock),
			CurrentSpeed:      fmt.Sprintf("%d MHz", pkg.CurrentSpeed),
			Status:            pkg.Status.String(),
			Voltage:           fmt.Sprintf("%0.2f v", pkg.GetVoltage()),
			CoreCount:         strconv.Itoa(pkg.GetCoreCount()),
			CoreEnabled:       strconv.Itoa(pkg.GetCoreEnabled()),
			ThreadCount:       strconv.Itoa(pkg.GetThreadCount()),
			Characteristics:   pkg.Characteristics.StringList(),
		}

		if socketRegex0.MatchString(pkg.SocketDesignation) {
			socketID = "0"
		} else if socketRegex1.MatchString(pkg.SocketDesignation) {
			socketID = "1"
		}

		if thr, exists := threadInfo.pkgMap[socketID]; exists && len(thr) > 0 {
			res.ThreadEntries = thr
		}
		c.PhysicalCPU = append(c.PhysicalCPU, res)
	}
	return nil
}

func (c *CPU) diagnose() {
	var sb strings.Builder
	if c.Sockets != strconv.Itoa(len(c.PhysicalCPU)) {
		fmt.Fprintf(&sb, "Sockets: %s, PhysicalCPU: %d;", c.Sockets, len(c.PhysicalCPU))
	}

	for _, pkg := range c.PhysicalCPU {
		if pkg.CoreCount != pkg.CoreEnabled {
			fmt.Fprintf(&sb, "CPU:%s, CoreCount: %s, CoreEnabled: %s;", pkg.SocketDesignation, pkg.CoreCount, pkg.CoreEnabled)
		}

		if pkg.Status != "Populated, Enabled" {
			fmt.Fprintf(&sb, "CPU:%s, Status: %s;", pkg.SocketDesignation, pkg.Status)
		}
	}

	if strings.HasPrefix(c.Architecture, "x86") {
		if c.ThreadsPerCore != "2" {
			fmt.Fprintf(&sb, "Threads Per Core: %s,except 2;", c.ThreadsPerCore)
		}
	}

	if sb.Len() > 0 {
		c.Diagnose = "Unhealthy"
		c.DiagnoseDetail = sb.String()
	}
}
