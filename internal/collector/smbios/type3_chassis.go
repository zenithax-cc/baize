package smbios

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Type3Chassis struct {
	Header             `smbios:"-"`
	Manufacturer       string
	ChassisType        ChassisType
	Version            string
	SerialNumber       string
	AssetTag           string
	BootupState        ChassisState
	PowerSupplyState   ChassisState
	ThermalState       ChassisState
	SecurityStatus     ChassisSecurityStatus
	OEMDefined         uint32
	Height             uint8
	NumberOfPowerCords uint8
	ContainedElements  ChassisContainedElements
	SKU                string
}

type ChassisContainedElement struct {
	Type ChassisElementType
	Min  uint8
	Max  uint8
}

func (cce *ChassisContainedElement) String() string {
	return fmt.Sprintf("%d %d-%d", cce.Type, cce.Min, cce.Max)
}

func parseType3Chassis(t *Table) (*Type3Chassis, error) {
	if t.Type != 3 {
		return nil, fmt.Errorf("%s:%d", ErrInvalidTableType, t.Type)
	}
	if t.Header.Length < 0x9 {
		return nil, fmt.Errorf("%s: chassis info table must be at least %d bytes", ErrInvalidTableLength, 0x9)
	}
	chassis := &Type3Chassis{
		Header: t.Header,
	}
	if _, err := parseType(t, 0, false, chassis); err != nil {
		return nil, fmt.Errorf("parse type 3 chassis error: %w", err)
	}
	return chassis, nil
}

type ChassisType uint8

const (
	ChassisTypeOther ChassisType = iota + 1
	ChassisUnknown
	ChassisTypeDesktop
	ChassisTypeLowProfileDesktop
	ChassisTypePizzaBox
	ChassisTypeMiniTower
	ChassisTypeTower
	ChassisTypePortable
	ChassisTypeLaptop
	ChassisTypeNotebook
	ChassisTypeHandHeld
	ChassisTypeDockingStation
	ChassisTypeAllInOne
	ChassisTypeSubNotebook
	ChassisTypeSpacesaving
	ChassisTypeLunchBox
	ChassisTypeMainServerChassis
	ChassisTypeExpansionChassis
	ChassisTypeSubChassis
	ChassisTypeBusExpansionChassis
	ChassisTypePeripheralChassis
	ChassisTypeRAIDChassis
	ChassisTypeRackMountChassis
	ChassisTypeSealedcasePC
	ChassisTypeMultisystemChassis
	ChassisTypeCompactPCI
	ChassisTypeAdvancedTCA
	ChassisTypeBlade
	ChassisTypeBladeChassis
	ChassisTypeTablet
	ChassisTypeConvertible
	ChassisTypeDetachable
	ChassisTypeIoTGateway
	ChassisTypeEmbeddedPC
	ChassisTypeMiniPC
	ChassisTypeStickPC
)

var chassisTypeStr = []string{
	"Other",
	"Unknown",
	"Desktop",
	"Low Profile Desktop",
	"Pizza Box",
	"Mini Tower",
	"Tower",
	"Portable",
	"Laptop",
	"Notebook",
	"Hand Held",
	"Docking Station",
	"All In One",
	"Sub Notebook",
	"Space-saving",
	"Lunch Box",
	"Main Server Chassis",
	"Expansion Chassis",
	"Sub Chassis",
	"Bus Expansion Chassis",
	"Peripheral Chassis",
	"RAID Chassis",
	"Rack Mount Chassis",
	"Sealed-case PC",
	"Multi-system",
	"CompactPCI",
	"AdvancedTCA",
	"Blade",
	"Blade Chassis",
	"Tablet",
	"Convertible",
	"Detachable",
	"IoT Gateway",
	"Embedded PC",
	"Mini PC",
	"Stick PC",
}

func (v ChassisType) String() string {
	idx := v&0x7f - 1
	if int(idx) < len(chassisTypeStr) {
		return chassisTypeStr[idx]
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type ChassisState uint8

const (
	ChassisStateOther ChassisState = iota + 1
	ChassisStateUnknown
	ChassisStateSafe
	ChassisStateWarning
	ChassisStateCritical
	ChassisStateNonrecoverable
)

var chassisStateStr = map[ChassisState]string{
	ChassisStateOther:          "Other",
	ChassisStateUnknown:        "Unknown",
	ChassisStateSafe:           "Safe",
	ChassisStateWarning:        "Warning",
	ChassisStateCritical:       "Critical",
	ChassisStateNonrecoverable: "Non-recoverable",
}

func (v ChassisState) String() string {
	if name, ok := chassisStateStr[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type ChassisSecurityStatus uint8

const (
	ChassisSecurityStatusOther ChassisSecurityStatus = iota + 1
	ChassisSecurityStatusUnknown
	ChassisSecurityStatusNone
	ChassisSecurityStatusExternalInterfaceLockedOut
	ChassisSecurityStatusExternalInterfaceEnabled
)

var chassisSecurityStatusStr = map[ChassisSecurityStatus]string{
	ChassisSecurityStatusOther:                      "Other",
	ChassisSecurityStatusUnknown:                    "Unknown",
	ChassisSecurityStatusNone:                       "None",
	ChassisSecurityStatusExternalInterfaceLockedOut: "External Interface Locked Out",
	ChassisSecurityStatusExternalInterfaceEnabled:   "External Interface Enabled",
}

func (v ChassisSecurityStatus) String() string {
	if name, ok := chassisSecurityStatusStr[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type ChassisElementType uint8
type ChassisContainedElements []ChassisContainedElement

func (cce *ChassisContainedElements) parseField(t *Table, offset int) (int, error) {
	num, err := t.GetByteAt(offset)
	if err != nil {
		return offset, err
	}
	offset++

	size, err := t.GetByteAt(offset)
	if err != nil {
		return offset, err
	}
	offset++

	if num == 0 || size == 0 {
		return offset, nil
	}
	if size != 3 {
		return offset, fmt.Errorf("invalid size %d for chassis contained elements,surpport 3", size)
	}

	for i := 0; i < int(num); i++ {
		var elem ChassisContainedElement
		if err := binary.Read(io.NewSectionReader(t, int64(offset), 3), binary.LittleEndian, &elem); err != nil {
			return offset, err
		}
		*cce = append(*cce, elem)
		offset += 3
	}
	return offset, nil
}
