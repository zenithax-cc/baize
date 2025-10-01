package raid

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/pci"
)

type intelController struct {
	controllerMap sync.Map // 按pci地址索引的控制器
	once          sync.Once
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

	ic := &intelController{}

	var multiErr utils.MultiError

	multiErr.Add(ic.loadControllers(ctx))

	if ctr, ok := ic.controllerMap.Load(c.PCIe.PCIeAddr); ok {
		cc := ctr.(*controller)
		cc.PCIe = c.PCIe
		*c = *cc
	}

	return multiErr.Unwrap()
}

func (ic *intelController) loadControllers(ctx context.Context) error {
	var err error

	ic.once.Do(func() {
		err = ic.associateLDWithCtrl(ctx)
	})

	return err
}

func (ic *intelController) associateLDWithCtrl(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var multiErr utils.MultiError
	var wg sync.WaitGroup
	var lds []*logicalDrive
	var err error
	wg.Add(2)

	go func() {
		defer wg.Done()
		multiErr.Add(ic.processControllers(ctx))
	}()

	go func() {
		defer wg.Done()
		lds, err = ic.processLogicalDrives(ctx)
		multiErr.Add(err)
	}()

	wg.Wait()

	if err := multiErr.Unwrap(); err != nil {
		return err
	}

	diskToCtrMap := ic.buildDiskToCtrMap()

	ic.associateLogicalDrives(lds, diskToCtrMap)

	return nil
}

func (ic *intelController) buildDiskToCtrMap() map[string]*controller {
	diskToCtrMap := make(map[string]*controller)
	ic.controllerMap.Range(func(key, value interface{}) bool {
		if ctr, ok := value.(*controller); ok {
			for _, pd := range ctr.PhysicalDrives {
				diskToCtrMap[pd.MappingFile] = ctr
			}
		}
		return true
	})

	return diskToCtrMap
}

func (ic *intelController) associateLogicalDrives(lds []*logicalDrive, diskToCtrMap map[string]*controller) {
	ctrToLDsMap := make(map[*controller][]*logicalDrive)

	for _, ld := range lds {
		for _, pd := range ld.PhysicalDrives {
			if ctr, ok := diskToCtrMap[pd.MappingFile]; ok {
				ld.Location = fmt.Sprintf("/c%s/v%s", ctr.ID, ld.Location)
				ctr.LogicalDrives = append(ctr.LogicalDrives, ld)
				ctrToLDsMap[ctr] = append(ctrToLDsMap[ctr], ld)
				break
			}
		}
	}

	for ctr, lds := range ctrToLDsMap {
		ctr.LogicalDrives = append(ctr.LogicalDrives, lds...)
		ic.controllerMap.Store(ctr.PCIe.PCIeAddr, ctr)
	}

}

func (ic *intelController) processControllers(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	output, err := utils.Run.Command(mdadmPath, "--detail-platform")
	if err != nil {
		return fmt.Errorf("error with mdadm finding vroc controller: %w", err)
	}

	ctrs := strings.Split(string(output), "\n\n")
	var wg sync.WaitGroup
	var multiErr utils.MultiError

	for i, content := range ctrs {
		if len(strings.TrimSpace(content)) == 0 {
			continue // 跳过空内容
		}

		wg.Add(1)
		go func(index int, content string) {
			defer wg.Done()

			ctr := &controller{
				CacheSize:          "0 MB",
				CurrentPersonality: "RAID Mode",
				ID:                 strconv.Itoa(index),
				PCIe:               &pci.PCIe{},
				PhysicalDrives:     make([]*physicalDrive, 0, 8),
				LogicalDrives:      make([]*logicalDrive, 0, 4),
			}

			multiErr.Add(ic.parseController(ctr, content))

			ic.controllerMap.Store(ctr.PCIe.PCIeAddr, ctr)
		}(i, content)
	}

	wg.Wait()

	return multiErr.Unwrap()
}

var ctrHandler = map[string]func(*controller, string){
	"Platform":    func(v *controller, value string) { v.ProductName = value },
	"Version":     func(v *controller, value string) { v.Firmware = value },
	"RAID Levels": func(v *controller, value string) { v.RaidLevelSupported = value },
	"Max Disks":   func(v *controller, value string) { v.SupportedDrives = value },
}

