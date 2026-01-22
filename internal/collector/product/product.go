package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	hostNamePath      = "/proc/sys/kernel/hostname"
	ostypePath        = "/proc/sys/kernel/ostype"
	kernelReleasePath = "/proc/sys/kernel/osrelease"
	kernelVersionPath = "/proc/sys/kernel/version"
	osReleasePath     = "/etc/os-release"
	centosReleasePath = "/etc/centos-release"
	redhatReleasePath = "/etc/redhat-release"
	rockyReleasePath  = "/etc/rocky-release"
	debianVersionPath = "/etc/debian_version"

	unknownValue = "Unknown"
	naValue      = "N/A"
)

var (
	regexOnce   sync.Once
	regexUbuntu *regexp.Regexp
	regexCentos *regexp.Regexp
	regexRedhat *regexp.Regexp
	regexRocky  *regexp.Regexp
	regexDebian *regexp.Regexp

	osReleaseFieldMap = map[string]func(*OS, string){
		"PRETTY_NAME":      func(os *OS, value string) { os.PrettyName = value },
		"NAME":             func(os *OS, value string) { os.Distr = value },
		"VERSION_ID":       func(os *OS, value string) { os.DistrVersion = value },
		"VERSION_CODENAME": func(os *OS, value string) { os.CodeName = value },
		"ID_LIKE":          func(os *OS, value string) { os.IDLike = value },
	}
)

type collectFunc func() error

func initRegex() {
	regexOnce.Do(func() {
		regexUbuntu = regexp.MustCompile(`[\( ]([\d\.]+)`)
		regexCentos = regexp.MustCompile(`^CentOS( Linux)? release ([\d\.]+)`)
		regexRocky = regexp.MustCompile(`^Rocky Linux release ([\d\.]+)`)
		regexRedhat = regexp.MustCompile(`[\( ]([\d\.]+)`)
		regexDebian = regexp.MustCompile(`^([\d\.]+)`)
	})
}

func New() *Product {
	return &Product{}
}

func (p *Product) Collect(ctx context.Context) error {
	collects := []collectFunc{
		p.collectKernel,
		p.collectDistribution,
		p.collectBIOS,
		p.collectSystem,
		p.collectBaseBoard,
		p.CollectChassis,
	}

	return p.executeTasksConcurrently(ctx, collects)
}

func (p *Product) executeTasksConcurrently(ctx context.Context, tasks []collectFunc) error {
	const maxWorkers = 3

	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))
	taskChan := make(chan collectFunc, len(tasks))

	// 启动工作协程
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-taskChan:
					if !ok {
						return
					}
					if err := task(); err != nil {
						select {
						case errChan <- err:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}()
	}

	// 发送任务到通道
	go func() {
		defer close(taskChan)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case taskChan <- task:
			}
		}
	}()

	// 等待所有任务完成
	wg.Wait()
	close(errChan)

	var errs []error
	if len(errChan) > 0 {
		errs = make([]error, 0, len(errChan))
		for err := range errChan {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}

	return nil
}

func (p *Product) collectBIOS() error {
	entries, err := smbios.GetTypeData[*smbios.Type0BIOS](smbios.SMBIOS, 0)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("found %d BIOS information from SMBIO: %v", len(entries), err)
	}

	entry := entries[0]
	p.BIOS = BIOS{
		BaseInfo: BaseInfo{
			Version: entry.Version,
		},
		Vendor:           entry.Vendor,
		ReleaseDate:      entry.ReleaseDate,
		ROMSize:          entry.GetROMSize(),
		BIOSRevision:     formatRevision(entry.BIOSMajorRelease, entry.BIOSMinorRelease),
		FirmwareRevision: formatRevision(entry.ECMajorRelease, entry.ECMinorRelease),
	}
	return nil
}

func (p *Product) collectSystem() error {
	entries, err := smbios.GetTypeData[*smbios.Type1System](smbios.SMBIOS, 1)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("found %d system information from SMBIO: %v", len(entries), err)
	}

	entry := entries[0]
	p.System = System{
		BaseInfo: BaseInfo{
			Manufacturer: entry.Manufacturer,
			Version:      entry.Version,
			SerialNumber: entry.SerialNumber,
		},
		ProductName: entry.ProductName,
		UUID:        entry.UUID.String(),
		WakeupType:  entry.WakeUpType.String(),
		Family:      entry.Family,
	}
	return nil
}

func (p *Product) collectBaseBoard() error {
	entries, err := smbios.GetTypeData[*smbios.Type2BaseBoard](smbios.SMBIOS, 2)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("found %d baseboard information from SMBIO: %v", len(entries), err)
	}

	entry := entries[0]
	p.BaseBoard = BaseBoard{
		BaseInfo: BaseInfo{
			Manufacturer: entry.Manufacturer,
			Version:      entry.Version,
			SerialNumber: entry.SerialNumber,
		},
		ProductName: entry.Product,
		Type:        entry.BoardType.String(),
	}
	return nil
}

