package smbios

import (
	"fmt"
	"strings"
)

type Type0BIOS struct {
	Header              `smbios:"-"`
	Vendor              string
	Version             string
	AddressSegment      uint16
	ReleaseDate         string
	ROMSize             uint8
	Characteristics     BIOSChars
	CharacteristicsExt1 BIOSCharsExt1
	CharacteristicsExt2 BIOSCharsExt2
	BIOSMajorRelease    uint8 `smbios:"default=0xff"`
	BIOSMinorRelease    uint8 `smbios:"default=0xff"`
	ECMajorRelease      uint8 `smbios:"default=0xff"`
	ECMinorRelease      uint8 `smbios:"default=0xff"`
	ExtendedROMSize     uint16
}

type BIOSChars uint64

const (
	BIOSCharsReserved BIOSChars = 1 << iota
	BIOSCharsReserved2
	BIOSCharsUnknown
	BIOSCharsAreNotSupported
	BIOSCharsISA
	BIOSCharsMCA
	BIOSCharsEISA
	BIOSCharsPCI
	BIOSCharsPCMCIA
	BIOSCharsPlugAndPlay
	BIOSCharsAPM
	BIOSCharsBIOSUpgradeableFlash
	BIOSCharsBIOSShadowingIsAllowed
	BIOSCharsVLVESA
	BIOSCharsESCD
	BIOSCharsBootFromCD
	BIOSCharsSelectableBoot
	BIOSCharsBIOSROMSocketed
	BIOSCharsBootFromPCMCIA
	BIOSCharsEDD
	BIOSCharsJapaneseFloppyNEC
	BIOSCharsJapaneseFloppyToshiba
	BIOSChars360KBFloppy
	BIOSChars12MBFloppy
	BIOSChars720KBFloppy
	BIOSChars288MBFloppy
	BIOSCharsInt5h
	BIOSCharsInt9h
	BIOSCharsInt14h
	BIOSCharsInt17h
	BIOSCharsInt10h
	BIOSCharsNECPC98
)

var biosCharMap = map[BIOSChars]string{
	BIOSCharsReserved:               "Reserved",
	BIOSCharsReserved2:              "Reserved",
	BIOSCharsUnknown:                "Unknown",
	BIOSCharsAreNotSupported:        "BIOS characteristics not supported",
	BIOSCharsISA:                    "ISA is supported",
	BIOSCharsMCA:                    "MCA is supported",
	BIOSCharsEISA:                   "EISA is supported",
	BIOSCharsPCI:                    "PCI is supported",
	BIOSCharsPCMCIA:                 "PC Card (PCMCIA) is supported",
	BIOSCharsPlugAndPlay:            "PNP is supported",
	BIOSCharsAPM:                    "APM is supported",
	BIOSCharsBIOSUpgradeableFlash:   "BIOS is upgradeable",
	BIOSCharsBIOSShadowingIsAllowed: "BIOS shadowing is allowed",
	BIOSCharsVLVESA:                 "VLB is supported",
	BIOSCharsESCD:                   "ESCD support is available",
	BIOSCharsBootFromCD:             "Boot from CD is supported",
	BIOSCharsSelectableBoot:         "Selectable boot is supported",
	BIOSCharsBIOSROMSocketed:        "BIOS ROM is socketed",
	BIOSCharsBootFromPCMCIA:         "Boot from PC Card (PCMCIA) is supported",
	BIOSCharsEDD:                    "EDD is supported",
	BIOSCharsJapaneseFloppyNEC:      "Japanese floppy for NEC 9800 1.2 MB is supported (int 13h)",
	BIOSCharsJapaneseFloppyToshiba:  "Japanese floppy for Toshiba 1.2 MB is supported (int 13h)",
	BIOSChars360KBFloppy:            "5.25\"/360 kB floppy services are supported (int 13h)",
	BIOSChars12MBFloppy:             "5.25\"/1.2 MB floppy services are supported (int 13h)",
	BIOSChars720KBFloppy:            "3.5\"/720 kB floppy services are supported (int 13h)",
	BIOSChars288MBFloppy:            "3.5\"/2.88 MB floppy services are supported (int 13h)",
	BIOSCharsInt5h:                  "Print screen service is supported (int 5h)",
	BIOSCharsInt9h:                  "8042 keyboard services are supported (int 9h)",
	BIOSCharsInt14h:                 "Serial services are supported (int 14h)",
	BIOSCharsInt17h:                 "Printer services are supported (int 17h)",
	BIOSCharsInt10h:                 "CGA/mono video services are supported (int 10h)",
	BIOSCharsNECPC98:                "NEC PC-98",
}

