package smbios

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Table struct {
	Header
	FormattedArea []byte
	StringArea    []string
}

func parseTables(r io.Reader) ([]*Table, error) {
	var tables []*Table
	br := bufio.NewReader(r)
	for {
		if _, err := br.Peek(1); err == io.EOF {
			return tables, nil
		}
		t, err := parseTable(br)
		if err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
}

func parseTable(br *bufio.Reader) (*Table, error) {
	var h Header
	if err := h.UnmarshalBinary(br); err != nil {
		return nil, fmt.Errorf("unmarshal header error: %w", err)
	}

	l := int(h.Length) - headerLength
	fa, err := readFormattedArea(br, l)
	if err != nil {
		return nil, err
	}

	sa, err := readStringArea(br)
	if err != nil {
		return nil, err
	}

	return &Table{
		Header:        h,
		FormattedArea: fa,
		StringArea:    sa,
	}, nil
}

func readFormattedArea(br *bufio.Reader, l int) ([]byte, error) {
	if l < 0 {
		return nil, fmt.Errorf("invalid formatted area length: %d", l)
	}

	if l == 0 {
		return nil, nil
	}

	b := make([]byte, l)
	if _, err := io.ReadFull(br, b); err != nil {
		return nil, fmt.Errorf("read formatted area error: %w", err)
	}

	return b, nil
}

func readStringArea(br *bufio.Reader) ([]string, error) {
	peek, err := br.Peek(2)
	if err != nil {
		return nil, fmt.Errorf("peek strings area error: %w", err)
	}

	if bytes.Equal(peek, []byte{0x00, 0x00}) {
		br.Discard(2)
		return nil, nil
	}

	var ss []string
	for {
		b, err := br.ReadBytes(0)
		if err != nil {
			return nil, fmt.Errorf("read strings delimiter error: %w", err)
		}
		b = bytes.TrimRight(b, "\x00")
		ss = append(ss, string(b))
		if p, err := br.Peek(1); err != nil {
			return nil, fmt.Errorf("peek strings delimiter error: %w", err)
		} else if bytes.Equal(p, []byte{0x00}) {
			br.Discard(1)
			return ss, nil
		}
	}
}

func (t *Table) GetStringAt(offset int) (string, error) {
	if offset < 0 || offset >= len(t.FormattedArea) {
		return "", fmt.Errorf("%w at offset %d", io.ErrUnexpectedEOF, offset)
	}

	index := t.FormattedArea[offset]
	switch {
	case index == 0:
		return "", nil
	case int(index) <= len(t.StringArea):
		return t.StringArea[index-1], nil
	default:
		return "<BAD INDEX>", fmt.Errorf("%w: string index %d beyond end of string table", io.ErrUnexpectedEOF, index)
	}
}

func (t *Table) GetByteAt(offset int) (uint8, error) {
	if offset < 0 || offset > len(t.FormattedArea) {
		return 0, fmt.Errorf("%w at offset %d", io.ErrUnexpectedEOF, offset)
	}

	return t.FormattedArea[offset], nil
}

func (t *Table) GetWordAt(offset int) (uint16, error) {
	if offset < 0 || offset+2 > len(t.FormattedArea) {
		return 0, fmt.Errorf("%w at offset %d with length 2", io.ErrUnexpectedEOF, offset)
	}

	return binary.LittleEndian.Uint16(t.FormattedArea[offset : offset+2]), nil
}

func (t *Table) GetDwordAt(offset int) (uint32, error) {
	if offset < 0 || offset+4 > len(t.FormattedArea) {
		return 0, fmt.Errorf("%w at offset %d with length 4", io.ErrUnexpectedEOF, offset)
	}

	return binary.LittleEndian.Uint32(t.FormattedArea[offset : offset+4]), nil
}

func (t *Table) GetQwordAt(offset int) (uint64, error) {
	if offset < 0 || offset+8 > len(t.FormattedArea) {
		return 0, fmt.Errorf("%w at offset %d with length 8,formattedArea length:%d", io.ErrUnexpectedEOF, offset, len(t.FormattedArea))
	}

	return binary.LittleEndian.Uint64(t.FormattedArea[offset : offset+8]), nil
}

func (t *Table) GetBytesAt(offset, length int) ([]byte, error) {
	if offset < 0 || offset+length > len(t.FormattedArea) {
		return nil, fmt.Errorf("%w at offset %d with length %d", io.ErrUnexpectedEOF, offset, length)
	}

	return t.FormattedArea[offset : offset+length], nil
}

func (t *Table) ReadAt(p []byte, off int64) (int, error) {
	if int(off) > len(t.FormattedArea)-len(p) {
		return 0, fmt.Errorf("%w at offset %d with length %d", io.ErrUnexpectedEOF, off, len(p))
	}
	n := copy(p, t.FormattedArea[int(off):])
	return n, nil
}