func (ic *intelController) parseController(v *controller, content string) error {
	var multiErr utils.MultiError

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		key, value, found := utils.Cut(line, ":")
		if !found {
			continue
		}

		if handler, ok := ctrHandler[key]; ok {
			handler(v, value)
			continue
		}

		if strings.HasPrefix(key, "Port") && !strings.Contains(value, "no device attached") {
			multiErr.Add(ic.procesSataDrive(v, key, value))
			continue
		}

		if key == "NVMe under VMD" {
			multiErr.Add(ic.processNVMeDrive(v, value))
			continue
		}

		if key == "I/O Controller" {
			if len(v.PCIe.PCIeAddr) == 0 {
				fields := strings.Fields(value)
				v.PCIe.PCIeAddr = filepath.Base(strings.TrimSpace(fields[0]))
			}
		}
	}

	return multiErr.Unwrap()
}

func (ic *intelController) procesSataDrive(ctr *controller, part, value string) error {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return fmt.Errorf("invalid SATA drive value: %s", value)
	}

	pd := &physicalDrive{
		MappingFile: strings.TrimSpace(fields[0]),
		Location:    part,
	}

	err := pd.getSmartctlData("vroc", "", "")
	pdc.Set(pd.MappingFile, pd)

	ctr.PhysicalDrives = append(ctr.PhysicalDrives, pd)
	return err
}

func (ic *intelController) processNVMeDrive(ctr *controller, value string) error {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return fmt.Errorf("invalid NVMe drive value: %s", value)
	}

	device := strings.TrimSpace(fields[0])
	mappingFile := device

	if strings.HasPrefix(device, "/sys/devices") {
		dev, err := GetBlockByDevicesPath(device)
		if err != nil {
			return fmt.Errorf("get block by devices path error: %w", err)
		}

		mappingFile = dev
		device = dev

		if strings.Contains(device, "nvme") {
			device = nvmeRegex.ReplaceAllString(device, "")
		}
	}

	if nvme, ok := pdc.Get(device); ok {
		nvme.MappingFile = mappingFile
		ctr.PhysicalDrives = append(ctr.PhysicalDrives, nvme)
	}

	return nil
}

func (ic *intelController) processLogicalDrives(ctx context.Context) ([]*logicalDrive, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(procMdstat)
	if err != nil {
		return nil, fmt.Errorf("read %s error: %w", procMdstat, err)
	}

	var num int
	var multiErr utils.MultiError
	result := make([]*logicalDrive, 0, 4)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "md") || strings.Contains(line, "inactive") {
			continue
		}

		fields := strings.Fields(line)

		if len(fields) < 5 {
			continue
		}

		ld := &logicalDrive{
			Location:    "v" + strconv.Itoa(num),
			MappingFile: "/dev/" + fields[0],
		}

		multiErr.Add(parseLogicalDrive(ld))
		associatePDWithLD(ld, fields[4:])
		result = append(result, ld)
		num++
	}

	return result, multiErr.Unwrap()
}

var parseLDFields = map[string]func(*logicalDrive, string){
	"Raid Level":         func(v *logicalDrive, value string) { v.Type = value },
	"Total Devices":      func(v *logicalDrive, value string) { v.NumberOfDrives = value },
	"Array Size":         func(v *logicalDrive, value string) { v.Capacity = value },
	"State":              func(v *logicalDrive, value string) { v.State = value },
	"Consistency Policy": func(v *logicalDrive, value string) { v.Cache = value },
	"UUID":               func(v *logicalDrive, value string) { v.ScsiNaaId = value },
}

func parseLogicalDrive(ld *logicalDrive) error {
	output, err := utils.Run.Command(mdadmPath, "-D", ld.MappingFile)
	if err != nil {
		return fmt.Errorf("error with mdadm -D %s: %w", ld.MappingFile, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}

		if handler, ok := parseLDFields[key]; ok {
			handler(ld, value)
		}
	}

	return scanner.Err()
}

func associatePDWithLD(ld *logicalDrive, fields []string) {
	for _, field := range fields {
		field = ldRegex.ReplaceAllString(field, "")
		device := "/dev/" + field
		name := device

		if strings.Contains(device, "nvme") {
			name = nvmeRegex.ReplaceAllString(device, "")
		}

		if pd, ok := pdc.Get(name); ok {
			ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
		}
	}
}
