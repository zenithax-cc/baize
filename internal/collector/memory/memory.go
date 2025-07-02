package memory

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

type memoryInfo struct {
	MemTotal     string `json:"memory_total,omitempty"`
	MemAvailable string `json:"memory_available,omitempty"`
	MemUsed      string `json:"memory_used,omitempty"`
	SwapTotal    string `json:"swap_total,omitempty"`
	SwapUsed     string `json:"swap_used,omitempty"`
	Buffer       string `json:"buffer,omitempty"`
	Cached       string `json:"cached,omitempty"`
	Slab         string `json:"slab,omitempty"`
	SReclaimable string `json:"s_reclaimable,omitempty"`
	SUnreclaim   string `json:"s_unreclaim,omitempty"`
	KReclaimable string `json:"k_reclaimable,omitempty"`
	KernelStack  string `json:"kernel_stack,omitempty"`
	PageTables   string `json:"page_tables,omitempty"`
	Dirty        string `json:"dirty,omitempty"`
	Writeback    string `json:"writeback,omitempty"`
	HPagesTotal  string `json:"huge_page_total,omitempty"`
	HPageSize    string `json:"huge_page_size,omitempty"`
	HugeTlb      string `json:"huge_tlb,omitempty"`

	MaximumSlots   string `json:"maximum_slot,omitempty"`
	UsedSlots      string `json:"used_slots,omitempty"`
	Diagnose       string `json:"diagnose,omitempty"`
	DiagnoseDetail string `json:"diagnose_detail,omitempty"`
}

type smbiosMemory struct {
	TotalWidth        string `json:"total_width,omitempty"`
	DataWidth         string `json:"data_width,omitempty"`
	Size              string `json:"size,omitempty"`
	FormFactor        string `json:"form_factor,omitempty"`
	DeviceLocator     string `json:"device_locator,omitempty"`
	BankLocator       string `json:"bank_locator,omitempty"`
	Type              string `json:"type,omitempty"`
	TypeDetail        string `json:"type_detail,omitempty"`
	Speed             string `json:"speed,omitempty"`
	Manufacturer      string `json:"manufacturer,omitempty"`
	SerialNumber      string `json:"serial_number,omitempty"`
	PartNumber        string `json:"part_number,omitempty"`
	Rank              string `json:"rank,omitempty"`
	ConfiguredSpeed   string `json:"configured_speed,omitempty"`
	ConfiguredVoltage string `json:"configured_voltage,omitempty"`
	Technology        string `json:"technology,omitempty"`
}

type edacMemory struct {
	CorrectableErrors   string `json:"correctable_errors,omitempty"`
	UncorrectableErrors string `json:"uncorrectable_errors,omitempty"`
	DeviceType          string `json:"device_type,omitempty"`
	EdacMode            string `json:"edac_mode,omitempty"`
	MemoryLocation      string `json:"memory_location,omitempty"`
	MemoryType          string `json:"memory_type,omitempty"`
	SocketID            string `json:"socket_id,omitempty"`
	MemoryControllerID  string `json:"memory_controller_id,omitempty"`
	ChannelID           string `json:"channel_id,omitempty"`
	DIMMID              string `json:"dimm_id,omitempty"`
	Size                string `json:"size,omitempty"`
}

type Memory struct {
	memoryInfo
	PhysicalMemoryEntries []*smbiosMemory `json:"physical_memory_entries,omitempty"`
	EdacMemoryEntries     []*edacMemory   `json:"edac_memory_entries,omitempty"`
}

const (
	edacPath    = "/sys/devices/system/edac/mc/"
	meminfoPath = "/proc/meminfo"
	kbSuffix    = "kB"
)

func New() *Memory {
	return &Memory{
		memoryInfo: memoryInfo{
			Diagnose: "Healthy",
		},
		PhysicalMemoryEntries: make([]*smbiosMemory, 0, 24),
		EdacMemoryEntries:     make([]*edacMemory, 0, 24),
	}
}

