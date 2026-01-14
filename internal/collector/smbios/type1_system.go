package smbios

import (
	"bytes"
	"fmt"
)

type Type1System struct {
	Header       `smbios:"-"`
	Manufacturer string
	ProductName  string
	Version      string
	SerialNumber string
	UUID         UUID
	WakeUpType   WakeUpType
	SKU          string
	Family       string
}

func parseType1System(t *Table) (*Type1System, error) {
	if t.Type != 1 {
		return nil, fmt.Errorf("%s: %d", ErrInvalidTableType, t.Type)
	}
	if t.Header.Length < 0x8 {
		return nil, fmt.Errorf("%s: system info table must be at least %d bytes", ErrInvalidTableLength, 8)
	}

	s := &Type1System{Header: t.Header}
	if _, err := parseType(t, 0, false, s); err != nil {
		return nil, fmt.Errorf("parse type 1 system error: %w", err)
	}

	return s, nil
}

type UUID [16]byte

func (u *UUID) parseField(t *Table, off int) (int, error) {
	ub, err := t.GetBytesAt(off, 16)
	if err != nil {
		return off, err
	}
	copy(u[:], ub)
	return off + 16, nil
}

func (u UUID) String() string {
	if bytes.Equal(u[:], []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
		return "Not Present"
	}
	if bytes.Equal(u[:], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) {
		return "Not Settable"
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		u[3], u[2], u[1], u[0],
		u[5], u[4],
		u[7], u[6],
		u[8], u[9],
		u[10], u[11], u[12], u[13], u[14], u[15],
	)
}

type WakeUpType uint8

const (
	WakeUpTypeReserved WakeUpType = iota
	WakeUpTypeOther
	WakeUpTypeUnknown
	WakeUpTypeAPMTimer
	WakeUpTypeModemRing
	WakeUpTypeLANRemote
	WakeUpTypePowerSwitch
	WakeUpTypePCIPME
	WakeUpTypeACPowerRestored
)

var wakeupStrings = map[WakeUpType]string{
	WakeUpTypeReserved:        "Reserved",
	WakeUpTypeOther:           "Other",
	WakeUpTypeUnknown:         "Unknown",
	WakeUpTypeAPMTimer:        "APM Timer",
	WakeUpTypeModemRing:       "Modem Ring",
	WakeUpTypeLANRemote:       "LAN Remote",
	WakeUpTypePowerSwitch:     "Power Switch",
	WakeUpTypePCIPME:          "PCI PME#",
	WakeUpTypeACPowerRestored: "AC Power Restored",
}

func (w WakeUpType) String() string {
	if str, ok := wakeupStrings[w]; ok {
		return str
	}
	return fmt.Sprintf("%#x", uint8(w))
}
