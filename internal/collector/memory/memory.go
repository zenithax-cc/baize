// Package memory provides functionality for collecting system memory information,
// including physical DIMM data (SMBIOS), kernel memory stats (/proc/meminfo),
// and EDAC (Error Detection and Correction) memory error reporting.
package memory

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

// New creates and returns a new Memory instance with default "Healthy" diagnose
// status and pre-allocated slices for SMBIOS and EDAC memory entries.
func New() *Memory {
	return &Memory{
		Diagnose:              "Healthy",
		PhysicalMemoryEntries: make([]*SmbiosMemoryEntry, 0, 32),
		EdacMemoryEntries:     make([]*EdacMemoryEntry, 0, 32),
	}
}

// Collect gathers all memory information by invoking sub-collectors for
// /proc/meminfo, SMBIOS type-17 tables, and EDAC sysfs entries.
// It then correlates results and runs a health diagnosis.
func (m *Memory) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	errs := make([]error, 0, 4)

	// Collect runtime memory statistics from /proc/meminfo.
	if err := m.collectFromMeminfo(ctx); err != nil {
		errs = append(errs, err)
	}

	// Collect physical DIMM information from SMBIOS type-17 tables.
	if err := m.collectFromSMBIOS(ctx); err != nil {
		errs = append(errs, err)
	}

	// Collect EDAC memory error counters from /sys/bus/edac/devices.
	if err := m.collectEdacMemory(ctx); err != nil {
		errs = append(errs, err)
	}

	// Associate EDAC entries with SMBIOS entries and calculate total EDAC size.
	if err := m.associate(); err != nil {
		errs = append(errs, err)
	}

	// Run health checks and populate Diagnose / DiagnoseDetail fields.
	m.diagnose()
	return errors.Join(errs...)
}

// JSON serializes the Memory struct to JSON and writes it to stdout.
func (m *Memory) JSON() error {
	return utils.JSONPrintln(m)
}

// Name returns the collector identifier used for module routing.
func (m *Memory) Name() string {
	return "memory"
}

// DetailPrintln prints the full memory details (including per-DIMM entries) to stdout.
func (m *Memory) DetailPrintln() {
	memInfo := struct {
		MemoryInfo []*Memory `name:"MEMORY INFO" output:"both"`
	}{}

	memInfo.MemoryInfo = append(memInfo.MemoryInfo, m)

	utils.SP.Print(memInfo, "detail")
}

// BriefPrintln prints a brief memory summary grouped by manufacturer/size/speed/type,
// showing the quantity of each identical DIMM group.
func (m *Memory) BriefPrintln() {
	memInfo := struct {
		MemoryInfo   []*Memory `name:"MEMORY INFO" output:"both"`
		MemoryDetial []string  `name:"Memory Detail" output:"brief"`
	}{}

	memInfo.MemoryInfo = append(memInfo.MemoryInfo, m)

	// Group DIMMs by their identifying attributes and count duplicates.
	memMap := make(map[string]int)
	for _, entry := range m.PhysicalMemoryEntries {
		key := strings.Join([]string{
			entry.Manufacturer, entry.Size, entry.Speed, entry.DeviceType,
		}, " ")

		memMap[key]++
	}

	for key, count := range memMap {
		memInfo.MemoryDetial = append(memInfo.MemoryDetial, key+" * "+strconv.Itoa(count))
	}

	utils.SP.Print(memInfo, "brief")
}

// associate correlates EDAC and SMBIOS data:
// it counts the number of used DIMM slots and sums total EDAC-reported memory size.
func (m *Memory) associate() error {
	var (
		errs      []error
		totalSize int
	)

	m.UsedSlots = strconv.Itoa(len(m.PhysicalMemoryEntries))

	// Sum EDAC reported sizes (in MiB) to compute total EDAC memory.
	for _, edac := range m.EdacMemoryEntries {
		size, err := strconv.Atoi(edac.Size)
		if err != nil {
			errs = append(errs, err)
		}
		totalSize += size
	}

	m.EdacMemorySize = utils.KGMT(float64(totalSize*1024*1024), true)

	return errors.Join(errs...)
}

// diagnose performs sanity checks on the collected memory data and populates
// Diagnose and DiagnoseDetail fields with any detected anomalies.
func (m *Memory) diagnose() {
	var msg []string

	// Check for slot count mismatch between SMBIOS and EDAC.
	if m.EdacSlots != "" && m.EdacSlots != m.UsedSlots {
		msg = append(msg, "SMBIOS and EDAC memory slots are not equal")
	}

	// Check if the OS-visible memory size diverges from SMBIOS physical size
	// by more than one DIMM's worth (which may indicate a failed/missing module).
	sysSize, sysErr := toBytes(m.MemTotal)
	smbiosSize, smbiosErr := toBytes(m.PhysicalMemorySize)
	if sysErr == nil && smbiosErr == nil {
		if smbiosSize-sysSize > smbiosSize/len(m.PhysicalMemoryEntries) {
			msg = append(msg, "has unhealthy memory")
		}
	}

	// Warn when DIMM count is odd, which typically indicates an asymmetric configuration.
	if len(m.PhysicalMemoryEntries)%2 != 0 {
		msg = append(msg, "memory count should be even")
	}

	if len(msg) != 0 {
		m.Diagnose = "Unhealthy"
		m.DiagnoseDetail = strings.Join(msg, "; ")
	}
}
