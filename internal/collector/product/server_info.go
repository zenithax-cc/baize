package product

import (
	"fmt"
	"strconv"

	"github.com/zenithax-cc/baize/internal/collector/smbios"
)

func collectSMBIOSType[T any](typeID smbios.TableType, typeName string, process func(T)) error {
	entries, err := smbios.GetTypeData[T](typeID)
	if err != nil {
		return fmt.Errorf("get %s from SMBIOS: %w", typeName, err)
	}

	if len(entries) <= 0 {
		return fmt.Errorf("no %s entry found in SMBIOS", typeName)
	}

	process(entries[0])

	return nil
}

func (p *Product) collectBIOS() error {
	entries, err := smbios.GetTypeData[*smbios.Type0BIOS](0)
	if err != nil {
		return fmt.Errorf("got BIOS failed: %w", err)
	}

	entry := entries[0]

	p.BIOS = BIOS{
		Version:          entry.Version,
		Vendor:           entry.Vendor,
		ReleaseDate:      entry.ReleaseDate,
		ROMSize:          entry.GetROMSize(),
		BIOSRevision:     formatRevision(entry.BIOSMajorRelease, entry.BIOSMinorRelease),
		FirmwareRevision: formatRevision(entry.ECMajorRelease, entry.ECMinorRelease),
	}

	return nil
}

func (p *Product) collectSystem() error {
	return collectSMBIOSType[*smbios.Type1System](1, "system", func(entry *smbios.Type1System) {
		p.System = System{
			Manufacturer: entry.Manufacturer,
			Version:      entry.Version,
			SerialNumber: entry.SerialNumber,
			ProductName:  entry.ProductName,
			UUID:         entry.UUID.String(),
			WakeupType:   entry.WakeUpType.String(),
			Family:       entry.Family,
		}
	})
}

func (p *Product) collectBaseBoard() error {
	return collectSMBIOSType[*smbios.Type2BaseBoard](2, "baseboard", func(entry *smbios.Type2BaseBoard) {
		p.BaseBoard = BaseBoard{
			Manufacturer: entry.Manufacturer,
			Version:      entry.Version,
			SerialNumber: entry.SerialNumber,
			ProductName:  entry.Product,
			Type:         entry.BoardType.String(),
		}
	})
}

func (p *Product) collectChassis() error {
	return collectSMBIOSType[*smbios.Type3Chassis](3, "chassis", func(entry *smbios.Type3Chassis) {
		p.Chassis = Chassis{

			Manufacturer:     entry.Manufacturer,
			SerialNumber:     entry.SerialNumber,
			Type:             entry.ChassisType.String(),
			SN:               entry.SerialNumber,
			AssetTag:         entry.AssetTag,
			BootupState:      entry.BootupState.String(),
			PowerSupplyState: entry.PowerSupplyState.String(),
			ThermalState:     entry.ThermalState.String(),
			SecurityStatus:   entry.SecurityStatus.String(),
			Height:           formatHeight(entry.Height),
			NumberOfPower:    strconv.FormatUint(uint64(entry.NumberOfPowerCords), 10),
			SKU:              entry.SKU,
		}
	})
}

func formatRevision(major, minor uint8) string {
	buf := make([]byte, 0, 8)
	buf = strconv.AppendUint(buf, uint64(major), 10)
	buf = append(buf, '.')
	buf = strconv.AppendUint(buf, uint64(minor), 10)
	return string(buf)
}

func formatHeight(height uint8) string {
	if height == 0 {
		return "Unkown"
	}

	if height <= 10 {
		return []string{
			"", "1 U", "2 U", "3 U", "4 U", "5 U",
			"6 U", "7 U", "8 U", "9 U", "10 U",
		}[height]
	}
	return strconv.FormatUint(uint64(height), 10) + " U"
}