func (m *Memory) Collect() error {
	var errs []error

	if err := m.readMeminfo(); err != nil {
		errs = append(errs, err)
	}

	if err := m.parseSmbiosMemory(); err != nil {
		errs = append(errs, err)
	}

	if err := m.parseEdacMemory(); err != nil {
		errs = append(errs, err)
	}

	m.diagnose()

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}
	return nil
}

func (m *Memory) readMeminfo() error {
	lines, err := utils.ReadLines(meminfoPath)
	if err != nil {
		return err
	}

	fieldMapping := map[string]*string{
		"MemTotal":        &m.MemTotal,
		"MemAvailable":    &m.MemAvailable,
		"SwapTotal":       &m.SwapTotal,
		"Buffers":         &m.Buffer,
		"Cached":          &m.Cached,
		"Slab":            &m.Slab,
		"SReclaimable":    &m.SReclaimable,
		"SUnreclaim":      &m.SUnreclaim,
		"KReclaimable":    &m.KReclaimable,
		"KernelStack":     &m.KernelStack,
		"PageTables":      &m.PageTables,
		"Dirty":           &m.Dirty,
		"Writeback":       &m.Writeback,
		"HugePages_Total": &m.HPagesTotal,
		"HugePagessize":   &m.HPageSize,
		"Hugetlb":         &m.HugeTlb,
	}

	memValue := map[string]float64{
		"MemTotal":     -1,
		"MemAvailable": -1,
		"SwapTotal":    -1,
		"SwapFree":     -1,
	}

	for _, line := range lines {
		key, value, ok := utils.Cut(line, ":")
		if !ok {
			continue
		}
		var valueFloat float64
		if strings.HasSuffix(value, kbSuffix) {
			trimmedValue := strings.TrimSpace(strings.TrimSuffix(value, kbSuffix))
			valueFloat, _ = strconv.ParseFloat(trimmedValue, 64)
		}
		if val, ok := memValue[key]; ok {
			if val == -1 {
				memValue[key] = valueFloat
			}
		}
		if field, exists := fieldMapping[key]; exists {
			val, err := utils.ConvertUnit(valueFloat, kbSuffix, true)
			if err != nil {
				val = "N/A"
			}
			*field = val
		}
	}

	m.MemUsed = calculateMemoryUsed(memValue["MemTotal"], memValue["MemAvailable"])
	m.SwapUsed = calculateMemoryUsed(memValue["SwapTotal"], memValue["SwapFree"])
	return nil
}

func calculateMemoryUsed(total, available float64) string {
	if total < 0 || available < 0 {
		return "N/A"
	}
	used, err := utils.ConvertUnit(total-available, kbSuffix, true)
	if err != nil {
		return "N/A"
	}
	return used
}

func (m *Memory) parseSmbiosMemory() error {
	memList, err := smbios.GetTypeData[*smbios.Type17MemoryDevice](smbios.SMBIOS, 17)

	if len(memList) == 0 {
		return fmt.Errorf("no memory device found in SMBIOS : %v", err)
	}

	bitWidthStr := func(v uint16) string {
		if v == 0 || v == 0xFFFF {
			return "Unknown"
		}
		return fmt.Sprintf("%d bits", v)
	}

	speedStr := func(v uint16) string {
		if v == 0 || v == 0xFFFF {
			return "Unknown"
		}
		return fmt.Sprintf("%d MT/s", v)
	}

	voltageStr := func(v uint16) string {
		switch {
		case v == 0:
			return "Unknown"
		case v%100 == 0:
			return fmt.Sprintf("%.1f V", float32(v)/1000.0)
		default:
			return fmt.Sprintf("%g V", float32(v)/1000.0)
		}
	}

	for _, mem := range memList {

		if speedStr(mem.Speed) == "Unknown" {
			continue
		}

		entry := &smbiosMemory{
			TotalWidth:        bitWidthStr(mem.TotalWidth),
			DataWidth:         bitWidthStr(mem.DataWidth),
			Size:              mem.GetSizeString(),
			FormFactor:        mem.FormFactor.String(),
			DeviceLocator:     mem.DeviceLocator,
			BankLocator:       mem.BankLocator,
			Type:              mem.Type.String(),
			TypeDetail:        mem.TypeDetail.String(),
			Speed:             speedStr(mem.Speed),
			Manufacturer:      mem.Manufacturer,
			SerialNumber:      mem.SerialNumber,
			PartNumber:        mem.PartNumber,
			Rank:              mem.GetRankString(),
			ConfiguredSpeed:   speedStr(mem.ConfiguredSpeed),
			ConfiguredVoltage: voltageStr(mem.ConfiguredVoltage),
			Technology:        mem.Technology.String(),
		}

		m.PhysicalMemoryEntries = append(m.PhysicalMemoryEntries, entry)
	}

	m.UsedSlots = strconv.Itoa(len(m.PhysicalMemoryEntries))
	m.MaximumSlots = strconv.Itoa(len(memList))

	return nil
}

