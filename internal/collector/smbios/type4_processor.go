package smbios

import (
	"fmt"
)

type Type4Processor struct {
	Header            `smbios:"-"`
	SocketDesignation string
	ProcessorType     ProcessorType
	Family            uint8
	Manufacturer      string
	ID                uint64
	Version           string
	Voltage           uint8
	ExternalClock     uint16
	MaxSpeed          uint16
	CurrentSpeed      uint16
	Status            ProcessorStatus
	ProcessorUpgrade  ProcessorUpgrade
	L1CacheHanle      uint16
	L2CacheHanle      uint16
	L3CacheHanle      uint16
	SerialNumber      string
	AssetTag          string
	PartNumber        string
	CoreCount         uint8
	CoreEnabled       uint8
	ThreadCount       uint8
	Characteristics   ProcessorCharacteristics
	Family2           ProcessorFamily
	CoreCount2        uint16
	CoreEnabled2      uint16
	ThreadCount2      uint16
	TreadEnabled      uint16
	SocketType        uint8
}

func parseType4Processor(t *Table) (*Type4Processor, error) {
	if t.Type != 4 {
		return nil, fmt.Errorf("%s: %d", ErrInvalidTableType, t.Type)
	}

	if t.Header.Length < 0x1A {
		return nil, fmt.Errorf("%s: processor table must be at least %d bytes", ErrInvalidTableLength, 0x1A)
	}

	p := &Type4Processor{
		Header: t.Header,
	}
	_, err := parseType(t, 0, false, p)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Type 4 Processor: %w", err)
	}
	return p, nil
}

func (p *Type4Processor) GetFamily() ProcessorFamily {
	if p.Family2 == 0xfe && p.Header.Length >= 0x2a {
		return p.Family2
	}
	return ProcessorFamily(p.Family)
}

func (p *Type4Processor) GetVoltage() float32 {
	if p.Voltage&0x80 == 0 {
		switch {
		case p.Voltage&1 != 0:
			return 5.0
		case p.Voltage&2 != 0:
			return 3.3
		case p.Voltage&4 != 0:
			return 2.9
		}
		return 0.0
	}
	return float32(p.Voltage&0x7f) / 10.0
}

func (p *Type4Processor) GetCoreCount() int {
	if p.Header.Length >= 0x2c && p.CoreCount == 0xff {
		return int(p.CoreCount2)
	}
	return int(p.CoreCount)
}

func (p *Type4Processor) GetCoreEnabled() int {
	if p.Header.Length >= 0x2e && p.CoreEnabled == 0xff {
		return int(p.CoreEnabled2)
	}
	return int(p.CoreEnabled)
}

func (p *Type4Processor) GetThreadCount() int {
	if p.Header.Length >= 0x30 && p.ThreadCount == 0xff {
		return int(p.ThreadCount2)
	}
	return int(p.ThreadCount)
}

type ProcessorType uint8

const (
	ProcessorTypeOther            ProcessorType = 0x01 // Other
	ProcessorTypeUnknown          ProcessorType = 0x02 // Unknown
	ProcessorTypeCentralProcessor ProcessorType = 0x03 // Central Processor
	ProcessorTypeMathProcessor    ProcessorType = 0x04 // Math Processor
	ProcessorTypeDSP              ProcessorType = 0x05 // DSP Processor
	processorTypeVideoProcessor   ProcessorType = 0x06 // Video Processor
)

var processorTypeNames = map[ProcessorType]string{
	ProcessorTypeOther:            "Other",
	ProcessorTypeUnknown:          "Unknown",
	ProcessorTypeCentralProcessor: "Central Processor",
	ProcessorTypeMathProcessor:    "Math Processor",
	ProcessorTypeDSP:              "DSP Processor",
	processorTypeVideoProcessor:   "Video Processor",
}

func (v ProcessorType) String() string {
	if s, ok := processorTypeNames[v]; ok {
		return s
	}
	return fmt.Sprintf("%#x", uint8(v))
}

// ProcessorFamily is defined in DSP0134 7.5.2.
type ProcessorFamily uint16

