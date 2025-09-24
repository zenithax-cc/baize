package raid

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/pci"
)

type vrocControllerManager struct {
	ctrMap map[string]*controller // 按pci地址索引的控制器
	mutex  sync.RWMutex
	once   sync.Once
}

const procMdstat = "/proc/mdstat"

var (
	ldRegex   = regexp.MustCompile(`\[\d+\]`)
	nvmeRegex = regexp.MustCompile(`n\d+$`)
)

func intelHandle(ctx context.Context, ctrNum int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	vrocCtr := &vrocControllerManager{
		ctrMap: make(map[string]*controller, 2),
	}

	if err := vrocCtr.init(); err != nil {
		return err
	}

	if vroc, ok := vrocCtr.ctrMap[c.PCIe.PCIeAddr]; ok {
		vroc.PCIe = c.PCIe
		*c = *vroc
	}

	return nil
}

func (vcm *vrocControllerManager) init() error {
	var err error
	vcm.once.Do(func() {
		err = vcm.associateLDWithCtrl()
	})
	return err
}

func (vcm *vrocControllerManager) associateLDWithCtrl() error {
	var errs []error

	if err := vcm.getControllers(); err != nil {
		errs = append(errs, err)
	}

	lds, err := vcm.getLogicalDrives()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}

	diskToCtrMap := make(map[string]*controller)
	for _, ctr := range vcm.ctrMap {
		for _, pd := range ctr.PhysicalDrives {
			diskToCtrMap[pd.MappingFile] = ctr
		}
	}

	for _, ld := range lds {
		for _, disk := range ld.PhysicalDrives {
			vcm.mutex.RLock()
			ctr, ok := diskToCtrMap[disk.MappingFile]
			vcm.mutex.RUnlock()

			if ok {
				ld.Location = fmt.Sprintf("/c%s/%s", ctr.ID, ld.Location)

				vcm.mutex.Lock()
				ctr.LogicalDrives = append(ctr.LogicalDrives, ld)
				vcm.ctrMap[ctr.PCIe.PCIeAddr] = ctr
				vcm.mutex.Unlock()

				break // 找到控制器后可以跳出循环
			}
		}
	}

	return nil

}

func (vcm *vrocControllerManager) getControllers() error {
	output, err := utils.Run.Command(mdadm, "--detail-platform")
	if err != nil {
		return fmt.Errorf("error with mdadm finding vroc controller: %w", err)
	}

	ctrs := strings.Split(string(output), "\n\n")
	var errs []error

	for i, content := range ctrs {
		if len(strings.TrimSpace(content)) == 0 {
			continue // 跳过空内容
		}

		ctr := &controller{
			CacheSize:          "0 MB",
			CurrentPersonality: "RAID Mode",
			ID:                 strconv.Itoa(i),
			PCIe:               &pci.PCIe{},
			PhysicalDrives:     make([]*physicalDrive, 0, 8),
			LogicalDrives:      make([]*logicalDrive, 0, 4),
		}

		if err := vcm.parseController(ctr, content); err != nil {
			errs = append(errs, err)
		}

		vcm.mutex.Lock()
		if _, ok := vcm.ctrMap[ctr.PCIe.PCIeAddr]; !ok {
			vcm.ctrMap[ctr.PCIe.PCIeAddr] = ctr
		}
		vcm.mutex.Unlock()
	}

	return utils.CombineErrors(errs)

}

