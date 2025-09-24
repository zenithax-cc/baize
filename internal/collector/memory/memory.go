package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

const (
	edacPath           = "/sys/devices/system/edac/mc/"
	meminfoPath        = "/proc/meminfo"
	kbSuffix           = "kB"
	defaultMemorySlots = 24
	unknownValue       = "Unknown"
	naValue            = "N/A"
	diagnoseHealthy    = "Healthy"
	diagnoseUnhealthy  = "Unhealthy"
)

func New() *Memory {
	return &Memory{
		MemoryInfo: MemoryInfo{
			Diagnose: diagnoseHealthy,
		},
		PhysicalMemoryEntries: make([]*SmbiosMemory, 0, defaultMemorySlots),
		EdacMemoryEntries:     make([]*EdacMemory, 0, defaultMemorySlots),
	}
}

func (me *Memory) Collect(ctx context.Context) error {

	type collectTask struct {
		name string
		fn   func() error
	}

	tasks := []collectTask{{name: "meminfo", fn: me.readMeminfo},
		{name: "smbiosMemory", fn: me.SmbiosMemory},
		{name: "edacMemory", fn: me.EdacMemory},
	}

	type resultTask struct {
		name string
		err  error
	}
	resultChan := make(chan resultTask, len(tasks))
	semaphore := make(chan struct{}, 3)
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t collectTask) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				resultChan <- resultTask{name: t.name, err: ctx.Err()}
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			err := t.fn()
			resultChan <- resultTask{name: t.name, err: err}

		}(task)
	}

	wg.Wait()
	close(resultChan)

	var multiErr utils.MultiError
	for res := range resultChan {
		if res.err != nil {
			multiErr.Add(fmt.Errorf("collect %s failed: %w", res.name, res.err))
		}
	}

	me.diagnose()
	return multiErr.Unwrap()
}

func (me *Memory) readMeminfo() error {
	lines, err := utils.ReadLines(meminfoPath)
	if err != nil {
		return err
	}

	type fieldMapper struct {
		memInfoFields map[string]*string
		memValues     map[string]float64
	}

	mappers := &fieldMapper{
		memInfoFields: map[string]*string{
			"MemTotal":        &me.MemTotal,
			"MemAvailable":    &me.MemAvailable,
			"SwapTotal":       &me.SwapTotal,
			"Buffers":         &me.Buffer,
			"Cached":          &me.Cached,
			"Slab":            &me.Slab,
			"SReclaimable":    &me.SReclaimable,
			"SUnreclaim":      &me.SUnreclaim,
			"KReclaimable":    &me.KReclaimable,
			"KernelStack":     &me.KernelStack,
			"PageTables":      &me.PageTables,
			"Dirty":           &me.Dirty,
			"Writeback":       &me.Writeback,
			"HugePages_Total": &me.HPagesTotal,
			"HugePagessize":   &me.HPageSize,
			"Hugetlb":         &me.HugeTlb,
		},
		memValues: map[string]float64{
			"MemTotal":     -1,
			"MemAvailable": -1,
			"SwapTotal":    -1,
			"SwapFree":     -1,
		},
	}

	for _, line := range lines {
		key, value, ok := utils.Cut(line, ":")
		if !ok {
			continue
		}

		var valueFloat float64
		if strings.HasSuffix(value, kbSuffix) {
			trimmedValue := value[:len(value)-len(kbSuffix)]
			trimmedValue = strings.TrimSpace(trimmedValue)
			valueFloat, _ = strconv.ParseFloat(trimmedValue, 64)
		}

		if val, exists := mappers.memValues[key]; exists && val == -1 {
			mappers.memValues[key] = valueFloat
		}

		if field, exists := mappers.memInfoFields[key]; exists {
			val := convertUnitSafe(valueFloat, kbSuffix)
			*field = val
		}
	}

	me.MemUsed = calculateUsed(mappers.memValues["MemTotal"], mappers.memValues["MemAvailable"])
	me.SwapUsed = calculateUsed(mappers.memValues["SwapTotal"], mappers.memValues["SwapFree"])

	return nil
}

