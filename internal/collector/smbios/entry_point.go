package smbios

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	intermediateAnchor = "_DMI_"

	checksum32Offset           = 4
	intermediateRegionStart    = 0x10
	intermediateRegionEnd      = 0x1F
	IntermediateChecksumOffset = 5
)

var (
	ErrInvalidAnchor   = errors.New("smbios: invalid anchor string")
	ErrInvalidLength   = errors.New("smbios: invalid entry point length")
	ErrInvalidChecksum = errors.New("smbios: invalid entry point checksum")
	ErrDataTooShort    = errors.New("smbios: data too short")
)

type EntryPoint interface {
	Table() (address int, length int)
	Version() (major int, minor int)
	MarshalBinary() ([]byte, error)
	UnmarshalBinary(data []byte) error
}

func parseEntryPoint(r io.Reader) (EntryPoint, error) {
	bf := bufio.NewReader(r)
	peek, err := bf.Peek(5)
	if err != nil {
		return nil, fmt.Errorf("peek anchor: %w", err)
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
		return nil, fmt.Errorf("%w: got %q", ErrInvalidAnchor, peek)
	}

	if _, err := io.ReadFull(bf, data); err != nil {
		return nil, fmt.Errorf("read entry point: %w", err)
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
	return -sum
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

func (e *entryPoint32) Table() (address int, length int) {
	return int(e.TableAddress), int(e.TableLength)
}

func (e *entryPoint32) Version() (major int, minor int) {
	return int(e.MajorVersion), int(e.MinorVersion)
}

func (e *entryPoint32) MarshalBinary() ([]byte, error) {
	buf := make([]byte, anchor32Len)
	writer := bytes.NewBuffer(buf[:0])
	if err := binary.Write(writer, binary.LittleEndian, e); err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func (e *entryPoint32) UnmarshalBinary(data []byte) error {
	if len(data) < anchor32Len {
		return fmt.Errorf("%w: got %d,need %d", ErrDataTooShort, len(data), anchor32Len)
	}

	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, e); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	if !bytes.Equal(e.AnchorString[:], []byte(anchor32)) {
		return fmt.Errorf("%w: %q", ErrInvalidAnchor, e.AnchorString[:])
	}

	if e.Length != anchor32Len {
		return fmt.Errorf("%w: got %d,expected %d", ErrInvalidLength, e.Length, anchor32Len)
	}

	if e.Checksum != calChecksum(data[:anchor32Len], checksum32Offset) {
		return fmt.Errorf("%w: header checksum mismatch", ErrInvalidChecksum)
	}

	if !bytes.Equal(e.IntermediateAnchorString[:], []byte(intermediateAnchor)) {
		return fmt.Errorf("%w: intermediate anchor %q", ErrInvalidAnchor, e.IntermediateAnchorString[:])
	}

	intermediateRegion := data[intermediateRegionStart:intermediateRegionEnd]
	if e.IntermediateChecksum != calChecksum(intermediateRegion, IntermediateChecksumOffset) {
		return fmt.Errorf("%w: intermediate checksum mismatch", ErrInvalidChecksum)
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
	MaximumStructureSize  uint32
	TableAddress          uint64
}

func (e *entryPoint64) Table() (address int, length int) {
	return int(e.TableAddress), int(e.MaximumStructureSize)
}

func (e *entryPoint64) Version() (major int, minor int) {
	return int(e.MajorVersion), int(e.MinorVersion)
}

func (e *entryPoint64) MarshalBinary() ([]byte, error) {
	buf := make([]byte, anchor64Len)
	writer := bytes.NewBuffer(buf[:0])
	if err := binary.Write(writer, binary.LittleEndian, e); err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func (e *entryPoint64) UnmarshalBinary(data []byte) error {
	if len(data) < anchor64Len {
		return fmt.Errorf("%w: got %d,need %d", ErrDataTooShort, len(data), anchor64Len)
	}

	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, e); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	if !bytes.Equal(e.AnchorString[:], []byte(anchor64)) {
		return fmt.Errorf("%w: %q", ErrInvalidAnchor, e.AnchorString[:])
	}

	if e.Length != anchor64Len {
		return fmt.Errorf("%w: got %d,expected %d", ErrInvalidLength, e.Length, anchor64Len)
	}

	if e.Checksum != calChecksum(data[:anchor64Len], 5) {
		return fmt.Errorf("%w: header checksum mismatch", ErrInvalidChecksum)
	}

	return nil
}
