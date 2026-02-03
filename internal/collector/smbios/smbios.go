package smbios

import (
	"bytes"
	"context"
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
	startAddr       = 0xF0000
	endAddr         = 0x100000
	maxTableSize    = 1024 * 1024 // 1MB limit
	readTimeout     = 10 * time.Second
)

// 错误类型定义
type SMBIOSError struct {
	Op   string
	Path string
	Err  error
}

func (e *SMBIOSError) Error() string {
	return fmt.Sprintf("smbios %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *SMBIOSError) Unwrap() error {
	return e.Err
}

// 接口定义
type Reader interface {
	readTables(ctx context.Context, tableAddr, tableLen int) ([]*Table, error)
	readEntryPoint(ctx context.Context) (EntryPoint, error)
	Close() error
}

// sysfs reader implementation
type sysfsReader struct{}

func (r *sysfsReader) readTables(ctx context.Context, tableAddr, tableLen int) ([]*Table, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if tableLen <= 0 || tableLen > maxTableSize {
		return nil, fmt.Errorf("invalid table length: %d", tableLen)
	}

	file, err := os.Open(sysfsDMI)
	if err != nil {
		return nil, &SMBIOSError{Op: "open", Path: sysfsDMI, Err: err}
	}
	defer file.Close()

	return parseTables(file)
}

func (r *sysfsReader) readEntryPoint(ctx context.Context) (EntryPoint, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	file, err := os.Open(sysfsEntryPoint)
	if err != nil {
		return nil, &SMBIOSError{Op: "open", Path: sysfsEntryPoint, Err: err}
	}
	defer file.Close()

	return parseEntryPoint(file)
}

func (r *sysfsReader) Close() error {
	return nil // No resources to clean up
}

// devmem reader implementation
type devMemReader struct {
	file   *os.File
	mutex  sync.RWMutex
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
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.closed && r.file != nil {
		err := r.file.Close()
		r.file = nil
		r.closed = true
		return err
	}
	return nil
}

func (r *devMemReader) isClosed() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.closed
}

func (r *devMemReader) readTables(ctx context.Context, tableAddr, tableLen int) ([]*Table, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if tableAddr < 0 {
		return nil, fmt.Errorf("invalid table address: 0x%x", tableAddr)
	}
	if tableLen <= 0 || tableLen > maxTableSize {
		return nil, fmt.Errorf("invalid table length: %d", tableLen)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.closed || r.file == nil {
		return nil, fmt.Errorf("reader is closed")
	}

	if _, err := r.file.Seek(int64(tableAddr), io.SeekStart); err != nil {
		return nil, &SMBIOSError{Op: "seek", Path: devMem, Err: err}
	}

	data := make([]byte, tableLen)
	if _, err := io.ReadFull(r.file, data); err != nil {
		return nil, &SMBIOSError{Op: "read", Path: devMem, Err: err}
	}

	return parseTables(bytes.NewReader(data))
}

func (r *devMemReader) readEntryPoint(ctx context.Context) (EntryPoint, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.closed || r.file == nil {
		return nil, fmt.Errorf("reader is closed")
	}

	epAddr, err := r.findEntryPointAddr(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := r.file.Seek(int64(epAddr), io.SeekStart); err != nil {
		return nil, &SMBIOSError{Op: "seek", Path: devMem, Err: err}
	}

	return parseEntryPoint(r.file)
}

func (r *devMemReader) findEntryPointAddr(ctx context.Context) (int, error) {

	if _, err := r.file.Seek(int64(startAddr), io.SeekStart); err != nil {
		return 0, &SMBIOSError{Op: "seek", Path: devMem, Err: err}
	}

	const paragraph = 16
	b := make([]byte, paragraph)

	for addr := startAddr; addr < endAddr; addr += paragraph {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if _, err := io.ReadFull(r.file, b); err != nil {
			return 0, &SMBIOSError{Op: "read", Path: devMem, Err: err}
		}

		if bytes.HasPrefix(b, []byte("_SM")) {
			return addr, nil
		}
	}

	return 0, fmt.Errorf("SMBIOS entry point not found in memory range 0x%x-0x%x", startAddr, endAddr)
}

func smbiosReader(ctx context.Context) (EntryPoint, []*Table, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), readTimeout)
		defer cancel()
	}

	if _, err := os.Stat(sysfsEntryPoint); err == nil {
		reader := &sysfsReader{}
		return readFromSource(ctx, reader)
	}

	reader, err := NewDevMemReader()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create devmem reader: %w", err)
	}
	defer reader.Close()

	return readFromSource(ctx, reader)
}

func readFromSource(ctx context.Context, reader Reader) (EntryPoint, []*Table, error) {
	ep, err := reader.readEntryPoint(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read entry point: %w", err)
	}

	tableAddr, tableLen := ep.Table()
	tables, err := reader.readTables(ctx, tableAddr, tableLen)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read tables: %w", err)
	}

	return ep, tables, nil
}