func convertUnitSafe(value float64, suffix string) string {
	if val, err := utils.ConvertUnit(value, suffix, true); err == nil {
		return val
	}

	return naValue
}

func calculateUsed(total, available float64) string {
	if total < 0 || available < 0 {
		return naValue
	}
	return convertUnitSafe(total-available, kbSuffix)
}

func (me *Memory) SmbiosMemory() error {
	memList, err := smbios.GetTypeData[*smbios.Type17MemoryDevice](smbios.SMBIOS, 17)
	if err != nil || len(memList) == 0 {
		return fmt.Errorf("memory device found in SMBIOS: %d,errors: %v", len(memList), err)
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

	var (
		totalSize int
		unit      string
		mutilErr  utils.MultiError
	)

	validMemory := make([]*SmbiosMemory, 0, len(memList))

	for _, mem := range memList {
		speed := speedStr(mem.Speed)
		if speed == unknownValue {
			continue
		}

		entry := &SmbiosMemory{
			BaseMemoryInfo: BaseMemoryInfo{
				Size:         mem.GetSizeString(),
				SerialNumber: mem.SerialNumber,
				Manufacturer: mem.Manufacturer,
			},
			TotalWidth:        bitWidthStr(mem.TotalWidth),
			DataWidth:         bitWidthStr(mem.DataWidth),
			FormFactor:        mem.FormFactor.String(),
			DeviceLocator:     mem.DeviceLocator,
			BankLocator:       mem.BankLocator,
			Type:              mem.Type.String(),
			TypeDetail:        mem.TypeDetail.String(),
			Speed:             speed,
			PartNumber:        mem.PartNumber,
			Rank:              mem.GetRankString(),
			ConfiguredSpeed:   speedStr(mem.ConfiguredSpeed),
			ConfiguredVoltage: voltageStr(mem.ConfiguredVoltage),
			Technology:        mem.Technology.String(),
		}

		validMemory = append(validMemory, entry)

		if size, u, ok := valiateMemorySize(entry.Size); ok {
			totalSize += size
			unit = u
		} else {
			mutilErr.Add(fmt.Errorf("invalid memory size: %s", entry.Size))
		}
	}

	me.PhysicalMemoryEntries = validMemory
	me.UsedSlots = strconv.Itoa(len(validMemory))
	me.MaximumSlots = strconv.Itoa(len(memList))
	me.PhysicalMemoryTotal = fmt.Sprintf("%d %s", totalSize, unit)

	return mutilErr.Unwrap()
}

func valiateMemorySize(size string) (int, string, bool) {
	fields := strings.Fields(size)
	if len(fields) != 2 {
		return 0, "", false
	}

	num, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, "", false
	}

	return num, fields[1], true
}

func (me *Memory) EdacMemory() error {
	if _, err := utils.ReadDir(edacPath); err != nil {
		return err
	}

	dimmDirs, err := filepath.Glob(filepath.Join(edacPath, "mc*", "dimm*"))
	if err != nil {
		return fmt.Errorf("failed to list DIMM directories: %v", err)
	}

	if len(dimmDirs) == 0 {
		return fmt.Errorf("no DIMM directories found")
	}

	type dimmRes struct {
		dimm *EdacMemory
		err  error
	}
	resChan := make(chan *dimmRes, len(dimmDirs))
	var wg sync.WaitGroup

	for _, dimmDir := range dimmDirs {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			dimm, err := parseDimmDir(dir)
			resChan <- &dimmRes{dimm: dimm, err: err}
		}(dimmDir)
	}

	wg.Wait()
	close(resChan)

	var multiErr utils.MultiError
	for res := range resChan {
		if res.err != nil {
			multiErr.Add(res.err)
		} else {
			me.EdacMemoryEntries = append(me.EdacMemoryEntries, res.dimm)
		}
	}

	return multiErr.Unwrap()
}

