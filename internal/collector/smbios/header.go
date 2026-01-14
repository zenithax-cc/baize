package smbios

import (
	"encoding/binary"
	"io"
)

type Header struct {
	Type   uint8
	Length uint8
	Handle uint16
}

const headerLength = 4

func (h *Header) UnmarshalBinary(r io.Reader) error {
	if err := binary.Read(io.LimitReader(r, headerLength), binary.LittleEndian, h); err != nil {
		return err
	}
	return nil
}