// ProcessorFamily values are defined in DSP0134 7.5.2.
const (
	ProcessorFamilyOther                        ProcessorFamily = 0x01  // Other
	ProcessorFamilyUnknown                      ProcessorFamily = 0x02  // Unknown
	ProcessorFamily8086                         ProcessorFamily = 0x03  // 8086
	ProcessorFamily80286                        ProcessorFamily = 0x04  // 80286
	ProcessorFamily80386                        ProcessorFamily = 0x05  // 80386
	ProcessorFamily80486                        ProcessorFamily = 0x06  // 80486
	ProcessorFamily8087                         ProcessorFamily = 0x07  // 8087
	ProcessorFamily80287                        ProcessorFamily = 0x08  // 80287
	ProcessorFamily80387                        ProcessorFamily = 0x09  // 80387
	ProcessorFamily80487                        ProcessorFamily = 0x0a  // 80487
	ProcessorFamilyPentium                      ProcessorFamily = 0x0b  // Pentium
	ProcessorFamilyPentiumPro                   ProcessorFamily = 0x0c  // Pentium Pro
	ProcessorFamilyPentiumII                    ProcessorFamily = 0x0d  // Pentium II
	ProcessorFamilyPentiumMMX                   ProcessorFamily = 0x0e  // Pentium MMX
	ProcessorFamilyCeleron                      ProcessorFamily = 0x0f  // Celeron
	ProcessorFamilyPentiumIIXeon                ProcessorFamily = 0x10  // Pentium II Xeon
	ProcessorFamilyPentiumIII                   ProcessorFamily = 0x11  // Pentium III
	ProcessorFamilyM1                           ProcessorFamily = 0x12  // M1
	ProcessorFamilyM2                           ProcessorFamily = 0x13  // M2
	ProcessorFamilyCeleronM                     ProcessorFamily = 0x14  // Celeron M
	ProcessorFamilyPentium4HT                   ProcessorFamily = 0x15  // Pentium 4 HT
	ProcessorFamilyDuron                        ProcessorFamily = 0x18  // Duron
	ProcessorFamilyK5                           ProcessorFamily = 0x19  // K5
	ProcessorFamilyK6                           ProcessorFamily = 0x1a  // K6
	ProcessorFamilyK62                          ProcessorFamily = 0x1b  // K6-2
	ProcessorFamilyK63                          ProcessorFamily = 0x1c  // K6-3
	ProcessorFamilyAthlon                       ProcessorFamily = 0x1d  // Athlon
	ProcessorFamilyAMD29000                     ProcessorFamily = 0x1e  // AMD29000
	ProcessorFamilyK62Plus                      ProcessorFamily = 0x1f  // K6-2+
	ProcessorFamilyPowerPC                      ProcessorFamily = 0x20  // Power PC
	ProcessorFamilyPowerPC601                   ProcessorFamily = 0x21  // Power PC 601
	ProcessorFamilyPowerPC603                   ProcessorFamily = 0x22  // Power PC 603
	ProcessorFamilyPowerPC603Plus               ProcessorFamily = 0x23  // Power PC 603+
	ProcessorFamilyPowerPC604                   ProcessorFamily = 0x24  // Power PC 604
	ProcessorFamilyPowerPC620                   ProcessorFamily = 0x25  // Power PC 620
	ProcessorFamilyPowerPCX704                  ProcessorFamily = 0x26  // Power PC x704
	ProcessorFamilyPowerPC750                   ProcessorFamily = 0x27  // Power PC 750
	ProcessorFamilyCoreDuo                      ProcessorFamily = 0x28  // Core Duo
	ProcessorFamilyCoreDuoMobile                ProcessorFamily = 0x29  // Core Duo Mobile
	ProcessorFamilyCoreSoloMobile               ProcessorFamily = 0x2a  // Core Solo Mobile
	ProcessorFamilyAtom                         ProcessorFamily = 0x2b  // Atom
	ProcessorFamilyCoreM                        ProcessorFamily = 0x2c  // Core M
	ProcessorFamilyCoreM3                       ProcessorFamily = 0x2d  // Core m3
	ProcessorFamilyCoreM5                       ProcessorFamily = 0x2e  // Core m5
	ProcessorFamilyCoreM7                       ProcessorFamily = 0x2f  // Core m7
	ProcessorFamilyAlpha                        ProcessorFamily = 0x30  // Alpha
	ProcessorFamilyAlpha21064                   ProcessorFamily = 0x31  // Alpha 21064
	ProcessorFamilyAlpha21066                   ProcessorFamily = 0x32  // Alpha 21066
	ProcessorFamilyAlpha21164                   ProcessorFamily = 0x33  // Alpha 21164
	ProcessorFamilyAlpha21164PC                 ProcessorFamily = 0x34  // Alpha 21164PC
	ProcessorFamilyAlpha21164a                  ProcessorFamily = 0x35  // Alpha 21164a
	ProcessorFamilyAlpha21264                   ProcessorFamily = 0x36  // Alpha 21264
	ProcessorFamilyAlpha21364                   ProcessorFamily = 0x37  // Alpha 21364
	ProcessorFamilyTurionIIUltraDualCoreMobileM ProcessorFamily = 0x38  // Turion II Ultra Dual-Core Mobile M
	ProcessorFamilyTurionIIDualCoreMobileM      ProcessorFamily = 0x39  // Turion II Dual-Core Mobile M
	ProcessorFamilyAthlonIIDualCoreM            ProcessorFamily = 0x3a  // Athlon II Dual-Core M
	ProcessorFamilyOpteron6100                  ProcessorFamily = 0x3b  // Opteron 6100
	ProcessorFamilyOpteron4100                  ProcessorFamily = 0x3c  // Opteron 4100
	ProcessorFamilyOpteron6200                  ProcessorFamily = 0x3d  // Opteron 6200
	ProcessorFamilyOpteron4200                  ProcessorFamily = 0x3e  // Opteron 4200
	ProcessorFamilyFX                           ProcessorFamily = 0x3f  // FX
	ProcessorFamilyMIPS                         ProcessorFamily = 0x40  // MIPS
	ProcessorFamilyMIPSR4000                    ProcessorFamily = 0x41  // MIPS R4000
	ProcessorFamilyMIPSR4200                    ProcessorFamily = 0x42  // MIPS R4200
	ProcessorFamilyMIPSR4400                    ProcessorFamily = 0x43  // MIPS R4400
	ProcessorFamilyMIPSR4600                    ProcessorFamily = 0x44  // MIPS R4600
	ProcessorFamilyMIPSR10000                   ProcessorFamily = 0x45  // MIPS R10000
	ProcessorFamilyCSeries                      ProcessorFamily = 0x46  // C-Series
	ProcessorFamilyESeries                      ProcessorFamily = 0x47  // E-Series
	ProcessorFamilyASeries                      ProcessorFamily = 0x48  // A-Series
	ProcessorFamilyGSeries                      ProcessorFamily = 0x49  // G-Series
	ProcessorFamilyZSeries                      ProcessorFamily = 0x4a  // Z-Series
	ProcessorFamilyRSeries                      ProcessorFamily = 0x4b  // R-Series
	ProcessorFamilyOpteron4300                  ProcessorFamily = 0x4c  // Opteron 4300
	ProcessorFamilyOpteron6300                  ProcessorFamily = 0x4d  // Opteron 6300
	ProcessorFamilyOpteron3300                  ProcessorFamily = 0x4e  // Opteron 3300
	ProcessorFamilyFirePro                      ProcessorFamily = 0x4f  // FirePro
	ProcessorFamilySPARC                        ProcessorFamily = 0x50  // SPARC
	ProcessorFamilySuperSPARC                   ProcessorFamily = 0x51  // SuperSPARC
	ProcessorFamilyMicroSPARCII                 ProcessorFamily = 0x52  // MicroSPARC II
	ProcessorFamilyMicroSPARCIIep               ProcessorFamily = 0x53  // MicroSPARC IIep
	ProcessorFamilyUltraSPARC                   ProcessorFamily = 0x54  // UltraSPARC
	ProcessorFamilyUltraSPARCII                 ProcessorFamily = 0x55  // UltraSPARC II
	ProcessorFamilyUltraSPARCIIi                ProcessorFamily = 0x56  // UltraSPARC IIi
	ProcessorFamilyUltraSPARCIII                ProcessorFamily = 0x57  // UltraSPARC III
	ProcessorFamilyUltraSPARCIIIi               ProcessorFamily = 0x58  // UltraSPARC IIIi
	ProcessorFamily68040                        ProcessorFamily = 0x60  // 68040
	ProcessorFamily68xxx                        ProcessorFamily = 0x61  // 68xxx
	ProcessorFamily68000                        ProcessorFamily = 0x62  // 68000
	ProcessorFamily68010                        ProcessorFamily = 0x63  // 68010
	ProcessorFamily68020                        ProcessorFamily = 0x64  // 68020
	ProcessorFamily68030                        ProcessorFamily = 0x65  // 68030
	ProcessorFamilyAthlonX4                     ProcessorFamily = 0x66  // Athlon X4
	ProcessorFamilyOpteronX1000                 ProcessorFamily = 0x67  // Opteron X1000
	ProcessorFamilyOpteronX2000                 ProcessorFamily = 0x68  // Opteron X2000
	ProcessorFamilyOpteronASeries               ProcessorFamily = 0x69  // Opteron A-Series
	ProcessorFamilyOpteronX3000                 ProcessorFamily = 0x6a  // Opteron X3000
	ProcessorFamilyZen                          ProcessorFamily = 0x6b  // Zen
	ProcessorFamilyHobbit                       ProcessorFamily = 0x70  // Hobbit
	ProcessorFamilyCrusoeTM5000                 ProcessorFamily = 0x78  // Crusoe TM5000
	ProcessorFamilyCrusoeTM3000                 ProcessorFamily = 0x79  // Crusoe TM3000
	ProcessorFamilyEfficeonTM8000               ProcessorFamily = 0x7a  // Efficeon TM8000
	ProcessorFamilyWeitek                       ProcessorFamily = 0x80  // Weitek
	ProcessorFamilyItanium                      ProcessorFamily = 0x82  // Itanium
	ProcessorFamilyAthlon64                     ProcessorFamily = 0x83  // Athlon 64
	ProcessorFamilyOpteron                      ProcessorFamily = 0x84  // Opteron
	ProcessorFamilySempron                      ProcessorFamily = 0x85  // Sempron
	ProcessorFamilyTurion64                     ProcessorFamily = 0x86  // Turion 64
	ProcessorFamilyDualCoreOpteron              ProcessorFamily = 0x87  // Dual-Core Opteron
	ProcessorFamilyAthlon64X2                   ProcessorFamily = 0x88  // Athlon 64 X2
	ProcessorFamilyTurion64X2                   ProcessorFamily = 0x89  // Turion 64 X2
	ProcessorFamilyQuadCoreOpteron              ProcessorFamily = 0x8a  // Quad-Core Opteron
	ProcessorFamilyThirdGenerationOpteron       ProcessorFamily = 0x8b  // Third-Generation Opteron
	ProcessorFamilyPhenomFX                     ProcessorFamily = 0x8c  // Phenom FX
	ProcessorFamilyPhenomX4                     ProcessorFamily = 0x8d  // Phenom X4
	ProcessorFamilyPhenomX2                     ProcessorFamily = 0x8e  // Phenom X2
	ProcessorFamilyAthlonX2                     ProcessorFamily = 0x8f  // Athlon X2
	ProcessorFamilyPARISC                       ProcessorFamily = 0x90  // PA-RISC
	ProcessorFamilyPARISC8500                   ProcessorFamily = 0x91  // PA-RISC 8500
	ProcessorFamilyPARISC8000                   ProcessorFamily = 0x92  // PA-RISC 8000
	ProcessorFamilyPARISC7300LC                 ProcessorFamily = 0x93  // PA-RISC 7300LC
	ProcessorFamilyPARISC7200                   ProcessorFamily = 0x94  // PA-RISC 7200
	ProcessorFamilyPARISC7100LC                 ProcessorFamily = 0x95  // PA-RISC 7100LC
	ProcessorFamilyPARISC7100                   ProcessorFamily = 0x96  // PA-RISC 7100
	ProcessorFamilyV30                          ProcessorFamily = 0xa0  // V30
	ProcessorFamilyQuadCoreXeon3200             ProcessorFamily = 0xa1  // Quad-Core Xeon 3200
	ProcessorFamilyDualCoreXeon3000             ProcessorFamily = 0xa2  // Dual-Core Xeon 3000
	ProcessorFamilyQuadCoreXeon5300             ProcessorFamily = 0xa3  // Quad-Core Xeon 5300
	ProcessorFamilyDualCoreXeon5100             ProcessorFamily = 0xa4  // Dual-Core Xeon 5100
	ProcessorFamilyDualCoreXeon5000             ProcessorFamily = 0xa5  // Dual-Core Xeon 5000
	ProcessorFamilyDualCoreXeonLV               ProcessorFamily = 0xa6  // Dual-Core Xeon LV
	ProcessorFamilyDualCoreXeonULV              ProcessorFamily = 0xa7  // Dual-Core Xeon ULV
	ProcessorFamilyDualCoreXeon7100             ProcessorFamily = 0xa8  // Dual-Core Xeon 7100
	ProcessorFamilyQuadCoreXeon5400             ProcessorFamily = 0xa9  // Quad-Core Xeon 5400
	ProcessorFamilyQuadCoreXeon                 ProcessorFamily = 0xaa  // Quad-Core Xeon
	ProcessorFamilyDualCoreXeon5200             ProcessorFamily = 0xab  // Dual-Core Xeon 5200
	ProcessorFamilyDualCoreXeon7200             ProcessorFamily = 0xac  // Dual-Core Xeon 7200
	ProcessorFamilyQuadCoreXeon7300             ProcessorFamily = 0xad  // Quad-Core Xeon 7300
	ProcessorFamilyQuadCoreXeon7400             ProcessorFamily = 0xae  // Quad-Core Xeon 7400
	ProcessorFamilyMultiCoreXeon7400            ProcessorFamily = 0xaf  // Multi-Core Xeon 7400
	ProcessorFamilyPentiumIIIXeon               ProcessorFamily = 0xb0  // Pentium III Xeon
	ProcessorFamilyPentiumIIISpeedstep          ProcessorFamily = 0xb1  // Pentium III Speedstep
	ProcessorFamilyPentium4                     ProcessorFamily = 0xb2  // Pentium 4
	ProcessorFamilyXeon                         ProcessorFamily = 0xb3  // Xeon
	ProcessorFamilyAS400                        ProcessorFamily = 0xb4  // AS400
	ProcessorFamilyXeonMP                       ProcessorFamily = 0xb5  // Xeon MP
	ProcessorFamilyAthlonXP                     ProcessorFamily = 0xb6  // Athlon XP
	ProcessorFamilyAthlonMP                     ProcessorFamily = 0xb7  // Athlon MP
	ProcessorFamilyItanium2                     ProcessorFamily = 0xb8  // Itanium 2
	ProcessorFamilyPentiumM                     ProcessorFamily = 0xb9  // Pentium M
	ProcessorFamilyCeleronD                     ProcessorFamily = 0xba  // Celeron D
	ProcessorFamilyPentiumD                     ProcessorFamily = 0xbb  // Pentium D
	ProcessorFamilyPentiumEE                    ProcessorFamily = 0xbc  // Pentium EE
	ProcessorFamilyCoreSolo                     ProcessorFamily = 0xbd  // Core Solo
	ProcessorFamilyHandledAsASpecialCase        ProcessorFamily = 0xbe  // handled as a special case */
	ProcessorFamilyCore2Duo                     ProcessorFamily = 0xbf  // Core 2 Duo
	ProcessorFamilyCore2Solo                    ProcessorFamily = 0xc0  // Core 2 Solo
	ProcessorFamilyCore2Extreme                 ProcessorFamily = 0xc1  // Core 2 Extreme
	ProcessorFamilyCore2Quad                    ProcessorFamily = 0xc2  // Core 2 Quad
	ProcessorFamilyCore2ExtremeMobile           ProcessorFamily = 0xc3  // Core 2 Extreme Mobile
	ProcessorFamilyCore2DuoMobile               ProcessorFamily = 0xc4  // Core 2 Duo Mobile
	ProcessorFamilyCore2SoloMobile              ProcessorFamily = 0xc5  // Core 2 Solo Mobile
	ProcessorFamilyCoreI7                       ProcessorFamily = 0xc6  // Core i7
	ProcessorFamilyDualCoreCeleron              ProcessorFamily = 0xc7  // Dual-Core Celeron
	ProcessorFamilyIBM390                       ProcessorFamily = 0xc8  // IBM390
	ProcessorFamilyG4                           ProcessorFamily = 0xc9  // G4
	ProcessorFamilyG5                           ProcessorFamily = 0xca  // G5
	ProcessorFamilyESA390G6                     ProcessorFamily = 0xcb  // ESA/390 G6
	ProcessorFamilyZarchitecture                ProcessorFamily = 0xcc  // z/Architecture
	ProcessorFamilyCoreI5                       ProcessorFamily = 0xcd  // Core i5
	ProcessorFamilyCoreI3                       ProcessorFamily = 0xce  // Core i3
	ProcessorFamilyCoreI9                       ProcessorFamily = 0xcf  // Core i9
	ProcessorFamilyC7M                          ProcessorFamily = 0xd2  // C7-M
	ProcessorFamilyC7D                          ProcessorFamily = 0xd3  // C7-D
	ProcessorFamilyC7                           ProcessorFamily = 0xd4  // C7
	ProcessorFamilyEden                         ProcessorFamily = 0xd5  // Eden
	ProcessorFamilyMultiCoreXeon                ProcessorFamily = 0xd6  // Multi-Core Xeon
	ProcessorFamilyDualCoreXeon3xxx             ProcessorFamily = 0xd7  // Dual-Core Xeon 3xxx
	ProcessorFamilyQuadCoreXeon3xxx             ProcessorFamily = 0xd8  // Quad-Core Xeon 3xxx
	ProcessorFamilyNano                         ProcessorFamily = 0xd9  // Nano
	ProcessorFamilyDualCoreXeon5xxx             ProcessorFamily = 0xda  // Dual-Core Xeon 5xxx
	ProcessorFamilyQuadCoreXeon5xxx             ProcessorFamily = 0xdb  // Quad-Core Xeon 5xxx
	ProcessorFamilyDualCoreXeon7xxx             ProcessorFamily = 0xdd  // Dual-Core Xeon 7xxx
	ProcessorFamilyQuadCoreXeon7xxx             ProcessorFamily = 0xde  // Quad-Core Xeon 7xxx
	ProcessorFamilyMultiCoreXeon7xxx            ProcessorFamily = 0xdf  // Multi-Core Xeon 7xxx
	ProcessorFamilyMultiCoreXeon3400            ProcessorFamily = 0xe0  // Multi-Core Xeon 3400
	ProcessorFamilyOpteron3000                  ProcessorFamily = 0xe4  // Opteron 3000
	ProcessorFamilySempronII                    ProcessorFamily = 0xe5  // Sempron II
	ProcessorFamilyEmbeddedOpteronQuadCore      ProcessorFamily = 0xe6  // Embedded Opteron Quad-Core
	ProcessorFamilyPhenomTripleCore             ProcessorFamily = 0xe7  // Phenom Triple-Core
	ProcessorFamilyTurionUltraDualCoreMobile    ProcessorFamily = 0xe8  // Turion Ultra Dual-Core Mobile
	ProcessorFamilyTurionDualCoreMobile         ProcessorFamily = 0xe9  // Turion Dual-Core Mobile
	ProcessorFamilyAthlonDualCore               ProcessorFamily = 0xea  // Athlon Dual-Core
	ProcessorFamilySempronSI                    ProcessorFamily = 0xeb  // Sempron SI
	ProcessorFamilyPhenomII                     ProcessorFamily = 0xec  // Phenom II
	ProcessorFamilyAthlonII                     ProcessorFamily = 0xed  // Athlon II
	ProcessorFamilySixCoreOpteron               ProcessorFamily = 0xee  // Six-Core Opteron
	ProcessorFamilySempronM                     ProcessorFamily = 0xef  // Sempron M
	ProcessorFamilyI860                         ProcessorFamily = 0xfa  // i860
	ProcessorFamilyI960                         ProcessorFamily = 0xfb  // i960
	ProcessorFamilyARMv7                        ProcessorFamily = 0x100 // ARMv7
	ProcessorFamilyARMv8                        ProcessorFamily = 0x101 // ARMv8
	ProcessorFamilySH3                          ProcessorFamily = 0x104 // SH-3
	ProcessorFamilySH4                          ProcessorFamily = 0x105 // SH-4
	ProcessorFamilyARM                          ProcessorFamily = 0x118 // ARM
	ProcessorFamilyStrongARM                    ProcessorFamily = 0x119 // StrongARM
	ProcessorFamily6x86                         ProcessorFamily = 0x12c // 6x86
	ProcessorFamilyMediaGX                      ProcessorFamily = 0x12d // MediaGX
	ProcessorFamilyMII                          ProcessorFamily = 0x12e // MII
	ProcessorFamilyWinChip                      ProcessorFamily = 0x140 // WinChip
	ProcessorFamilyDSP                          ProcessorFamily = 0x15e // DSP
	ProcessorFamilyVideoProcessor               ProcessorFamily = 0x1f4 // Video Processor
	ProcessorFamilyRV32                         ProcessorFamily = 0x200 // RISC-V RV32
	ProcessorFamilyRV64                         ProcessorFamily = 0x201 // RISC-V RV64
	ProcessorFamilyRV128                        ProcessorFamily = 0x202 // RISC-V RV128
	ProcessorFamilyLoongArch                    ProcessorFamily = 0x258 // LoongArch
	processorFamilyLoongson1                    ProcessorFamily = 0x259 // Loongson™ 1 Processor Family
	ProcessorFamilyLoongson2                    ProcessorFamily = 0x25a // Loongson™ 2 Processor Family
	ProcessorFamilyLoongson3                    ProcessorFamily = 0x25b // Loongson™ 3 Processor Family
	ProcessorFamilyLoongson2K                   ProcessorFamily = 0x25c // Loongson™ 2K Processor Family
	ProcessorFamilyLoongson3A                   ProcessorFamily = 0x25d // Loongson™ 3A Processor Family
	ProcessorFamilyLoongson3B                   ProcessorFamily = 0x25e // Loongson™ 3B Processor Family
	ProcessorFamilyLoongson3C                   ProcessorFamily = 0x25f // Loongson™ 3C Processor Family
	ProcessorFamilyLoongson3D                   ProcessorFamily = 0x260 // Loongson™ 3D Processor Family
	ProcessorFamilyLoongson3E                   ProcessorFamily = 0x261 // Loongson™ 3E Processor Family
	ProcessorFamilyLoongsonDual2K               ProcessorFamily = 0x262 // Dual-Core Loongson™ 2K Processor Series
	ProcessorFamilyLoongsonQuad3A               ProcessorFamily = 0x26c // Quad-Core Loongson™ 3A Processor Series
	processorFamilyLoongsonMulti3A              ProcessorFamily = 0x26d // Multi-Core Loongson™ 3A Processor Series
	ProcessorFamilyLoongsonQuad3B               ProcessorFamily = 0x26e // Quad-Core Loongson™ 3B Processor Series
	ProcessorFamilyLoongsonMulti3B              ProcessorFamily = 0x26f // Multi-Core Loongson™ 3B Processor Series
	ProcessorFamilyLoongsonMulti3C              ProcessorFamily = 0x270 // Multi-Core Loongson™ 3C Processor Series
	ProcessorFamilyLoongsonMulti3D              ProcessorFamily = 0x271 // Multi-Core Loongson™ 3D Processor Series
	ProcessorFamilyIntel3                       ProcessorFamily = 0x300 // Intel® Core™ 3
	ProcessorFamilyIntel5                       ProcessorFamily = 0x301 // Intel® Core™ 5
	ProcessorFamilyIntel7                       ProcessorFamily = 0x302 // Intel® Core™ 7
	ProcessorFamilyIntel9                       ProcessorFamily = 0x303 // Intel® Core™ 9
	ProcessorFamilyIntelUltra3                  ProcessorFamily = 0x304 // Intel® Core™ Ultra 3
	ProcessorFamilyIntelUltra5                  ProcessorFamily = 0x305 // Intel® Core™ Ultra 5
	ProcessorFamilyIntelUltra7                  ProcessorFamily = 0x306 // Intel® Core™ Ultra 7
	ProcessorFamilyIntelUltra9                  ProcessorFamily = 0x307 // Intel® Core™ Ultra 9
)