func parseDimmDir(dir string) (*EdacMemory, error) {
	dimm := &EdacMemory{
		DIMMID: filepath.Base(dir),
	}

	files, err := utils.ReadDir(dir)
	if err != nil {
		return dimm, fmt.Errorf("failed to read DIMM directory %s: %v", dir, err)
	}

	var dimmFileHandlers = map[string]func(*EdacMemory, string){
		"dimm_ce_count":  func(d *EdacMemory, v string) { d.CorrectableErrors = v },
		"dimm_ue_count":  func(d *EdacMemory, v string) { d.UncorrectableErrors = v },
		"dimm_dev_type":  func(d *EdacMemory, v string) { d.DeviceType = v },
		"dimm_edac_mode": func(d *EdacMemory, v string) { d.EdacMode = v },
		"dimm_location":  func(d *EdacMemory, v string) { d.MemoryLocation = v },
		"dimm_mem_type":  func(d *EdacMemory, v string) { d.MemoryType = v },
		"size":           func(d *EdacMemory, v string) { d.Size = v },
		"dimm_label":     func(d *EdacMemory, v string) { parseDimmLabel(v, d) },
	}

	var mutilErr utils.MultiError
	for _, file := range files {
		if handler, ok := dimmFileHandlers[file.Name()]; ok {
			content, err := utils.ReadOneLineFile(filepath.Join(dir, file.Name()))
			if err != nil {
				mutilErr.Add(fmt.Errorf("failed to read file %s: %v", file.Name(), err))
				continue
			}

			handler(dimm, content)
		}
	}

	return dimm, mutilErr.Unwrap()
}

func parseDimmLabel(label string, dimm *EdacMemory) {
	var labelKeyMap = map[string]func(*EdacMemory, string){
		"SrcID": func(d *EdacMemory, v string) { d.SocketID = v },
		"MC":    func(d *EdacMemory, v string) { d.MemoryControllerID = v },
		"Chan":  func(d *EdacMemory, v string) { d.ChannelID = v },
		"DIMM":  func(d *EdacMemory, v string) { d.DIMMID = v },
	}

	items := strings.Split(label, "_")
	for _, item := range items {
		if key, value, found := utils.Cut(item, "#"); found {
			if handler, ok := labelKeyMap[key]; ok {
				handler(dimm, value)
			}
		}
	}
}

func (me *Memory) diagnose() {
	sb := utils.StrBuilderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		utils.StrBuilderPool.Put(sb)
	}()

	// 检查内存数量是否为偶数
	if len(me.PhysicalMemoryEntries)%2 != 0 {
		fmt.Fprintf(sb, "memory count should be even: %s;", me.UsedSlots)
	}

	// 检查EDAC内存数量
	if len(me.EdacMemoryEntries) > 0 && len(me.EdacMemoryEntries) != len(me.PhysicalMemoryEntries) {
		fmt.Fprintf(sb, "EDAC memory count mismatch: %d vs %d;",
			len(me.EdacMemoryEntries), len(me.PhysicalMemoryEntries))
	}

	// 检查物理内存与系统内存
	if sysSize, phySize, ok := parseMemorySizes(me.MemTotal, me.PhysicalMemoryTotal); ok {
		if phySize-sysSize > 16 {
			fmt.Fprintf(sb, "physical memory exceeds system memory: %s vs %s;",
				me.PhysicalMemoryTotal, me.MemTotal)
		}
	}

	if sb.Len() > 0 {
		me.Diagnose = diagnoseUnhealthy
		me.DiagnoseDetail = sb.String()
	}
}

func parseMemorySizes(sys, phy string) (float64, float64, bool) {
	sysSize := strings.Fields(sys)
	phySize := strings.Fields(phy)

	if len(sysSize) != 2 || len(phySize) != 2 {
		return 0, 0, false
	}

	if sysSize[1] != phySize[1] {
		return 0, 0, false
	}

	sysNum, err1 := strconv.ParseFloat(sysSize[0], 64)
	phyNum, err2 := strconv.ParseFloat(phySize[0], 64)

	return sysNum, phyNum, err1 == nil && err2 == nil
}
