package memory

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/utils"
)

func New() *Memory {
	return &Memory{
		Diagnose:              "Healthy",
		PhysicalMemoryEntries: make([]*SmbiosMemoryEntry, 0, 32),
		EdacMemoryEntries:     make([]*EdacMemoryEntry, 0, 32),
	}
}

func (m *Memory) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	errs := make([]error, 0, 4)

	if err := m.collectFromMeminfo(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := m.collectFromSMBIOS(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := m.collectEdacMemory(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := m.associate(); err != nil {
		errs = append(errs, err)
	}

	m.diagnose()
	return errors.Join(errs...)
}

func (m *Memory) JSON() error {
	return utils.JSONPrintln(m)
}

func (m *Memory) Name() string {
	return "memory"
}

func (m *Memory) DetailPrintln() {
	memInfo := struct {
		MemoryInfo []*Memory `name:"MEMORY INFO" output:"both"`
	}{}

	memInfo.MemoryInfo = append(memInfo.MemoryInfo, m)

	utils.SP.Print(memInfo, "detail")
}

func (m *Memory) BriefPrintln() {
	memInfo := struct {
		MemoryInfo   []*Memory `name:"MEMORY INFO" output:"both"`
		MemoryDetial []string  `name:"Memory Detail" output:"brief"`
	}{}

	memInfo.MemoryInfo = append(memInfo.MemoryInfo, m)
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

func (m *Memory) associate() error {
	var (
		errs      []error
		totalSize int
	)
	m.UsedSlots = strconv.Itoa(len(m.PhysicalMemoryEntries))
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

func (m *Memory) diagnose() {
	var msg []string

	if m.EdacSlots != "" && m.EdacSlots != m.UsedSlots {
		msg = append(msg, "SMBIOS and EDAC memory slots are not equal")
	}

	sysSize, sysErr := toBytes(m.MemTotal)
	smbiosSize, smbiosErr := toBytes(m.PhysicalMemorySize)
	if sysErr == nil && smbiosErr == nil {
		if smbiosSize-sysSize > smbiosSize/len(m.PhysicalMemoryEntries) {
			msg = append(msg, "has unhealthy memory")
		}
	}

	if len(m.PhysicalMemoryEntries)%2 != 0 {
		msg = append(msg, "memory count should be even")
	}

	if len(msg) != 0 {
		m.Diagnose = "Unhealthy"
		m.DiagnoseDetail = strings.Join(msg, "; ")
	}
}