var processorFamilyStr = map[ProcessorFamily]string{
	ProcessorFamilyOther:                        "Other",
	ProcessorFamilyUnknown:                      "Unknown",
	ProcessorFamily8086:                         "8086",
	ProcessorFamily80286:                        "80286",
	ProcessorFamily80386:                        "80386",
	ProcessorFamily80486:                        "80486",
	ProcessorFamily8087:                         "8087",
	ProcessorFamily80287:                        "80287",
	ProcessorFamily80387:                        "80387",
	ProcessorFamily80487:                        "80487",
	ProcessorFamilyPentium:                      "Pentium",
	ProcessorFamilyPentiumPro:                   "Pentium Pro",
	ProcessorFamilyPentiumII:                    "Pentium II",
	ProcessorFamilyPentiumMMX:                   "Pentium MMX",
	ProcessorFamilyCeleron:                      "Celeron",
	ProcessorFamilyPentiumIIXeon:                "Pentium II Xeon",
	ProcessorFamilyPentiumIII:                   "Pentium III",
	ProcessorFamilyM1:                           "M1",
	ProcessorFamilyM2:                           "M2",
	ProcessorFamilyCeleronM:                     "Celeron M",
	ProcessorFamilyPentium4HT:                   "Pentium 4 HT",
	ProcessorFamilyDuron:                        "Duron",
	ProcessorFamilyK5:                           "K5",
	ProcessorFamilyK6:                           "K6",
	ProcessorFamilyK62:                          "K6-2",
	ProcessorFamilyK63:                          "K6-3",
	ProcessorFamilyAthlon:                       "Athlon",
	ProcessorFamilyAMD29000:                     "AMD29000",
	ProcessorFamilyK62Plus:                      "K6-2+",
	ProcessorFamilyPowerPC:                      "Power PC",
	ProcessorFamilyPowerPC601:                   "Power PC 601",
	ProcessorFamilyPowerPC603:                   "Power PC 603",
	ProcessorFamilyPowerPC603Plus:               "Power PC 603+",
	ProcessorFamilyPowerPC604:                   "Power PC 604",
	ProcessorFamilyPowerPC620:                   "Power PC 620",
	ProcessorFamilyPowerPCX704:                  "Power PC x704",
	ProcessorFamilyPowerPC750:                   "Power PC 750",
	ProcessorFamilyCoreDuo:                      "Core Duo",
	ProcessorFamilyCoreDuoMobile:                "Core Duo Mobile",
	ProcessorFamilyCoreSoloMobile:               "Core Solo Mobile",
	ProcessorFamilyAtom:                         "Atom",
	ProcessorFamilyCoreM:                        "Core M",
	ProcessorFamilyCoreM3:                       "Core m3",
	ProcessorFamilyCoreM5:                       "Core m5",
	ProcessorFamilyCoreM7:                       "Core m7",
	ProcessorFamilyAlpha:                        "Alpha",
	ProcessorFamilyAlpha21064:                   "Alpha 21064",
	ProcessorFamilyAlpha21066:                   "Alpha 21066",
	ProcessorFamilyAlpha21164:                   "Alpha 21164",
	ProcessorFamilyAlpha21164PC:                 "Alpha 21164PC",
	ProcessorFamilyAlpha21164a:                  "Alpha 21164a",
	ProcessorFamilyAlpha21264:                   "Alpha 21264",
	ProcessorFamilyAlpha21364:                   "Alpha 21364",
	ProcessorFamilyTurionIIUltraDualCoreMobileM: "Turion II Ultra Dual-Core Mobile M",
	ProcessorFamilyTurionIIDualCoreMobileM:      "Turion II Dual-Core Mobile M",
	ProcessorFamilyAthlonIIDualCoreM:            "Athlon II Dual-Core M",
	ProcessorFamilyOpteron6100:                  "Opteron 6100",
	ProcessorFamilyOpteron4100:                  "Opteron 4100",
	ProcessorFamilyOpteron6200:                  "Opteron 6200",
	ProcessorFamilyOpteron4200:                  "Opteron 4200",
	ProcessorFamilyFX:                           "FX",
	ProcessorFamilyMIPS:                         "MIPS",
	ProcessorFamilyMIPSR4000:                    "MIPS R4000",
	ProcessorFamilyMIPSR4200:                    "MIPS R4200",
	ProcessorFamilyMIPSR4400:                    "MIPS R4400",
	ProcessorFamilyMIPSR4600:                    "MIPS R4600",
	ProcessorFamilyMIPSR10000:                   "MIPS R10000",
	ProcessorFamilyCSeries:                      "C-Series",
	ProcessorFamilyESeries:                      "E-Series",
	ProcessorFamilyASeries:                      "A-Series",
	ProcessorFamilyGSeries:                      "G-Series",
	ProcessorFamilyZSeries:                      "Z-Series",
	ProcessorFamilyRSeries:                      "R-Series",
	ProcessorFamilyOpteron4300:                  "Opteron 4300",
	ProcessorFamilyOpteron6300:                  "Opteron 6300",
	ProcessorFamilyOpteron3300:                  "Opteron 3300",
	ProcessorFamilyFirePro:                      "FirePro",
	ProcessorFamilySPARC:                        "SPARC",
	ProcessorFamilySuperSPARC:                   "SuperSPARC",
	ProcessorFamilyMicroSPARCII:                 "MicroSPARC II",
	ProcessorFamilyMicroSPARCIIep:               "MicroSPARC IIep",
	ProcessorFamilyUltraSPARC:                   "UltraSPARC",
	ProcessorFamilyUltraSPARCII:                 "UltraSPARC II",
	ProcessorFamilyUltraSPARCIIi:                "UltraSPARC IIi",
	ProcessorFamilyUltraSPARCIII:                "UltraSPARC III",
	ProcessorFamilyUltraSPARCIIIi:               "UltraSPARC IIIi",
	ProcessorFamily68040:                        "68040",
	ProcessorFamily68xxx:                        "68xxx",
	ProcessorFamily68000:                        "68000",
	ProcessorFamily68010:                        "68010",
	ProcessorFamily68020:                        "68020",
	ProcessorFamily68030:                        "68030",
	ProcessorFamilyAthlonX4:                     "Athlon X4",
	ProcessorFamilyOpteronX1000:                 "Opteron X1000",
	ProcessorFamilyOpteronX2000:                 "Opteron X2000",
	ProcessorFamilyOpteronASeries:               "Opteron A-Series",
	ProcessorFamilyOpteronX3000:                 "Opteron X3000",
	ProcessorFamilyZen:                          "Zen",
	ProcessorFamilyHobbit:                       "Hobbit",
	ProcessorFamilyCrusoeTM5000:                 "Crusoe TM5000",
	ProcessorFamilyCrusoeTM3000:                 "Crusoe TM3000",
	ProcessorFamilyEfficeonTM8000:               "Efficeon TM8000",
	ProcessorFamilyWeitek:                       "Weitek",
	ProcessorFamilyItanium:                      "Itanium",
	ProcessorFamilyAthlon64:                     "Athlon 64",
	ProcessorFamilyOpteron:                      "Opteron",
	ProcessorFamilySempron:                      "Sempron",
	ProcessorFamilyTurion64:                     "Turion 64",
	ProcessorFamilyDualCoreOpteron:              "Dual-Core Opteron",
	ProcessorFamilyAthlon64X2:                   "Athlon 64 X2",
	ProcessorFamilyTurion64X2:                   "Turion 64 X2",
	ProcessorFamilyQuadCoreOpteron:              "Quad-Core Opteron",
	ProcessorFamilyThirdGenerationOpteron:       "Third-Generation Opteron",
	ProcessorFamilyPhenomFX:                     "Phenom FX",
	ProcessorFamilyPhenomX4:                     "Phenom X4",
	ProcessorFamilyPhenomX2:                     "Phenom X2",
	ProcessorFamilyAthlonX2:                     "Athlon X2",
	ProcessorFamilyPARISC:                       "PA-RISC",
	ProcessorFamilyPARISC8500:                   "PA-RISC 8500",
	ProcessorFamilyPARISC8000:                   "PA-RISC 8000",
	ProcessorFamilyPARISC7300LC:                 "PA-RISC 7300LC",
	ProcessorFamilyPARISC7200:                   "PA-RISC 7200",
	ProcessorFamilyPARISC7100LC:                 "PA-RISC 7100LC",
	ProcessorFamilyPARISC7100:                   "PA-RISC 7100",
	ProcessorFamilyV30:                          "V30",
	ProcessorFamilyQuadCoreXeon3200:             "Quad-Core Xeon 3200",
	ProcessorFamilyDualCoreXeon3000:             "Dual-Core Xeon 3000",
	ProcessorFamilyQuadCoreXeon5300:             "Quad-Core Xeon 5300",
	ProcessorFamilyDualCoreXeon5100:             "Dual-Core Xeon 5100",
	ProcessorFamilyDualCoreXeon5000:             "Dual-Core Xeon 5000",
	ProcessorFamilyDualCoreXeonLV:               "Dual-Core Xeon LV",
	ProcessorFamilyDualCoreXeonULV:              "Dual-Core Xeon ULV",
	ProcessorFamilyDualCoreXeon7100:             "Dual-Core Xeon 7100",
	ProcessorFamilyQuadCoreXeon5400:             "Quad-Core Xeon 5400",
	ProcessorFamilyQuadCoreXeon:                 "Quad-Core Xeon",
	ProcessorFamilyDualCoreXeon5200:             "Dual-Core Xeon 5200",
	ProcessorFamilyDualCoreXeon7200:             "Dual-Core Xeon 7200",
	ProcessorFamilyQuadCoreXeon7300:             "Quad-Core Xeon 7300",
	ProcessorFamilyQuadCoreXeon7400:             "Quad-Core Xeon 7400",
	ProcessorFamilyMultiCoreXeon7400:            "Multi-Core Xeon 7400",
	ProcessorFamilyPentiumIIIXeon:               "Pentium III Xeon",
	ProcessorFamilyPentiumIIISpeedstep:          "Pentium III Speedstep",
	ProcessorFamilyPentium4:                     "Pentium 4",
	ProcessorFamilyXeon:                         "Xeon",
	ProcessorFamilyAS400:                        "AS400",
	ProcessorFamilyXeonMP:                       "Xeon MP",
	ProcessorFamilyAthlonXP:                     "Athlon XP",
	ProcessorFamilyAthlonMP:                     "Athlon MP",
	ProcessorFamilyItanium2:                     "Itanium 2",
	ProcessorFamilyPentiumM:                     "Pentium M",
	ProcessorFamilyCeleronD:                     "Celeron D",
	ProcessorFamilyPentiumD:                     "Pentium D",
	ProcessorFamilyPentiumEE:                    "Pentium EE",
	ProcessorFamilyCoreSolo:                     "Core Solo",
	ProcessorFamilyHandledAsASpecialCase:        "handled as a special case */",
	ProcessorFamilyCore2Duo:                     "Core 2 Duo",
	ProcessorFamilyCore2Solo:                    "Core 2 Solo",
	ProcessorFamilyCore2Extreme:                 "Core 2 Extreme",
	ProcessorFamilyCore2Quad:                    "Core 2 Quad",
	ProcessorFamilyCore2ExtremeMobile:           "Core 2 Extreme Mobile",
	ProcessorFamilyCore2DuoMobile:               "Core 2 Duo Mobile",
	ProcessorFamilyCore2SoloMobile:              "Core 2 Solo Mobile",
	ProcessorFamilyCoreI7:                       "Core i7",
	ProcessorFamilyDualCoreCeleron:              "Dual-Core Celeron",
	ProcessorFamilyIBM390:                       "IBM390",
	ProcessorFamilyG4:                           "G4",
	ProcessorFamilyG5:                           "G5",
	ProcessorFamilyESA390G6:                     "ESA/390 G6",
	ProcessorFamilyZarchitecture:                "z/Architecture",
	ProcessorFamilyCoreI5:                       "Core i5",
	ProcessorFamilyCoreI3:                       "Core i3",
	ProcessorFamilyCoreI9:                       "Core i9",
	ProcessorFamilyC7M:                          "C7-M",
	ProcessorFamilyC7D:                          "C7-D",
	ProcessorFamilyC7:                           "C7",
	ProcessorFamilyEden:                         "Eden",
	ProcessorFamilyMultiCoreXeon:                "Multi-Core Xeon",
	ProcessorFamilyDualCoreXeon3xxx:             "Dual-Core Xeon 3xxx",
	ProcessorFamilyQuadCoreXeon3xxx:             "Quad-Core Xeon 3xxx",
	ProcessorFamilyNano:                         "Nano",
	ProcessorFamilyDualCoreXeon5xxx:             "Dual-Core Xeon 5xxx",
	ProcessorFamilyQuadCoreXeon5xxx:             "Quad-Core Xeon 5xxx",
	ProcessorFamilyDualCoreXeon7xxx:             "Dual-Core Xeon 7xxx",
	ProcessorFamilyQuadCoreXeon7xxx:             "Quad-Core Xeon 7xxx",
	ProcessorFamilyMultiCoreXeon7xxx:            "Multi-Core Xeon 7xxx",
	ProcessorFamilyMultiCoreXeon3400:            "Multi-Core Xeon 3400",
	ProcessorFamilyOpteron3000:                  "Opteron 3000",
	ProcessorFamilySempronII:                    "Sempron II",
	ProcessorFamilyEmbeddedOpteronQuadCore:      "Embedded Opteron Quad-Core",
	ProcessorFamilyPhenomTripleCore:             "Phenom Triple-Core",
	ProcessorFamilyTurionUltraDualCoreMobile:    "Turion Ultra Dual-Core Mobile",
	ProcessorFamilyTurionDualCoreMobile:         "Turion Dual-Core Mobile",
	ProcessorFamilyAthlonDualCore:               "Athlon Dual-Core",
	ProcessorFamilySempronSI:                    "Sempron SI",
	ProcessorFamilyPhenomII:                     "Phenom II",
	ProcessorFamilyAthlonII:                     "Athlon II",
	ProcessorFamilySixCoreOpteron:               "Six-Core Opteron",
	ProcessorFamilySempronM:                     "Sempron M",
	ProcessorFamilyI860:                         "i860",
	ProcessorFamilyI960:                         "i960",
	ProcessorFamilyARMv7:                        "ARMv7",
	ProcessorFamilyARMv8:                        "ARMv8",
	ProcessorFamilySH3:                          "SH-3",
	ProcessorFamilySH4:                          "SH-4",
	ProcessorFamilyARM:                          "ARM",
	ProcessorFamilyStrongARM:                    "StrongARM",
	ProcessorFamily6x86:                         "6x86",
	ProcessorFamilyMediaGX:                      "MediaGX",
	ProcessorFamilyMII:                          "MII",
	ProcessorFamilyWinChip:                      "WinChip",
	ProcessorFamilyDSP:                          "DSP",
	ProcessorFamilyVideoProcessor:               "Video Processor",
	ProcessorFamilyRV32:                         " RISC-V RV32",
	ProcessorFamilyRV64:                         " RISC-V RV64",
	ProcessorFamilyRV128:                        " RISC-V RV128",
	ProcessorFamilyLoongArch:                    " LoongArch",
	processorFamilyLoongson1:                    " Loongson™ 1 Processor Family",
	ProcessorFamilyLoongson2:                    " Loongson™ 2 Processor Family",
	ProcessorFamilyLoongson3:                    " Loongson™ 3 Processor Family",
	ProcessorFamilyLoongson2K:                   " Loongson™ 2K Processor Family",
	ProcessorFamilyLoongson3A:                   " Loongson™ 3A Processor Family",
	ProcessorFamilyLoongson3B:                   " Loongson™ 3B Processor Family",
	ProcessorFamilyLoongson3C:                   " Loongson™ 3C Processor Family",
	ProcessorFamilyLoongson3D:                   " Loongson™ 3D Processor Family",
	ProcessorFamilyLoongson3E:                   " Loongson™ 3E Processor Family",
	ProcessorFamilyLoongsonDual2K:               " Dual-Core Loongson™ 2K Processor Series",
	ProcessorFamilyLoongsonQuad3A:               " Quad-Core Loongson™ 3A Processor Series",
	processorFamilyLoongsonMulti3A:              " Multi-Core Loongson™ 3A Processor Series",
	ProcessorFamilyLoongsonQuad3B:               " Quad-Core Loongson™ 3B Processor Series",
	ProcessorFamilyLoongsonMulti3B:              " Multi-Core Loongson™ 3B Processor Series",
	ProcessorFamilyLoongsonMulti3C:              " Multi-Core Loongson™ 3C Processor Series",
	ProcessorFamilyLoongsonMulti3D:              " Multi-Core Loongson™ 3D Processor Series",
	ProcessorFamilyIntel3:                       " Intel® Core™ 3",
	ProcessorFamilyIntel5:                       " Intel® Core™ 5",
	ProcessorFamilyIntel7:                       " Intel® Core™ 7",
	ProcessorFamilyIntel9:                       " Intel® Core™ 9",
	ProcessorFamilyIntelUltra3:                  " Intel® Core™ Ultra 3",
	ProcessorFamilyIntelUltra5:                  " Intel® Core™ Ultra 5",
	ProcessorFamilyIntelUltra7:                  " Intel® Core™ Ultra 7",
	ProcessorFamilyIntelUltra9:                  " Intel® Core™ Ultra 9",
}