func (b BIOSChars) String() string {
	var ss []string
	for i := range 32 {
		if b&(1<<i) != 0 {
			ss = append(ss, biosCharMap[1<<i])
		}
	}
	return strings.Join(ss, ", ")
}

type BIOSCharsExt1 uint8

const (
	BIOSCharsExt1ACPI               BIOSCharsExt1 = 1 << iota // ACPI is supported.
	BIOSCharsExt1USBLegacy                                    // USB Legacy is supported.
	BIOSCharsExt1AGP                                          // AGP is supported.
	BIOSCharsExt1I2OBoot                                      // I2O boot is supported.
	BIOSCharsExt1LS120SuperDiskBoot                           // LS-120 SuperDisk boot is supported.
	BIOSCharsExt1ATAPIZIPDriveBoot                            // ATAPI ZIP drive boot is supported.
	BIOSCharsExt11394Boot                                     // 1394 boot is supported.
	BIOSCharsExt1SmartBattery                                 // Smart battery is supported.
)

var biosCharExt1Map = map[BIOSCharsExt1]string{
	BIOSCharsExt1ACPI:               "ACPI is supported",
	BIOSCharsExt1USBLegacy:          "USB legacy is supported",
	BIOSCharsExt1AGP:                "AGP is supported",
	BIOSCharsExt1I2OBoot:            "I2O boot is supported",
	BIOSCharsExt1LS120SuperDiskBoot: "LS-120 boot is supported",
	BIOSCharsExt1ATAPIZIPDriveBoot:  "ATAPI Zip drive boot is supported",
	BIOSCharsExt11394Boot:           "IEEE 1394 boot is supported",
	BIOSCharsExt1SmartBattery:       "Smart battery is supported",
}

func (b BIOSCharsExt1) String() string {
	var ss []string
	for i := range 8 {
		if b&(1<<i) != 0 {
			ss = append(ss, biosCharExt1Map[1<<i])
		}
	}
	return strings.Join(ss, ", ")
}

type BIOSCharsExt2 uint8

const (
	BIOSCharsExt2BIOSBootSpecification       BIOSCharsExt2 = 1 << iota // BIOS Boot Specification is supported.
	BIOSCharsExt2FnNetworkServiceBoot                                  // Function key-initiated network service boot is supported.
	BIOSCharsExt2TargetedContentDistribution                           // Enable targeted content distribution.
	BIOSCharsExt2UEFISpecification                                     // UEFI Specification is supported.
	BIOSCharsExt2SMBIOSTableDescribesVM                                // SMBIOS table describes a virtual machine. (If this bit is not set, no inference can be made
)

var biosCharExt2Map = map[BIOSCharsExt2]string{
	BIOSCharsExt2BIOSBootSpecification:       "BIOS Boot Specification is supported",
	BIOSCharsExt2FnNetworkServiceBoot:        "Function key-initiated network service boot is supported",
	BIOSCharsExt2TargetedContentDistribution: "Enable targeted content distribution",
	BIOSCharsExt2UEFISpecification:           "UEFI Specification is supported",
	BIOSCharsExt2SMBIOSTableDescribesVM:      "SMBIOS table describes a virtual machine",
}

func (b BIOSCharsExt2) String() string {
	var ss []string
	for i := range 8 {
		if b&(1<<i) != 0 {
			ss = append(ss, biosCharExt2Map[1<<i])
		}
	}
	return strings.Join(ss, ", ")
}

func (b *Type0BIOS) GetROMSize() string {
	if b.ExtendedROMSize == 0 || b.ROMSize != 0xFF {
		return kgmt((uint64(b.ROMSize) + 1) * 65536)
	}
	extSize := uint64(b.ExtendedROMSize)
	unit := (extSize >> 14)
	multilplier := uint64(1)
	switch unit {
	case 0:
		multilplier = 1024 * 1024
	case 1:
		multilplier = 1024 * 1024 * 1024
	}
	return kgmt((extSize & 0x3FFF) * multilplier)
}

const (
	ErrInvalidTableType   = "invalid table type"
	ErrInvalidTableLength = "invalid table length"
)

func parseType0BIOS(t *Table) (*Type0BIOS, error) {
	if t.Header.Type != 0 {
		return nil, fmt.Errorf("%s:%v", ErrInvalidTableType, t.Header.Type)
	}
	if t.Header.Length < 0x12 {
		return nil, fmt.Errorf("%s:%v", ErrInvalidTableLength, t.Header.Length)
	}
	b := &Type0BIOS{
		Header: t.Header,
	}

	if _, err := parseType(t, 0, false, b); err != nil {
		return nil, fmt.Errorf("parse type 0 bios error: %w", err)
	}

	return b, nil
}
