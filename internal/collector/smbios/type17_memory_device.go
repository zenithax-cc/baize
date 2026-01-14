package smbios

import (
	"fmt"
	"strings"
)

type Type17MemoryDevice struct {
	Header                            `smbios:"-"`
	PhysicalMemoryArrayHandle         uint16                              // 04h
	MemoryErrorInfoHandle             uint16                              // 06h
	TotalWidth                        uint16                              // 08h
	DataWidth                         uint16                              // 0Ah
	Size                              uint16                              // 0Ch
	FormFactor                        MemoryDeviceFormFactor              // 0Eh
	DeviceSet                         uint8                               // 0Fh
	DeviceLocator                     string                              // 10h
	BankLocator                       string                              // 11h
	Type                              MemoryDeviceType                    // 12h
	TypeDetail                        MemoryDeviceTypeDetail              // 13h
	Speed                             uint16                              // 15h
	Manufacturer                      string                              // 17h
	SerialNumber                      string                              // 18h
	AssetTag                          string                              // 19h
	PartNumber                        string                              // 1Ah
	Attributes                        uint8                               // 1Bh
	ExtendedSize                      uint32                              // 1Ch
	ConfiguredSpeed                   uint16                              // 20h
	MinimumVoltage                    uint16                              // 22h
	MaximumVoltage                    uint16                              // 24h
	ConfiguredVoltage                 uint16                              // 26h
	Technology                        MemoryDeviceTechnology              // 28h
	OperatingModeCapability           MemoryDeviceOperatingModeCapability // 29h
	FirmwareVersion                   string                              // 2Bh
	ModuleManufacturerID              uint16                              // 2Ch
	ModuleProductID                   uint16                              // 2Eh
	SubsystemControllerManufacturerID uint16                              // 30h
	SubsystemControllerProductID      uint16                              // 32h
	NonvolatileSize                   uint64                              // 34h
	VolatileSize                      uint64                              // 3Ch
	CacheSize                         uint64                              // 44h
	LogicalSize                       uint64                              // 4Ch
	ExtendedSpeed                     uint32                              // 54h
	ExtendedConfiguredSpeed           uint32                              // 58h
	PMICManufacturerID                uint16                              // 5Ch
	PMICRevisionNumber                uint16                              // 5Eh
	RCDManufacturerID                 uint16                              // 60h
	RCDRevisionNumber                 uint16                              // 62h
}

func parseType17MemoryDevice(t *Table) (*Type17MemoryDevice, error) {
	if t.Header.Type != 17 {
		return nil, fmt.Errorf("%s:%d", ErrInvalidTableType, t.Header.Type)
	}

	if t.Header.Length < 0x15 {
		return nil, fmt.Errorf("%s: memory device table must be at least %d bytes", ErrInvalidTableLength, 0x15)
	}

	md := &Type17MemoryDevice{
		Header: t.Header,
	}

	if _, err := parseType(t, 0, false, md); err != nil {
		return nil, fmt.Errorf("failed to parse Type 17 Memory Device: %w", err)
	}

	return md, nil
}

func (t *Type17MemoryDevice) GetSizeString() string {
	switch t.Size {
	case 0:
		return "No Module Installed"
	case 0xFFFF:
		return "Unknown"
	case 0x7FFF:
		return kgmt(uint64(t.ExtendedSize&0x7FFFFFFF) * 1024 * 1024)
	default:
		mul := uint64(1024 * 1024)
		if t.Size&0x8000 != 0 {
			mul = 1024
		}
		return kgmt(uint64(t.Size&0x7FFF) * mul)
	}
}

func (t *Type17MemoryDevice) GetRankString() string {
	rankStr := "Unknown"
	if t.Attributes&0x0F != 0 {
		rankStr = fmt.Sprintf("%d", t.Attributes&0x0F)
	}
	return rankStr
}

type MemoryDeviceFormFactor uint8

const (
	MemoryDeviceFormFactorOther           MemoryDeviceFormFactor = 0x01 // Other
	MemoryDeviceFormFactorUnknown         MemoryDeviceFormFactor = 0x02 // Unknown
	MemoryDeviceFormFactorSIMM            MemoryDeviceFormFactor = 0x03 // SIMM
	MemoryDeviceFormFactorSIP             MemoryDeviceFormFactor = 0x04 // SIP
	MemoryDeviceFormFactorChip            MemoryDeviceFormFactor = 0x05 // Chip
	MemoryDeviceFormFactorDIP             MemoryDeviceFormFactor = 0x06 // DIP
	MemoryDeviceFormFactorZIP             MemoryDeviceFormFactor = 0x07 // ZIP
	MemoryDeviceFormFactorProprietaryCard MemoryDeviceFormFactor = 0x08 // Proprietary Card
	MemoryDeviceFormFactorDIMM            MemoryDeviceFormFactor = 0x09 // DIMM
	MemoryDeviceFormFactorTSOP            MemoryDeviceFormFactor = 0x0a // TSOP
	MemoryDeviceFormFactorRowOfChips      MemoryDeviceFormFactor = 0x0b // Row of chips
	MemoryDeviceFormFactorRIMM            MemoryDeviceFormFactor = 0x0c // RIMM
	MemoryDeviceFormFactorSODIMM          MemoryDeviceFormFactor = 0x0d // SODIMM
	MemoryDeviceFormFactorSRIMM           MemoryDeviceFormFactor = 0x0e // SRIMM
	MemoryDeviceFormFactorFBDIMM          MemoryDeviceFormFactor = 0x0f // FB-DIMM
)