func (v ProcessorFamily) String() string {
	if name, ok := processorFamilyStr[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

// ProcessorStatus is defined in DSP0134 3.8.0
type ProcessorStatus uint8

var processorStatusStr = []string{
	"Unknown",
	"Enabled",
	"Disabled By User",
	"Disabled By BIOS",
	"Idle",
	"Reserved5",
	"Reserved6",
	"Other",
}

func (v ProcessorStatus) String() string {
	if v&0x40 == 0 {
		return "Unpopulated"
	}
	return "Populated, " + processorStatusStr[v&7]
}

type ProcessorUpgrade uint8

const (
	ProcessorUpgradeOther                ProcessorUpgrade = 0x01 // Other
	ProcessorUpgradeUnknown              ProcessorUpgrade = 0x02 // Unknown
	ProcessorUpgradeDaughterBoard        ProcessorUpgrade = 0x03 // Daughter Board
	ProcessorUpgradeZIFSocket            ProcessorUpgrade = 0x04 // ZIF Socket
	ProcessorUpgradeReplaceablePiggyBack ProcessorUpgrade = 0x05 // Replaceable Piggy Back
	ProcessorUpgradeNone                 ProcessorUpgrade = 0x06 // None
	ProcessorUpgradeLIFSocket            ProcessorUpgrade = 0x07 // LIF Socket
	ProcessorUpgradeSlot1                ProcessorUpgrade = 0x08 // Slot 1
	ProcessorUpgradeSlot2                ProcessorUpgrade = 0x09 // Slot 2
	ProcessorUpgrade370pinSocket         ProcessorUpgrade = 0x0a // 370-pin Socket
	ProcessorUpgradeSlotA                ProcessorUpgrade = 0x0b // Slot A
	ProcessorUpgradeSlotM                ProcessorUpgrade = 0x0c // Slot M
	ProcessorUpgradeSocket423            ProcessorUpgrade = 0x0d // Socket 423
	ProcessorUpgradeSocketA              ProcessorUpgrade = 0x0e // Socket A (Socket 462)
	ProcessorUpgradeSocket478            ProcessorUpgrade = 0x0f // Socket 478
	ProcessorUpgradeSocket754            ProcessorUpgrade = 0x10 // Socket 754
	ProcessorUpgradeSocket940            ProcessorUpgrade = 0x11 // Socket 940
	ProcessorUpgradeSocket939            ProcessorUpgrade = 0x12 // Socket 939
	ProcessorUpgradeSocketMpga604        ProcessorUpgrade = 0x13 // Socket mPGA604
	ProcessorUpgradeSocketLGA771         ProcessorUpgrade = 0x14 // Socket LGA771
	ProcessorUpgradeSocketLGA775         ProcessorUpgrade = 0x15 // Socket LGA775
	ProcessorUpgradeSocketS1             ProcessorUpgrade = 0x16 // Socket S1
	ProcessorUpgradeSocketAM2            ProcessorUpgrade = 0x17 // Socket AM2
	ProcessorUpgradeSocketF1207          ProcessorUpgrade = 0x18 // Socket F (1207)
	ProcessorUpgradeSocketLGA1366        ProcessorUpgrade = 0x19 // Socket LGA1366
	ProcessorUpgradeSocketG34            ProcessorUpgrade = 0x1a // Socket G34
	ProcessorUpgradeSocketAM3            ProcessorUpgrade = 0x1b // Socket AM3
	ProcessorUpgradeSocketC32            ProcessorUpgrade = 0x1c // Socket C32
	ProcessorUpgradeSocketLGA1156        ProcessorUpgrade = 0x1d // Socket LGA1156
	ProcessorUpgradeSocketLGA1567        ProcessorUpgrade = 0x1e // Socket LGA1567
	ProcessorUpgradeSocketPGA988A        ProcessorUpgrade = 0x1f // Socket PGA988A
	ProcessorUpgradeSocketBGA1288        ProcessorUpgrade = 0x20 // Socket BGA1288
	ProcessorUpgradeSocketRpga988b       ProcessorUpgrade = 0x21 // Socket rPGA988B
	ProcessorUpgradeSocketBGA1023        ProcessorUpgrade = 0x22 // Socket BGA1023
	ProcessorUpgradeSocketBGA1224        ProcessorUpgrade = 0x23 // Socket BGA1224
	ProcessorUpgradeSocketBGA1155        ProcessorUpgrade = 0x24 // Socket BGA1155
	ProcessorUpgradeSocketLGA1356        ProcessorUpgrade = 0x25 // Socket LGA1356
	ProcessorUpgradeSocketLGA2011        ProcessorUpgrade = 0x26 // Socket LGA2011
	ProcessorUpgradeSocketFS1            ProcessorUpgrade = 0x27 // Socket FS1
	ProcessorUpgradeSocketFS2            ProcessorUpgrade = 0x28 // Socket FS2
	ProcessorUpgradeSocketFM1            ProcessorUpgrade = 0x29 // Socket FM1
	ProcessorUpgradeSocketFM2            ProcessorUpgrade = 0x2a // Socket FM2
	ProcessorUpgradeSocketLGA20113       ProcessorUpgrade = 0x2b // Socket LGA2011-3
	ProcessorUpgradeSocketLGA13563       ProcessorUpgrade = 0x2c // Socket LGA1356-3
	ProcessorUpgradeSocketLGA1150        ProcessorUpgrade = 0x2d // Socket LGA1150
	ProcessorUpgradeSocketBGA1168        ProcessorUpgrade = 0x2e // Socket BGA1168
	ProcessorUpgradeSocketBGA1234        ProcessorUpgrade = 0x2f // Socket BGA1234
	ProcessorUpgradeSocketBGA1364        ProcessorUpgrade = 0x30 // Socket BGA1364
	ProcessorUpgradeSocketAM4            ProcessorUpgrade = 0x31 // Socket AM4
	ProcessorUpgradeSocketLGA1151        ProcessorUpgrade = 0x32 // Socket LGA1151
	ProcessorUpgradeSocketBGA1356        ProcessorUpgrade = 0x33 // Socket BGA1356
	ProcessorUpgradeSocketBGA1440        ProcessorUpgrade = 0x34 // Socket BGA1440
	ProcessorUpgradeSocketBGA1515        ProcessorUpgrade = 0x35 // Socket BGA1515
	ProcessorUpgradeSocketLGA36471       ProcessorUpgrade = 0x36 // Socket LGA3647-1
	ProcessorUpgradeSocketSP3            ProcessorUpgrade = 0x37 // Socket SP3
	ProcessorUpgradeSocketSP3r2          ProcessorUpgrade = 0x38 // Socket SP3r2
	ProcessorUpgradeSocketLGA2066        ProcessorUpgrade = 0x39 // Socket LGA2066
	ProcessorUpgradeSocketBGA1392        ProcessorUpgrade = 0x3a // Socket BGA1392
	ProcessorUpgradeSocketBGA1510        ProcessorUpgrade = 0x3b // Socket BGA1510
	ProcessorUpgradeSocketBGA1528        ProcessorUpgrade = 0x3c // Socket BGA1528
	ProcessorUpgradeSocketLGA4189        ProcessorUpgrade = 0x3D // Socket LGA4189
	ProcessorUpgradeSocketLGA1200        ProcessorUpgrade = 0x3E // Socket LGA1200
	ProcessorUpgradeSocketLGA4677        ProcessorUpgrade = 0x3F // Socket LGA4677
	ProcessorUpgradeSocketLGA1700        ProcessorUpgrade = 0x40 // Socket LGA1700
	ProcessorUpgradeSocketBGA1744        ProcessorUpgrade = 0x41 // Socket BGA1744
	ProcessorUpgradeSocketBGA1781        ProcessorUpgrade = 0x42 // Socket BGA1781
	ProcessorUpgradeSocketBGA1211        ProcessorUpgrade = 0x43 // Socket BGA1211
	ProcessorUpgradeSocketBGA2422        ProcessorUpgrade = 0x44 // Socket BGA2422
	ProcessorUpgradeSocketLGA1211        ProcessorUpgrade = 0x45 // Socket LGA1211
	ProcessorUpgradeSocketLGA2422        ProcessorUpgrade = 0x46 // Socket LGA2422
	ProcessorUpgradeSocketLGA5773        ProcessorUpgrade = 0x47 // Socket LGA5773
	ProcessorUpgradeSocketBGA5773        ProcessorUpgrade = 0x48 // Socket BGA5773
	ProcessorUpgradeSocketAM5            ProcessorUpgrade = 0x49 // Socket AM5
	ProcessorUpgradeSocketSP5            ProcessorUpgrade = 0x4A // Socket SP5
	ProcessorUpgradeSocketSP6            ProcessorUpgrade = 0x4B // Socket SP6
	ProcessorUpgradeSocketBGA883         ProcessorUpgrade = 0x4C // Socket BGA883
	ProcessorUpgradeSocketBGA1190        ProcessorUpgrade = 0x4D // Socket BGA1190
	ProcessorUpgradeSocketBGA4129        ProcessorUpgrade = 0x4E // Socket BGA4129
	ProcessorUpgradeSocketLGA4710        ProcessorUpgrade = 0x4F // Socket LGA4710
	ProcessorUpgradeSocketLGA7529        ProcessorUpgrade = 0x50 // Socket LGA7529
	ProcessorUpgradeSocketBGA1964        ProcessorUpgrade = 0x51 // Socket BGA1964
	ProcessorUpgradeSocketBGA1792        ProcessorUpgrade = 0x52 // Socket BGA1792
	ProcessorUpgradeSocketBGA2049        ProcessorUpgrade = 0x53 // Socket BGA2049
	ProcessorUpgradeSocketBGA2551        ProcessorUpgrade = 0x54 // Socket BGA2551
	ProcessorUpgradeSocketLGA1851        ProcessorUpgrade = 0x55 // Socket LGA1851
	ProcessorUpgradeSocketBGA2114        ProcessorUpgrade = 0x56 // Socket BGA2114
	ProcessorUpgradeSocketBGA2833        ProcessorUpgrade = 0x57 // Socket BGA2833
)

var processorUpgradeStr = map[ProcessorUpgrade]string{
	ProcessorUpgradeOther:                "Other",
	ProcessorUpgradeUnknown:              "Unknown",
	ProcessorUpgradeDaughterBoard:        "Daughter Board",
	ProcessorUpgradeZIFSocket:            "ZIF Socket",
	ProcessorUpgradeReplaceablePiggyBack: "Replaceable Piggy Back",
	ProcessorUpgradeNone:                 "None",
	ProcessorUpgradeLIFSocket:            "LIF Socket",
	ProcessorUpgradeSlot1:                "Slot 1",
	ProcessorUpgradeSlot2:                "Slot 2",
	ProcessorUpgrade370pinSocket:         "370-pin Socket",
	ProcessorUpgradeSlotA:                "Slot A",
	ProcessorUpgradeSlotM:                "Slot M",
	ProcessorUpgradeSocket423:            "Socket 423",
	ProcessorUpgradeSocketA:              "Socket A (Socket 462)",
	ProcessorUpgradeSocket478:            "Socket 478",
	ProcessorUpgradeSocket754:            "Socket 754",
	ProcessorUpgradeSocket940:            "Socket 940",
	ProcessorUpgradeSocket939:            "Socket 939",
	ProcessorUpgradeSocketMpga604:        "Socket mPGA604",
	ProcessorUpgradeSocketLGA771:         "Socket LGA771",
	ProcessorUpgradeSocketLGA775:         "Socket LGA775",
	ProcessorUpgradeSocketS1:             "Socket S1",
	ProcessorUpgradeSocketAM2:            "Socket AM2",
	ProcessorUpgradeSocketF1207:          "Socket F (1207)",
	ProcessorUpgradeSocketLGA1366:        "Socket LGA1366",
	ProcessorUpgradeSocketG34:            "Socket G34",
	ProcessorUpgradeSocketAM3:            "Socket AM3",
	ProcessorUpgradeSocketC32:            "Socket C32",
	ProcessorUpgradeSocketLGA1156:        "Socket LGA1156",
	ProcessorUpgradeSocketLGA1567:        "Socket LGA1567",
	ProcessorUpgradeSocketPGA988A:        "Socket PGA988A",
	ProcessorUpgradeSocketBGA1288:        "Socket BGA1288",
	ProcessorUpgradeSocketRpga988b:       "Socket rPGA988B",
	ProcessorUpgradeSocketBGA1023:        "Socket BGA1023",
	ProcessorUpgradeSocketBGA1224:        "Socket BGA1224",
	ProcessorUpgradeSocketBGA1155:        "Socket BGA1155",
	ProcessorUpgradeSocketLGA1356:        "Socket LGA1356",
	ProcessorUpgradeSocketLGA2011:        "Socket LGA2011",
	ProcessorUpgradeSocketFS1:            "Socket FS1",
	ProcessorUpgradeSocketFS2:            "Socket FS2",
	ProcessorUpgradeSocketFM1:            "Socket FM1",
	ProcessorUpgradeSocketFM2:            "Socket FM2",
	ProcessorUpgradeSocketLGA20113:       "Socket LGA2011-3",
	ProcessorUpgradeSocketLGA13563:       "Socket LGA1356-3",
	ProcessorUpgradeSocketLGA1150:        "Socket LGA1150",
	ProcessorUpgradeSocketBGA1168:        "Socket BGA1168",
	ProcessorUpgradeSocketBGA1234:        "Socket BGA1234",
	ProcessorUpgradeSocketBGA1364:        "Socket BGA1364",
	ProcessorUpgradeSocketAM4:            "Socket AM4",
	ProcessorUpgradeSocketLGA1151:        "Socket LGA1151",
	ProcessorUpgradeSocketBGA1356:        "Socket BGA1356",
	ProcessorUpgradeSocketBGA1440:        "Socket BGA1440",
	ProcessorUpgradeSocketBGA1515:        "Socket BGA1515",
	ProcessorUpgradeSocketLGA36471:       "Socket LGA3647-1",
	ProcessorUpgradeSocketSP3:            "Socket SP3",
	ProcessorUpgradeSocketSP3r2:          "Socket SP3r2",
	ProcessorUpgradeSocketLGA2066:        "Socket LGA2066",
	ProcessorUpgradeSocketBGA1392:        "Socket BGA1392",
	ProcessorUpgradeSocketBGA1510:        "Socket BGA1510",
	ProcessorUpgradeSocketBGA1528:        "Socket BGA1528",
	ProcessorUpgradeSocketLGA4189:        "Socket LGA4189",
	ProcessorUpgradeSocketLGA1200:        "Socket LGA1200",
	ProcessorUpgradeSocketLGA4677:        "Socket LGA4677",
	ProcessorUpgradeSocketLGA1700:        "Socket LGA1700",
	ProcessorUpgradeSocketBGA1744:        "Socket BGA1744",
	ProcessorUpgradeSocketBGA1781:        "Socket BGA1781",
	ProcessorUpgradeSocketBGA1211:        "Socket BGA1211",
	ProcessorUpgradeSocketBGA2422:        "Socket BGA2422",
	ProcessorUpgradeSocketLGA1211:        "Socket LGA1211",
	ProcessorUpgradeSocketLGA2422:        "Socket LGA2422",
	ProcessorUpgradeSocketLGA5773:        "Socket LGA5773",
	ProcessorUpgradeSocketBGA5773:        "Socket BGA5773",
	ProcessorUpgradeSocketAM5:            "Socket AM5",
	ProcessorUpgradeSocketSP5:            "Socket SP5",
	ProcessorUpgradeSocketSP6:            "Socket SP6",
	ProcessorUpgradeSocketBGA883:         "Socket BGA883",
	ProcessorUpgradeSocketBGA1190:        "Socket BGA1190",
	ProcessorUpgradeSocketBGA4129:        "Socket BGA4129",
	ProcessorUpgradeSocketLGA4710:        "Socket LGA4710",
	ProcessorUpgradeSocketLGA7529:        "Socket LGA7529",
	ProcessorUpgradeSocketBGA1964:        "Socket BGA1964",
	ProcessorUpgradeSocketBGA1792:        "Socket BGA1792",
	ProcessorUpgradeSocketBGA2049:        "Socket BGA2049",
	ProcessorUpgradeSocketBGA2551:        "Socket BGA2551",
	ProcessorUpgradeSocketLGA1851:        "Socket LGA1851",
	ProcessorUpgradeSocketBGA2114:        "Socket BGA2114",
	ProcessorUpgradeSocketBGA2833:        "Socket BGA2833",
}

func (v ProcessorUpgrade) String() string {
	if name, ok := processorUpgradeStr[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

// ProcessorCharacteristics values are defined in DSP0134 7.5.9.
type ProcessorCharacteristics uint16

// ProcessorCharacteristics fields are defined in DSP0134 x.x.x.
const (
	ProcessorCharacteristicsReserved                ProcessorCharacteristics = 1 << 0 // Reserved
	ProcessorCharacteristicsUnknown                 ProcessorCharacteristics = 1 << 1 // Unknown
	ProcessorCharacteristics64bitCapable            ProcessorCharacteristics = 1 << 2 // 64-bit Capable
	ProcessorCharacteristicsMultiCore               ProcessorCharacteristics = 1 << 3 // Multi-Core
	ProcessorCharacteristicsHardwareThread          ProcessorCharacteristics = 1 << 4 // Hardware Thread
	ProcessorCharacteristicsExecuteProtection       ProcessorCharacteristics = 1 << 5 // Execute Protection
	ProcessorCharacteristicsEnhancedVirtualization  ProcessorCharacteristics = 1 << 6 // Enhanced Virtualization
	ProcessorCharacteristicsPowerPerformanceControl ProcessorCharacteristics = 1 << 7 // Power/Performance Control
	ProcessorCharacteristics128bitCapable           ProcessorCharacteristics = 1 << 8 // 128-bit Capable
	ProcessorCharacteristicsArm64SoCID              ProcessorCharacteristics = 1 << 9 // ARM64 SoC ID
)

var procChars = map[ProcessorCharacteristics]string{
	ProcessorCharacteristicsReserved:                "Reserved",
	ProcessorCharacteristicsUnknown:                 "Unknown",
	ProcessorCharacteristics64bitCapable:            "64-bit capable",
	ProcessorCharacteristicsMultiCore:               "Multi-Core",
	ProcessorCharacteristicsHardwareThread:          "Hardware Thread",
	ProcessorCharacteristicsExecuteProtection:       "Execute Protection",
	ProcessorCharacteristicsEnhancedVirtualization:  "Enhanced Virtualization",
	ProcessorCharacteristicsPowerPerformanceControl: "Power/Performance Control",
	ProcessorCharacteristics128bitCapable:           "128-bit capable",
	ProcessorCharacteristicsArm64SoCID:              "ARM64 SoC ID",
}

func (v ProcessorCharacteristics) StringList() []string {
	var lines []string
	for i := 0; i < 10; i++ {
		if v&(1<<i) != 0 {
			lines = append(lines, procChars[1<<i])
		}
	}
	return lines
}
