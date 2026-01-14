package smbios

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	sysfsDMI        = "/sys/firmware/dmi/tables/DMI"
	sysfsEntryPoint = "/sys/firmware/dmi/tables/smbios_entry_point"
	devMem          = "/dev/mem"

	startAddr = 0x000F0000
	endAddr   = 0x00100000

	maxTableSize    = 1 << 20
	maxAddressValue = 0xFFFFFFFF
	readTimeout     = 10 * time.Second

	paragraphSize    = 1 << 4
	searchRegionSize = endAddr - startAddr

	anchor32    = "_SM_"
	anchor64    = "_SM3_"
	anchor32Len = 0x1f
	anchor64Len = 0x18
)

var (
	ErrReaderClosed       = errors.New("smbios: reader closed")
	ErrEntryPointNotFound = errors.New("smbios: entry point not found")
	ErrInvalidTableAddr   = errors.New("smbios: invalid table address")
	ErrInvalidTableLen    = errors.New("smbios: invalid table length")
	ErrAddressOverflow    = errors.New("smbios: address overflow")
)

type SMBIOSError struct {
	Op   string
	Path string
	Err  error
}

func (e *SMBIOSError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("smbios %s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("smbios %s: %v", e.Op, e.Err)
}

func (e *SMBIOSError) Unwrap() error {
	return e.Err
}

func (e *SMBIOSError) Is(target error) bool {
	t, ok := target.(*SMBIOSError)
	if !ok {
		return false
	}
	return (t.Op == "" || e.Op == t.Op) && (t.Path == "" || e.Path == t.Path)
}

type tableReader interface {
	readTables(ctx context.Context, tableAddr, tableLen int) ([]*Table, error)
	readEntryPoint(ctx context.Context) (EntryPoint, error)
	Close() error
}

type sysfsReader struct{}

func (r *sysfsReader) readTables(ctx context.Context, tableAddr, tableLen int) ([]*Table, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(sysfsDMI)
	if err != nil {
		return nil, &SMBIOSError{Op: "open", Path: sysfsDMI, Err: err}
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, int64(maxTableSize))
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, &SMBIOSError{Op: "read", Path: sysfsDMI, Err: err}
	}

	return parseTables(bytes.NewReader(data))
}

func (r *sysfsReader) readEntryPoint(ctx context.Context) (EntryPoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(sysfsEntryPoint)
	if err != nil {
		return nil, &SMBIOSError{Op: "open", Path: sysfsEntryPoint, Err: err}
	}
	defer file.Close()

	return parseEntryPoint(file)
}

func (r *sysfsReader) Close() error {
	return nil
}

type devMemReader struct {
	file   *os.File
	mu     sync.Mutex
	closed bool
}

func NewDevMemReader() (*devMemReader, error) {
	file, err := os.Open(devMem)
	if err != nil {
		return nil, &SMBIOSError{Op: "open", Path: devMem, Err: err}
	}

	return &devMemReader{file: file}, nil
}

func (r *devMemReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.closed && r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil
	r.closed = true
	return err
}

func (r *devMemReader) readTables(ctx context.Context, tableAddr, tableLen int) ([]*Table, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := validateTableParams(tableAddr, tableLen); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || r.file == nil {
		return nil, ErrReaderClosed
	}

	data, err := r.readAtLocked(int64(tableAddr), tableLen)
	if err != nil {
		return nil, err
	}

	return parseTables(bytes.NewReader(data))
}

func (r *devMemReader) readEntryPoint(ctx context.Context) (EntryPoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || r.file == nil {
		return nil, ErrReaderClosed
	}

	eps, err := r.searchEntryPointLocked(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := r.file.Seek(int64(eps), io.SeekStart); err != nil {
		return nil, &SMBIOSError{Op: "seek", Path: devMem, Err: err}
	}

	return parseEntryPoint(r.file)
}

func (r *devMemReader) readAtLocked(offset int64, length int) ([]byte, error) {
	if _, err := r.file.Seek(offset, io.SeekStart); err != nil {
		return nil, &SMBIOSError{Op: "seek", Path: devMem, Err: err}
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r.file, data); err != nil {
		return nil, &SMBIOSError{Op: "read", Path: devMem, Err: err}
	}

	return data, nil
}

func (r *devMemReader) searchEntryPointLocked(ctx context.Context) (int, error) {
	data, err := r.readAtLocked(startAddr, searchRegionSize)
	if err != nil {
		return 0, err
	}

	for offset := 0; offset+paragraphSize <= len(data); offset += paragraphSize {
		if offset&0xFFF == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}

		chunk := data[offset:]
		if bytes.HasPrefix(chunk, []byte(anchor32)) {
			return startAddr + offset, nil
		}

		if bytes.HasPrefix(chunk, []byte(anchor64)) {
			return startAddr + offset, nil
		}
	}

	return 0, fmt.Errorf("%w: scanned 0x%X - 0x%X", ErrEntryPointNotFound, startAddr, endAddr)
}

func validateTableParams(tableAddr, tableLen int) error {
	if tableAddr < 0 || tableAddr > maxAddressValue {
		return fmt.Errorf("%w: 0x%X", ErrInvalidTableAddr, tableAddr)
	}

	if tableLen <= 0 || tableLen > maxTableSize {
		return fmt.Errorf("%w: %d (max: %d)", ErrInvalidTableLen, tableLen, maxTableSize)
	}

	endAddress := int64(tableAddr) + int64(tableLen)
	if endAddress > maxAddressValue {
		return fmt.Errorf("%w: end address 0x%X exceeds limit", ErrAddressOverflow, endAddress)
	}

	return nil
}

func readSMBIOS(ctx context.Context) (EntryPoint, []*Table, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), readTimeout)
		defer cancel()
	}

	reader, cleanup, err := selectReader()
	if err != nil {
		return nil, nil, err
	}
	defer cleanup()

	return readFromSource(ctx, reader)
}

func selectReader() (tableReader, func(), error) {
	if _, err := os.Stat(sysfsEntryPoint); err == nil {
		reader := &sysfsReader{}
		return reader, func() {}, nil
	}

	reader, err := NewDevMemReader()
	if err != nil {
		return nil, nil, fmt.Errorf("no available SMBIOS source: %w", err)
	}

	cleanup := func() {
		_ = reader.Close()
	}

	return reader, cleanup, nil
}

func readFromSource(ctx context.Context, reader tableReader) (EntryPoint, []*Table, error) {
	ep, err := reader.readEntryPoint(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("reading entry point: %w", err)
	}

	tableAddr, tableLen := ep.Table()
	tables, err := reader.readTables(ctx, tableAddr, tableLen)
	if err != nil {
		return nil, nil, fmt.Errorf("reading tables: %w", err)
	}

	return ep, tables, nil
}