var memoryDeviceFormFactorNames = map[MemoryDeviceFormFactor]string{
	MemoryDeviceFormFactorOther:           "Other",
	MemoryDeviceFormFactorUnknown:         "Unknown",
	MemoryDeviceFormFactorSIMM:            "SIMM",
	MemoryDeviceFormFactorSIP:             "SIP",
	MemoryDeviceFormFactorChip:            "Chip",
	MemoryDeviceFormFactorDIP:             "DIP",
	MemoryDeviceFormFactorZIP:             "ZIP",
	MemoryDeviceFormFactorProprietaryCard: "Proprietary Card",
	MemoryDeviceFormFactorDIMM:            "DIMM",
	MemoryDeviceFormFactorTSOP:            "TSOP",
	MemoryDeviceFormFactorRowOfChips:      "Row of chips",
	MemoryDeviceFormFactorRIMM:            "RIMM",
	MemoryDeviceFormFactorSODIMM:          "SODIMM",
	MemoryDeviceFormFactorSRIMM:           "SRIMM",
	MemoryDeviceFormFactorFBDIMM:          "FB-DIMM",
}

func (v MemoryDeviceFormFactor) String() string {
	if name, ok := memoryDeviceFormFactorNames[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type MemoryDeviceType uint8

const (
	MemoryDeviceTypeOther                    MemoryDeviceType = 0x01 // Other
	MemoryDeviceTypeUnknown                  MemoryDeviceType = 0x02 // Unknown
	MemoryDeviceTypeDRAM                     MemoryDeviceType = 0x03 // DRAM
	MemoryDeviceTypeEDRAM                    MemoryDeviceType = 0x04 // EDRAM
	MemoryDeviceTypeVRAM                     MemoryDeviceType = 0x05 // VRAM
	MemoryDeviceTypeSRAM                     MemoryDeviceType = 0x06 // SRAM
	MemoryDeviceTypeRAM                      MemoryDeviceType = 0x07 // RAM
	MemoryDeviceTypeROM                      MemoryDeviceType = 0x08 // ROM
	MemoryDeviceTypeFlash                    MemoryDeviceType = 0x09 // Flash
	MemoryDeviceTypeEEPROM                   MemoryDeviceType = 0x0a // EEPROM
	MemoryDeviceTypeFEPROM                   MemoryDeviceType = 0x0b // FEPROM
	MemoryDeviceTypeEPROM                    MemoryDeviceType = 0x0c // EPROM
	MemoryDeviceTypeCDRAM                    MemoryDeviceType = 0x0d // CDRAM
	MemoryDeviceType3DRAM                    MemoryDeviceType = 0x0e // 3DRAM
	MemoryDeviceTypeSDRAM                    MemoryDeviceType = 0x0f // SDRAM
	MemoryDeviceTypeSGRAM                    MemoryDeviceType = 0x10 // SGRAM
	MemoryDeviceTypeRDRAM                    MemoryDeviceType = 0x11 // RDRAM
	MemoryDeviceTypeDDR                      MemoryDeviceType = 0x12 // DDR
	MemoryDeviceTypeDDR2                     MemoryDeviceType = 0x13 // DDR2
	MemoryDeviceTypeDDR2FBDIMM               MemoryDeviceType = 0x14 // DDR2 FB-DIMM
	MemoryDeviceTypeDDR3                     MemoryDeviceType = 0x18 // DDR3
	MemoryDeviceTypeFBD2                     MemoryDeviceType = 0x19 // FBD2
	MemoryDeviceTypeDDR4                     MemoryDeviceType = 0x1a // DDR4
	MemoryDeviceTypeLPDDR                    MemoryDeviceType = 0x1b // LPDDR
	MemoryDeviceTypeLPDDR2                   MemoryDeviceType = 0x1c // LPDDR2
	MemoryDeviceTypeLPDDR3                   MemoryDeviceType = 0x1d // LPDDR3
	MemoryDeviceTypeLPDDR4                   MemoryDeviceType = 0x1e // LPDDR4
	MemoryDeviceTypeLogicalNonvolatileDevice MemoryDeviceType = 0x1f // Logical non-volatile device
	MemoryDeviceTypeDDR5                     MemoryDeviceType = 0x22 // DDR5
)

var memoryDeviceTypeNames = map[MemoryDeviceType]string{
	MemoryDeviceTypeOther:                    "Other",
	MemoryDeviceTypeUnknown:                  "Unknown",
	MemoryDeviceTypeDRAM:                     "DRAM",
	MemoryDeviceTypeEDRAM:                    "EDRAM",
	MemoryDeviceTypeVRAM:                     "VRAM",
	MemoryDeviceTypeSRAM:                     "SRAM",
	MemoryDeviceTypeRAM:                      "RAM",
	MemoryDeviceTypeROM:                      "ROM",
	MemoryDeviceTypeFlash:                    "Flash",
	MemoryDeviceTypeEEPROM:                   "EEPROM",
	MemoryDeviceTypeFEPROM:                   "FEPROM",
	MemoryDeviceTypeEPROM:                    "EPROM",
	MemoryDeviceTypeCDRAM:                    "CDRAM",
	MemoryDeviceType3DRAM:                    "3DRAM",
	MemoryDeviceTypeSDRAM:                    "SDRAM",
	MemoryDeviceTypeSGRAM:                    "SGRAM",
	MemoryDeviceTypeRDRAM:                    "RDRAM",
	MemoryDeviceTypeDDR:                      "DDR",
	MemoryDeviceTypeDDR2:                     "DDR2",
	MemoryDeviceTypeDDR2FBDIMM:               "DDR2 FB-DIMM",
	MemoryDeviceTypeDDR3:                     "DDR3",
	MemoryDeviceTypeFBD2:                     "FBD2",
	MemoryDeviceTypeDDR4:                     "DDR4",
	MemoryDeviceTypeLPDDR:                    "LPDDR",
	MemoryDeviceTypeLPDDR2:                   "LPDDR2",
	MemoryDeviceTypeLPDDR3:                   "LPDDR3",
	MemoryDeviceTypeLPDDR4:                   "LPDDR4",
	MemoryDeviceTypeLogicalNonvolatileDevice: "Logical non-volatile device",
	MemoryDeviceTypeDDR5:                     "DDR5",
}

func (v MemoryDeviceType) String() string {
	if name, ok := memoryDeviceTypeNames[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type MemoryDeviceTypeDetail uint16

const (
	MemoryDeviceTypeDetailOther                  MemoryDeviceTypeDetail = 1 << 1  // Other
	MemoryDeviceTypeDetailUnknown                MemoryDeviceTypeDetail = 1 << 2  // Unknown
	MemoryDeviceTypeDetailFastpaged              MemoryDeviceTypeDetail = 1 << 3  // Fast-paged
	MemoryDeviceTypeDetailStaticColumn           MemoryDeviceTypeDetail = 1 << 4  // Static column
	MemoryDeviceTypeDetailPseudostatic           MemoryDeviceTypeDetail = 1 << 5  // Pseudo-static
	MemoryDeviceTypeDetailRAMBUS                 MemoryDeviceTypeDetail = 1 << 6  // RAMBUS
	MemoryDeviceTypeDetailSynchronous            MemoryDeviceTypeDetail = 1 << 7  // Synchronous
	MemoryDeviceTypeDetailCMOS                   MemoryDeviceTypeDetail = 1 << 8  // CMOS
	MemoryDeviceTypeDetailEDO                    MemoryDeviceTypeDetail = 1 << 9  // EDO
	MemoryDeviceTypeDetailWindowDRAM             MemoryDeviceTypeDetail = 1 << 10 // Window DRAM
	MemoryDeviceTypeDetailCacheDRAM              MemoryDeviceTypeDetail = 1 << 11 // Cache DRAM
	MemoryDeviceTypeDetailNonvolatile            MemoryDeviceTypeDetail = 1 << 12 // Non-volatile
	MemoryDeviceTypeDetailRegisteredBuffered     MemoryDeviceTypeDetail = 1 << 13 // Registered (Buffered)
	MemoryDeviceTypeDetailUnbufferedUnregistered MemoryDeviceTypeDetail = 1 << 14 // Unbuffered (Unregistered)
	MemoryDeviceTypeDetailLRDIMM                 MemoryDeviceTypeDetail = 1 << 15 // LRDIMM
)

var memoryDeviceTypeDetailNames = map[MemoryDeviceTypeDetail]string{
	MemoryDeviceTypeDetailOther:                  "Other",
	MemoryDeviceTypeDetailUnknown:                "Unknown",
	MemoryDeviceTypeDetailFastpaged:              "Fast-paged",
	MemoryDeviceTypeDetailStaticColumn:           "Static column",
	MemoryDeviceTypeDetailPseudostatic:           "Pseudo-static",
	MemoryDeviceTypeDetailRAMBUS:                 "RAMBUS",
	MemoryDeviceTypeDetailSynchronous:            "Synchronous",
	MemoryDeviceTypeDetailCMOS:                   "CMOS",
	MemoryDeviceTypeDetailEDO:                    "EDO",
	MemoryDeviceTypeDetailWindowDRAM:             "Window DRAM",
	MemoryDeviceTypeDetailCacheDRAM:              "Cache DRAM",
	MemoryDeviceTypeDetailNonvolatile:            "Non-volatile",
	MemoryDeviceTypeDetailRegisteredBuffered:     "Registered (Buffered)",
	MemoryDeviceTypeDetailUnbufferedUnregistered: "Unbuffered (Unregistered)",
	MemoryDeviceTypeDetailLRDIMM:                 "LRDIMM",
}

func (v MemoryDeviceTypeDetail) String() string {
	if v&0xfffe == 0 {
		return "None"
	}
	var lines []string
	for i := 1; i < 16; i++ {
		if v&(1<<i) != 0 {
			lines = append(lines, memoryDeviceTypeDetailNames[1<<i])
		}
	}
	return strings.Join(lines, " ")
}

type MemoryDeviceTechnology uint8

const (
	MemoryDeviceTechnologyOther                 MemoryDeviceTechnology = 0x01 // Other
	MemoryDeviceTechnologyUnknown               MemoryDeviceTechnology = 0x02 // Unknown
	MemoryDeviceTechnologyDRAM                  MemoryDeviceTechnology = 0x03 // DRAM
	MemoryDeviceTechnologyNVDIMMN               MemoryDeviceTechnology = 0x04 // NVDIMM-N
	MemoryDeviceTechnologyNVDIMMF               MemoryDeviceTechnology = 0x05 // NVDIMM-F
	MemoryDeviceTechnologyNVDIMMP               MemoryDeviceTechnology = 0x06 // NVDIMM-P
	MemoryDeviceTechnologyIntelPersistentMemory MemoryDeviceTechnology = 0x07 // Intel persistent memory
)

var memoryDeviceTechnologyNames = map[MemoryDeviceTechnology]string{
	MemoryDeviceTechnologyOther:                 "Other",
	MemoryDeviceTechnologyUnknown:               "Unknown",
	MemoryDeviceTechnologyDRAM:                  "DRAM",
	MemoryDeviceTechnologyNVDIMMN:               "NVDIMM-N",
	MemoryDeviceTechnologyNVDIMMF:               "NVDIMM-F",
	MemoryDeviceTechnologyNVDIMMP:               "NVDIMM-P",
	MemoryDeviceTechnologyIntelPersistentMemory: "Intel persistent memory",
}

func (v MemoryDeviceTechnology) String() string {
	if name, ok := memoryDeviceTechnologyNames[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type MemoryDeviceOperatingModeCapability uint16

const (
	MemoryDeviceOperatingModeCapabilityOther                           MemoryDeviceOperatingModeCapability = 1 << 1 // Other
	MemoryDeviceOperatingModeCapabilityUnknown                         MemoryDeviceOperatingModeCapability = 1 << 2 // Unknown
	MemoryDeviceOperatingModeCapabilityVolatileMemory                  MemoryDeviceOperatingModeCapability = 1 << 3 // Volatile memory
	MemoryDeviceOperatingModeCapabilityByteaccessiblePersistentMemory  MemoryDeviceOperatingModeCapability = 1 << 4 // Byte-accessible persistent memory
	MemoryDeviceOperatingModeCapabilityBlockaccessiblePersistentMemory MemoryDeviceOperatingModeCapability = 1 << 5 // Block-accessible persistent memory
)

var memoryDeviceOperatingModeCapabilityNames = map[MemoryDeviceOperatingModeCapability]string{
	MemoryDeviceOperatingModeCapabilityOther:                           "Other",
	MemoryDeviceOperatingModeCapabilityUnknown:                         "Unknown",
	MemoryDeviceOperatingModeCapabilityVolatileMemory:                  "Volatile memory",
	MemoryDeviceOperatingModeCapabilityByteaccessiblePersistentMemory:  "Byte-accessible persistent memory",
	MemoryDeviceOperatingModeCapabilityBlockaccessiblePersistentMemory: "Block-accessible persistent memory",
}

func (v MemoryDeviceOperatingModeCapability) String() string {
	if v&0xfffe == 0 {
		return "None"
	}

	lines := []string{}

	for i := 1; i < 6; i++ {
		if v&(1<<i) != 0 {
			lines = append(lines, memoryDeviceOperatingModeCapabilityNames[1<<i])
		}
	}

	return strings.Join(lines, " ")
}
