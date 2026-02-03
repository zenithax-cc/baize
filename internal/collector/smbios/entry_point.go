package smbios

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	anchor32    = "_SM_"
	anchor64    = "_SM3_"
	anchor32Len = 0x1f
	anchor64Len = 0x18
)

type EntryPoint interface {
	Table() (int, int)
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}

func parseEntryPoint(r io.Reader) (EntryPoint, error) {
	bf := bufio.NewReader(r)
	peek, err := bf.Peek(5)
	if err != nil {
		return nil, err
	}
	var eps EntryPoint
	var data []byte
	switch {
	case bytes.Equal(peek[:4], []byte(anchor32)):
		data = make([]byte, anchor32Len)
		eps = &entryPoint32{}
	case bytes.Equal(peek, []byte(anchor64)):
		data = make([]byte, anchor64Len)
		eps = &entryPoint64{}
	default:
		return nil, fmt.Errorf("invalid anchor string:%v", string(peek[:]))
	}

	if _, err := io.ReadFull(bf, data); err != nil {
		return nil, err
	}

	if err := eps.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	return eps, nil
}

func calChecksum(data []byte, skipIndex int) uint8 {
	var sum uint8
	for i, b := range data {
		if i == skipIndex {
			continue
		}
		sum += b
	}
	return uint8(0x100 - int(sum))
}

type entryPoint32 struct {
	AnchorString             [4]uint8
	Checksum                 uint8
	Length                   uint8
	MajorVersion             uint8
	MinorVersion             uint8
	MaximumStructureSize     uint16
	Revision                 uint8
	FormattedArea            [5]uint8
	IntermediateAnchorString [5]uint8
	IntermediateChecksum     uint8
	TableLength              uint16
	TableAddress             uint32
	NumberOfStructures       uint16
	BCDRevision              uint8
}

func (e *entryPoint32) Table() (int, int) {
	return int(e.TableAddress), int(e.TableLength)
}

func (e *entryPoint32) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, e); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *entryPoint32) UnmarshalBinary(data []byte) error {
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, e); err != nil {
		return err
	}
	if !bytes.Equal(e.AnchorString[:], []byte(anchor32)) {
		return fmt.Errorf("invalid anchor string %s", e.AnchorString[:])
	}

	if e.Length != anchor32Len {
		return fmt.Errorf("invalid length %d", e.Length)
	}

	if e.Checksum != calChecksum(data, 4) {
		return fmt.Errorf("invalid checksum %d", e.Checksum)
	}

	if !bytes.Equal(e.IntermediateAnchorString[:], []byte("_DMI_")) {
		return fmt.Errorf("invalid intermediate anchor string %s", e.IntermediateAnchorString[:])
	}

	if e.IntermediateChecksum != calChecksum(data[0x10:0x1F], 5) {
		return fmt.Errorf("invalid intermediate checksum %d", e.IntermediateChecksum)
	}

	return nil
}

type entryPoint64 struct {
	AnchorString          [5]uint8
	Checksum              uint8
	Length                uint8
	MajorVersion          uint8
	MinorVersion          uint8
	DocumentationRevision uint8
	Revision              uint8
	Reserved              uint8
	MaximumStructureSize  uint16
	TableAddress          uint64
}

func (e *entryPoint64) Table() (int, int) {
	return int(e.TableAddress), int(e.MaximumStructureSize)
}

func (e *entryPoint64) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, e); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *entryPoint64) UnmarshalBinary(data []byte) error {
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, e); err != nil {
		return err
	}
	if !bytes.Equal(e.AnchorString[:], []byte(anchor64)) {
		return fmt.Errorf("invalid anchor string %s", e.AnchorString[:])
	}

	if e.Length != anchor64Len {
		return fmt.Errorf("invalid length %d", e.Length)
	}

	if e.Checksum != calChecksum(data, 5) {
		return fmt.Errorf("invalid checksum %d", e.Checksum)
	}

	return nil
}