func (m *Memory) parseEdacMemory() error {
	_, err := utils.ReadDir(edacPath)
	if err != nil {
		return err
	}

	dimmDirs, err := filepath.Glob(filepath.Join(edacPath, "mc*", "dimm*"))
	if err != nil {
		return fmt.Errorf("glob edac memory directory failed: %w", err)
	}

	if len(dimmDirs) == 0 {
		return fmt.Errorf("no EDAC memory directories found in %s", edacPath)
	}

	var errs []error
	for _, dimmDir := range dimmDirs {
		dimm, err := parseDimmDir(dimmDir)
		if err != nil {
			errs = append(errs, fmt.Errorf("parse dimm directory %s failed: %w", dimmDir, err))
		}
		m.EdacMemoryEntries = append(m.EdacMemoryEntries, dimm)
	}

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}

	return nil
}

func parseDimmDir(dir string) (*edacMemory, error) {
	dimm := &edacMemory{
		DIMMID: filepath.Base(dir),
	}

	files, err := utils.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s failed: %w", dir, err)
	}

	var errs []error
	handlerFunc := func(file string) string {
		content, err := utils.ReadOneLineFile(file)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s failed: %w", file, err))
		}
		return content
	}

	for _, file := range files {
		filePath := filepath.Join(dir, file.Name())
		switch file.Name() {
		case "dimm_ce_count":
			dimm.CorrectableErrors = handlerFunc(filePath)
		case "dimm_ue_count":
			dimm.UncorrectableErrors = handlerFunc(filePath)
		case "dimm_dev_type":
			dimm.DeviceType = handlerFunc(filePath)
		case "dimm_edac_mode":
			dimm.EdacMode = handlerFunc(filePath)
		case "dimm_location":
			dimm.MemoryLocation = handlerFunc(filePath)
		case "dimm_mem_type":
			dimm.MemoryType = handlerFunc(filePath)
		case "dimm_label":
			label := handlerFunc(filePath)
			parseDimmLabel(label, dimm)
		case "size":
			dimm.Size = handlerFunc(filePath)
		default:
			continue
		}
	}
	return dimm, nil
}

func parseDimmLabel(label string, dimm *edacMemory) {
	items := strings.Split(label, "_")
	for _, item := range items {
		if !strings.Contains(item, "#") {
			continue
		}
		key, value, found := utils.Cut(item, "#")
		if !found {
			continue
		}
		switch key {
		case "SrcID":
			dimm.SocketID = value
		case "MC":
			dimm.MemoryControllerID = value
		case "Chan":
			dimm.ChannelID = value
		case "DIMM":
			dimm.DIMMID = value
		default:
			continue
		}
	}
}

func (m *Memory) diagnose() {
	var sb strings.Builder

	if len(m.PhysicalMemoryEntries)%2 != 0 {
		fmt.Fprintf(&sb, "the amount of memory should be an even number: %s;", m.UsedSlots)
	}

	if len(m.EdacMemoryEntries) != len(m.PhysicalMemoryEntries) {
		fmt.Fprintf(&sb, "the amount of EDAC memory should be the same as physical memory: %d vs %d;", len(m.EdacMemoryEntries), len(m.PhysicalMemoryEntries))
	}

	if sb.Len() > 0 {
		m.Diagnose = "Unhealthy"
		m.DiagnoseDetail = sb.String()
	}
}
