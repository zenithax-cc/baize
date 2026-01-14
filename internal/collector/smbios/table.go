package smbios

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	TableTypeEndOfTable = 0x7F
	defaultTableCap     = 64
	defaultStringCap    = 8
)

var (
	ErrInvalidOffset      = errors.New("smbios: invalid offset")
	ErrInvalidTableLength = errors.New("smbios: invalid table length")
	ErrNegativeOffset     = errors.New("smbios: negative offset")
	ErrStringIndexOOB     = errors.New("smbios: string index out of bounds")
	ErrInvalidTableData   = errors.New("smbios: invalid table data")
)

type Table struct {
	Header
	FormattedArea []byte
	StringArea    []string
}

func parseTables(r io.Reader) ([]*Table, error) {
	tables := make([]*Table, 0, defaultTableCap)
	br := bufio.NewReader(r)

	for {
		peek, err := br.Peek(1)
		if err != nil {
			if err == io.EOF {
				return tables, nil
			}
			return nil, fmt.Errorf("peek table: %w", err)
		}

		if peek[0] == TableTypeEndOfTable {
			t, err := parseTable(br)
			if err != nil {
				return nil, fmt.Errorf("parse table: %w", err)
			}
			tables = append(tables, t)
			return tables, nil
		}

		t, err := parseTable(br)
		if err != nil {
			return nil, fmt.Errorf("parse table: %w", err)
		}
		tables = append(tables, t)
	}
}

func parseTable(br *bufio.Reader) (*Table, error) {
	var h Header
	if err := h.UnmarshalBinary(br); err != nil {
		return nil, fmt.Errorf("unmarshal header: %w", err)
	}

	faLen := int(h.Length) - headerLength
	fa, err := readFormattedArea(br, faLen)
	if err != nil {
		return nil, fmt.Errorf("read formatted area: %w", err)
	}

	sa, err := readStringArea(br)
	if err != nil {
		return nil, fmt.Errorf("read string area: %w", err)
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
		return nil, err
	}

	return b, nil
}

func readStringArea(br *bufio.Reader) ([]string, error) {
	peek, err := br.Peek(2)
	if err != nil {
		return nil, fmt.Errorf("peek strings area error: %w", err)
	}

	if peek[0] == 0x00 && peek[1] == 0x00 {
		_, _ = br.Discard(2)
		return []string{}, nil
	}

	ss := make([]string, 0, defaultStringCap)
	for {
		b, err := br.ReadBytes(0x00)
		if err != nil {
			return nil, fmt.Errorf("read strings delimiter error: %w", err)
		}

		if len(b) > 1 {
			ss = append(ss, string(b[:len(b)-1]))
		}

		p, err := br.Peek(1)
		if err != nil {
			return nil, fmt.Errorf("peek strings terminator: %w", err)
		}
		if p[0] == 0x00 {
			_, _ = br.Discard(1)
			return ss, nil
		}
	}
}

func (t *Table) checkBounds(offset, length int) error {
	if offset < 0 {
		return fmt.Errorf("%w: offset %d", ErrNegativeOffset, offset)
	}

	if length < 0 {
		return fmt.Errorf("%w: length %d", ErrInvalidTableLen, length)
	}

	if offset > len(t.FormattedArea) || length > len(t.FormattedArea)-offset {
		return fmt.Errorf("%w: offset=%d, length=%d, available=%d", ErrInvalidOffset,
			offset, length, len(t.FormattedArea))
	}

	return nil
}

func (t *Table) GetStringAt(offset int) (string, error) {
	if err := t.checkBounds(offset, 1); err != nil {
		return "", err
	}

	index := t.FormattedArea[offset]
	if index == 0 {
		return "", nil
	}

	if int(index) > len(t.StringArea) {
		return "<BAD INDEX>", fmt.Errorf("%w: string index %d beyond end of string table", ErrInvalidTableData, index)
	}

	return t.StringArea[index-1], nil
}

func (t *Table) GetByteAt(offset int) (uint8, error) {
	if err := t.checkBounds(offset, 1); err != nil {
		return 0, err
	}

	return t.FormattedArea[offset], nil
}

func (t *Table) GetWordAt(offset int) (uint16, error) {
	if err := t.checkBounds(offset, 2); err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(t.FormattedArea[offset : offset+2]), nil
}

func (t *Table) GetDwordAt(offset int) (uint32, error) {
	if err := t.checkBounds(offset, 4); err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(t.FormattedArea[offset : offset+4]), nil
}

func (t *Table) GetQwordAt(offset int) (uint64, error) {
	if err := t.checkBounds(offset, 8); err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(t.FormattedArea[offset : offset+8]), nil
}

func (t *Table) GetBytesAt(offset, length int) ([]byte, error) {
	if err := t.checkBounds(offset, length); err != nil {
		return nil, err
	}

	result := make([]byte, length)
	copy(result, t.FormattedArea[offset:offset+length])

	return result, nil
}

func (t *Table) GetBytesAtUnsafe(offset, length int) ([]byte, error) {
	if err := t.checkBounds(offset, length); err != nil {
		return nil, err
	}

	return t.FormattedArea[offset : offset+length], nil
}

func (t *Table) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("%w: offset %d", ErrNegativeOffset, off)
	}

	dataLen := int64(len(t.FormattedArea))
	if off >= dataLen {
		return 0, io.EOF
	}

	n := copy(p, t.FormattedArea[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}
