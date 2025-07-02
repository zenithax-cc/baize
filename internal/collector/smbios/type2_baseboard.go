package smbios

import (
	"fmt"
	"strings"
)

type Type2BaseBoard struct {
	Header        `smbios:"-"`
	Manufacturer  string
	Product       string
	Version       string
	SerialNumber  string
	AssetTag      string
	FeatureFlags  BoardFeatures
	Location      string
	ChassisHandle uint16
	BoardType     BoardType
	ObjectHandles ObjectHandles
}

func parseType2BaseBoard(t *Table) (*Type2BaseBoard, error) {
	if t.Type != 2 {
		return nil, fmt.Errorf("%s:%d", ErrInvalidTableType, t.Type)
	}
	if t.Header.Length < 8 {
		return nil, fmt.Errorf("%s: baseboard info table must be at least %d bytes", ErrInvalidTableLength, 8)
	}

	b := &Type2BaseBoard{
		Header: t.Header,
	}

	if _, err := parseType(t, 0, false, b); err != nil {
		return nil, fmt.Errorf("parse type 2 baseboard error: %w", err)
	}
	return b, nil
}

type BoardFeatures uint8

const (
	BoardFeaturesIsAHostingBoard                 BoardFeatures = 1 << 0
	BoardFeaturesRequiresAtLeastOneDaughterBoard BoardFeatures = 1 << 1
	BoardFeaturesIsRemovable                     BoardFeatures = 1 << 2
	BoardFeaturesIsReplaceable                   BoardFeatures = 1 << 3
	BoardFeaturesIsHotSwappable                  BoardFeatures = 1 << 4
)

var boardFeatureStr = map[BoardFeatures]string{
	BoardFeaturesIsAHostingBoard:                 "Board is a hosting board",
	BoardFeaturesRequiresAtLeastOneDaughterBoard: "Board requires at least one daughter board",
	BoardFeaturesIsRemovable:                     "Board is removable",
	BoardFeaturesIsReplaceable:                   "Board is replaceable",
	BoardFeaturesIsHotSwappable:                  "Board is hot swappable",
}

func (v BoardFeatures) String() string {
	var lines []string
	for i := 0; i < 5; i++ {
		if v&(1<<i) != 0 {
			lines = append(lines, boardFeatureStr[1<<i])
		}
	}
	return strings.Join(lines, ", ")
}

type BoardType uint8

const (
	BoardTypeUnknown BoardType = iota + 1
	BoardTypeOther
	BoardTypeServerBlade
	BoardTypeConnectivitySwitch
	BoardTypeSystemManagementModule
	BoardTypeProcessorModule
	BoardTypeIOModule
	BoardTypeMemoryModule
	BoardTypeDaughterBoard
	BoardTypeMotherboard
	BoardTypeProcessorMemoryModule
	BoardTypeProcessorIOModule
	BoardTypeInterconnectBoard
)

var boardTypeStr = map[BoardType]string{
	BoardTypeUnknown:                "Unknown",
	BoardTypeOther:                  "Other",
	BoardTypeServerBlade:            "Server Blade",
	BoardTypeConnectivitySwitch:     "Connectivity Switch",
	BoardTypeSystemManagementModule: "System Management Module",
	BoardTypeProcessorModule:        "Processor Module",
	BoardTypeIOModule:               "I/O Module",
	BoardTypeMemoryModule:           "Memory Module",
	BoardTypeDaughterBoard:          "Daughter board",
	BoardTypeMotherboard:            "Motherboard (includes processor, memory, and I/O)",
	BoardTypeProcessorMemoryModule:  "Processor/Memory Module",
	BoardTypeProcessorIOModule:      "Processor/IO Module",
	BoardTypeInterconnectBoard:      "Interconnect board",
}

func (v BoardType) String() string {
	if name, ok := boardTypeStr[v]; ok {
		return name
	}
	return fmt.Sprintf("%#x", uint8(v))
}

type ObjectHandles []uint16

func (oh *ObjectHandles) parseField(t *Table, offset int) (int, error) {
	num, err := t.GetByteAt(offset)
	if err != nil {
		return offset, err
	}
	offset++

	for i := uint8(0); i < num; i++ {
		h, err := t.GetWordAt(offset)
		if err != nil {
			return offset, err
		}
		*oh = append(*oh, h)
		offset += 2
	}
	return offset, nil
}
