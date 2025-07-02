package product

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

type OS struct {
	KernelName    string `json:"kernel_name,omitempty"`
	KernelRelease string `json:"kernel_release,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
	HostName      string `json:"host_name,omitempty"`
	PrettyName    string `json:"pretty_name,omitempty"`
	Releases      string `json:"releases,omitempty"`
	DistrVersion  string `json:"distr_version,omitempty"`
	MinorVersion  string `json:"minor_version,omitempty"`
	IDLike        string `json:"id_like,omitempty"`
	CodeName      string `json:"code_name,omitempty"`
	Distr         string `json:"distr,omitempty"`
}

type BIOS struct {
	Vendor           string `json:"vendor,omitempty"`
	Version          string `json:"version,omitempty"`
	ReleaseDate      string `json:"release_date,omitempty"`
	ROMSize          string `json:"rom_size,omitempty"`
	BIOSRevision     string `json:"bios_revision,omitempty"`
	FirmwareRevision string `json:"firmware_revision,omitempty"`
}

type System struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	ProductName  string `json:"product_name,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	UUID         string `json:"uuid,omitempty"`
	WakeupType   string `json:"wake-up_type,omitempty"`
	Family       string `json:"family,omitempty"`
}

type BaseBoard struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	ProductName  string `json:"product_name,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Type         string `json:"type,omitempty"`
}

type Chassis struct {
	Manufacturer     string `json:"manufacturer,omitempty"`
	Type             string `json:"type,omitempty"`
	SN               string `json:"sn,omitempty"`
	AssetTag         string `json:"asset_tag,omitempty"`
	BootupState      string `json:"bootup_state,omitempty"`
	PowerSupplyState string `json:"power_supply_state,omitempty"`
	ThermalState     string `json:"thermal_state,omitempty"`
	SecurityStatus   string `json:"security_status,omitempty"`
	Height           string `json:"height,omitempty"`
	NumberOfPower    string `json:"number_of_power_cards,omitempty"`
	SKU              string `json:"sku_number,omitempty"`
}

type Product struct {
	OS        `json:"os,omitempty"`
	BIOS      `json:"bios,omitempty"`
	System    `json:"system,omitempty"`
	BaseBoard `json:"base_board,omitempty"`
	Chassis   `json:"chassis,omitempty"`
}

const (
	hostName      = "/proc/sys/kernel/hostname"
	ostype        = "/proc/sys/kernel/ostype"
	kernelRelease = "/proc/sys/kernel/osrelease"
	kernelVersion = "/proc/sys/kernel/version"
	osRelease     = "/etc/os-release"
	centosRelease = "/etc/centos-release"
	redhatRelease = "/etc/redhat-release"
	rockyRelease  = "/etc/rocky-release"
)

func New() *Product {
	return &Product{
		OS:        OS{},
		BIOS:      BIOS{},
		System:    System{},
		BaseBoard: BaseBoard{},
		Chassis:   Chassis{},
	}
}

func (p *Product) Collect() error {
	var errs []error

	if err := p.kernel(); err != nil {
		errs = append(errs, err)
	}

	if err := p.distribution(); err != nil {
		errs = append(errs, err)
	}

	if err := p.bios(); err != nil {
		errs = append(errs, err)
	}

	if err := p.system(); err != nil {
		errs = append(errs, err)
	}

	if err := p.baseBoard(); err != nil {
		errs = append(errs, err)
	}

	if err := p.chassis(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}

	return nil
}

func (p *Product) bios() error {
	entries, err := smbios.GetTypeData[*smbios.Type0BIOS](smbios.SMBIOS, 0)
	if entries == nil && err != nil {
		return fmt.Errorf("no bios information found in SMBIOS : %v", err)
	}

	b := entries[0] // only one bios entry
	p.BIOS = BIOS{
		Vendor:           b.Vendor,
		Version:          b.Version,
		ReleaseDate:      b.ReleaseDate,
		ROMSize:          b.GetROMSize(),
		BIOSRevision:     fmt.Sprintf("%d.%d", b.BIOSMajorRelease, b.BIOSMinorRelease),
		FirmwareRevision: fmt.Sprintf("%d.%d", b.ECMajorRelease, b.ECMinorRelease),
	}

	return nil
}

func (p *Product) system() error {
	entries, err := smbios.GetTypeData[*smbios.Type1System](smbios.SMBIOS, 1)
	if entries == nil {
		return fmt.Errorf("no system information found in SMBIOS: %v", err)
	}

	s := entries[0] // only one system entry
	p.System = System{
		Manufacturer: s.Manufacturer,
		ProductName:  s.ProductName,
		Version:      s.Version,
		SerialNumber: s.SerialNumber,
		UUID:         s.UUID.String(),
		WakeupType:   s.WakeUpType.String(),
		Family:       s.Family,
	}

	return nil
}

func (p *Product) baseBoard() error {
	entries, err := smbios.GetTypeData[*smbios.Type2BaseBoard](smbios.SMBIOS, 2)
	if entries == nil {
		return fmt.Errorf("no baseboard information found in SMBIOS : %v", err)
	}

	b := entries[0] // only one baseboard entry
	p.BaseBoard = BaseBoard{
		Manufacturer: b.Manufacturer,
		ProductName:  b.Product,
		Version:      b.Version,
		SerialNumber: b.SerialNumber,
		Type:         b.BoardType.String(),
	}

	return nil
}

func (p *Product) chassis() error {
	entries, err := smbios.GetTypeData[*smbios.Type3Chassis](smbios.SMBIOS, 3)
	if entries == nil {
		return fmt.Errorf("no chassis information found in SMBIOS : %v", err)
	}

	cha := entries[0] // only one chassis entry
	p.Chassis = Chassis{
		Manufacturer:     cha.Manufacturer,
		Type:             cha.ChassisType.String(),
		SN:               cha.SerialNumber,
		AssetTag:         cha.AssetTag,
		BootupState:      cha.BootupState.String(),
		PowerSupplyState: cha.PowerSupplyState.String(),
		ThermalState:     cha.ThermalState.String(),
		SecurityStatus:   cha.SecurityStatus.String(),
		Height:           fmt.Sprintf("%d U", cha.Height),
		NumberOfPower:    strconv.Itoa(int(cha.NumberOfPowerCords)),
		SKU:              cha.SKU,
	}
	return nil
}

func (p *Product) kernel() error {
	var errs []error
	handler := func(path string) string {
		content, err := utils.ReadOneLineFile(path)
		if err != nil {
			errs = append(errs, err)
			return ""
		}
		return content
	}

	p.OS.HostName = handler(hostName)
	p.OS.KernelName = handler(ostype)
	p.OS.KernelRelease = handler(kernelRelease)
	p.OS.KernelVersion = handler(kernelVersion)

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}
	return nil
}

func (p *Product) distribution() error {
	lines, err := utils.ReadLines(osRelease)
	if err != nil {
		return err
	}
	for _, line := range lines {
		key, value, found := utils.Cut(line, "=")
		if !found {
			continue
		}
		value = strings.Trim(value, "\"")
		switch key {
		case "PRETTY_NAME":
			p.OS.PrettyName = value
		case "NAME":
			p.OS.Distr = value
		case "VERSION_ID":
			p.OS.DistrVersion = value
		case "VERSION_CODENAME":
			p.OS.CodeName = value
		case "ID_LIKE":
			p.OS.IDLike = value
		default:
			continue
		}
	}

	p.OS.MinorVersion = getMinorVersion(p.OS.Distr)

	return nil
}

func getMinorVersion(distr string) string {
	res := "Unknown"
	distr = strings.ToLower(distr)

	var (
		reUbuntu = regexp.MustCompile(`[\( ]([\d\.]+)`)
		reCentOS = regexp.MustCompile(`^CentOS( Linux)? release ([\d\.]+)`)
		reRocky  = regexp.MustCompile(`^Rocky Linux release ([\d\.]+)`)
		reRedHat = regexp.MustCompile(`[\( ]([\d\.]+)`)
	)

	switch {
	case strings.HasPrefix(distr, "debian"):
		res, _ = utils.ReadOneLineFile("/etc/debian_version")
	case strings.HasPrefix(distr, "ubuntu"):
		if m := reUbuntu.FindStringSubmatch(distr); m != nil {
			res = m[1]
		}
	case strings.HasPrefix(distr, "centos"):
		if data, err := utils.ReadOneLineFile("/etc/centos-release"); data != "" && err == nil {
			if m := reCentOS.FindStringSubmatch(data); m != nil {
				res = m[2]
			}
		}
	case strings.HasPrefix(distr, "rhel"):
		if data, err := utils.ReadOneLineFile("/etc/redhat-release"); data != "" && err == nil {
			if m := reRedHat.FindStringSubmatch(data); m != nil {
				res = m[1]
			}
		}
	case strings.HasPrefix(distr, "rocky"):
		if data, err := utils.ReadOneLineFile("/etc/rocky-release"); data != "" && err == nil {
			if m := reRocky.FindStringSubmatch(data); m != nil {
				res = m[1]
			}
		}
	}

	return res
}