func (vcm *vrocControllerManager) parseController(v *controller, content string) error {
	lines := strings.Split(content, "\n")
	ctrFlag := false
	var errs []error

	for _, line := range lines {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case key == "Platform":
			v.ProductName = value
		case key == "Version":
			v.Firmware = value
		case key == "RAID Levels":
			v.RaidLevelSupported = value
		case key == "Max Disks":
			v.SupportedDrives = value
		case key == "I/O Controller" && !ctrFlag:
			ctrFlag = true
			fields := strings.Fields(value)
			if len(fields) > 0 {
				v.PCIe.PCIeAddr = filepath.Base(fields[0])
			}
		case strings.HasPrefix(key, "Port") && !strings.Contains(value, "no device attached"):
			if err := vcm.processPhysicalDrive(v, key, value); err != nil {
				errs = append(errs, err)
			}

		case key == "NVMe under VMD":
			// 处理NVMe设备
			if err := vcm.processNVMeDrive(v, value); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return utils.CombineErrors(errs)
}

func (vcm *vrocControllerManager) processPhysicalDrive(ctr *controller, location, value string) error {

	fields := strings.Fields(value)
	if len(fields) == 0 {
		return fmt.Errorf("invalid physical drive value: %s", value)
	}

	mappingFile := fields[0]
	pd := &physicalDrive{
		MappingFile: mappingFile,
		Location:    location,
	}

	// 获取SMART数据
	err := pd.getSmartctlData("vroc", "", "")

	// 存储到映射表中
	pdMapMutex.Lock()
	if _, ok := pdMap[pd.MappingFile]; !ok {
		pdMap[pd.MappingFile] = pd
	}
	pdMapMutex.Unlock()

	ctr.PhysicalDrives = append(ctr.PhysicalDrives, pd)

	return err
}

func (vcm *vrocControllerManager) processNVMeDrive(ctr *controller, value string) error {
	if strings.HasPrefix(value, "/sys/devices") {
		// 从系统路径获取NVMe设备
		return vcm.processNVMeFromSys(ctr)
	}

	// 从值字符串获取NVMe设备
	return vcm.processNVMeFromValue(ctr, value)
}

func (vcm *vrocControllerManager) processNVMeFromSys(ctr *controller) error {
	nvmePath := filepath.Join(sysBusPath, ctr.PCIe.PCIeAddr, "nvme")
	dirs, err := utils.ReadDir(nvmePath)
	if err != nil || len(dirs) == 0 {
		return fmt.Errorf("failed to read NVMe directory %s: %w", nvmePath, err)
	}

	pdMapMutex.RLock()
	pd, ok := pdMap[dirs[0].Name()]
	pdMapMutex.RUnlock()

	if ok {
		ctr.PhysicalDrives = append(ctr.PhysicalDrives, pd)
	}

	return nil
}

func (vcm *vrocControllerManager) processNVMeFromValue(ctr *controller, value string) error {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return fmt.Errorf("invalid NVMe value: %s", value)
	}

	name := fields[0]
	trimName := nvmeRegex.ReplaceAllString(name, "")

	pdMapMutex.RLock()
	pd, ok := pdMap[trimName]
	pdMapMutex.RUnlock()

	if ok {
		pd.MappingFile = name
		ctr.PhysicalDrives = append(ctr.PhysicalDrives, pd)
	}

	return nil
}

func (vcm *vrocControllerManager) getLogicalDrives() ([]*logicalDrive, error) {
	lines, err := utils.ReadLines(procMdstat)
	if err != nil {
		return nil, err
	}

	var errs []error
	lds := make([]*logicalDrive, 0, 4)
	for i, line := range lines[1:] {
		if !strings.HasPrefix(line, "md") || strings.Contains(line, "inactive") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		ld := &logicalDrive{
			Location:    "v" + strconv.Itoa(i),
			MappingFile: "/dev/" + fields[0],
		}

		if err := vcm.parseLogicalDrive(ld); err != nil {
			errs = append(errs, err)
		}

		vcm.associatePDWithLD(ld, fields[4:])

		lds = append(lds, ld)
	}
	return lds, utils.CombineErrors(errs)
}

func (vcm *vrocControllerManager) parseLogicalDrive(ld *logicalDrive) error {
	output, err := utils.Run.Command(mdadm, "-D", ld.MappingFile)
	if err != nil {
		return fmt.Errorf("error with mdadm -D %s: %w", ld.MappingFile, err)
	}

	fieldMap := map[string]*string{
		"Raid Level":         &ld.Type,
		"Total Devices":      &ld.NumberOfDrives,
		"Array Size":         &ld.Capacity,
		"State":              &ld.State,
		"Consistency Policy": &ld.Cache,
		"UUID":               &ld.ScsiNaaId}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		key, value, find := strings.Cut(line, ":")
		if !find {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if ptr, exists := fieldMap[key]; exists {
			*ptr = value
		}
	}
	return nil
}

func (vcm *vrocControllerManager) associatePDWithLD(ld *logicalDrive, fields []string) {
	for _, field := range fields {
		field = ldRegex.ReplaceAllString(field, "")
		device := "/dev/" + field
		name := device

		if strings.Contains(device, "nvme") {
			name = nvmeRegex.ReplaceAllString(device, "")
		}

		pdMapMutex.RLock()
		pd, ok := pdMap[name]
		pdMapMutex.RUnlock()

		if ok {
			ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
		}
	}
}