func (p *Product) CollectChassis() error {
	entries, err := smbios.GetTypeData[*smbios.Type3Chassis](smbios.SMBIOS, 3)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("found %d chassis information from SMBIO: %v", len(entries), err)
	}

	entry := entries[0]
	p.Chassis = Chassis{
		BaseInfo: BaseInfo{
			Manufacturer: entry.Manufacturer,
			SerialNumber: entry.SerialNumber,
		},
		Type:             entry.ChassisType.String(),
		SN:               entry.SerialNumber,
		AssetTag:         entry.AssetTag,
		BootupState:      entry.BootupState.String(),
		PowerSupplyState: entry.PowerSupplyState.String(),
		ThermalState:     entry.ThermalState.String(),
		SecurityStatus:   entry.SecurityStatus.String(),
		Height:           formatHeight(entry.Height),
		NumberOfPower:    strconv.Itoa(int(entry.NumberOfPowerCords)),
		SKU:              entry.SKU,
	}
	return nil
}

func formatRevision(major, minor uint8) string {
	return fmt.Sprintf("%d.%d", major, minor)
}

func formatHeight(height uint8) string {
	if height == 0 {
		return unknownValue
	}
	return fmt.Sprintf("%d U", height)
}

func (p *Product) collectKernel() error {
	type kernekCfg struct {
		path   string
		target *string
	}

	kernelCfgs := []kernekCfg{
		{path: ostypePath, target: &p.OS.KernelName},
		{path: kernelReleasePath, target: &p.OS.KernelRelease},
		{path: kernelVersionPath, target: &p.OS.KernelVersion},
		{path: hostNamePath, target: &p.OS.HostName},
	}

	var multiErr utils.MultiError
	for _, cfg := range kernelCfgs {
		if content, err := utils.ReadOneLineFile(cfg.path); err != nil {
			multiErr.Add(err)
		} else {
			*cfg.target = content
		}
	}

	return multiErr.Unwrap()
}

func (p *Product) collectDistribution() error {
	lines, err := utils.ReadLines(osReleasePath)
	if err != nil || len(lines) == 0 {
		return fmt.Errorf("found %d information from os-release: %v", len(lines), err)
	}

	for _, line := range lines {
		if key, value, found := utils.Cut(line, "="); found {
			cleanValue := strings.Trim(value, "\"")
			if fn, ok := osReleaseFieldMap[key]; ok {
				fn(&p.OS, cleanValue)
			}
		}
	}

	p.OS.MinorVersion = getMinorVersion(p.OS.Distr)

	return nil
}

func getMinorVersion(distr string) string {
	lowerDistr := strings.ToLower(distr)

	var strategyMap = map[string]versionStrategy{
		"debian": debianStrategy{},
		"ubuntu": ubuntuStrategy{},
		"centos": centosStrategy{},
		"rhel":   rhelStrategy{},
		"rocky":  rockyStrategy{},
	}

	for distr, strategy := range strategyMap {
		if strings.HasPrefix(lowerDistr, distr) {
			return strategy.getMinorVersion(distr)
		}
	}

	return unknownValue
}

type versionStrategy interface {
	getMinorVersion(distr string) string
}

type debianStrategy struct{}
type ubuntuStrategy struct{}
type centosStrategy struct{}
type rhelStrategy struct{}
type rockyStrategy struct{}

func (s debianStrategy) getMinorVersion(distr string) string {
	if content, err := utils.ReadOneLineFile(debianVersionPath); err == nil {
		initRegex()
		if match := regexDebian.FindStringSubmatch(content); len(match) > 1 {
			return match[1]
		}
	}

	return unknownValue
}

func (s ubuntuStrategy) getMinorVersion(distr string) string {
	initRegex()
	if matches := regexUbuntu.FindStringSubmatch(distr); len(matches) > 1 {
		return matches[1]
	}
	return unknownValue
}

func (s centosStrategy) getMinorVersion(distr string) string {
	if content, err := utils.ReadOneLineFile(centosReleasePath); err == nil {
		initRegex()
		if matches := regexCentos.FindStringSubmatch(content); len(matches) > 2 {
			return matches[2]
		}
	}
	return unknownValue
}

func (s rhelStrategy) getMinorVersion(distr string) string {
	if content, err := utils.ReadOneLineFile(redhatReleasePath); err == nil {
		initRegex()
		if matches := regexRedhat.FindStringSubmatch(content); len(matches) > 1 {
			return matches[1]
		}
	}
	return unknownValue
}

func (s rockyStrategy) getMinorVersion(distr string) string {
	if content, err := utils.ReadOneLineFile(rockyReleasePath); err == nil {
		initRegex()
		if matches := regexRocky.FindStringSubmatch(content); len(matches) > 1 {
			return matches[1]
		}
	}
	return unknownValue
}
