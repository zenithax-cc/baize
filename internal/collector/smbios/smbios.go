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

	readTimeout = 10 * time.Second
)

type Reader interface {
	readTables(tableAddr, tableLen int) ([]*Table, error)
	readEntryPoint() (EntryPoint, error)
}

type sysfsReader struct{}

type devMemReader struct {
	file  *os.File
	mutex sync.Mutex
}

func smbiosReader(ctx context.Context) (EntryPoint, []*Table, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), readTimeout)
		defer cancel()
	}

	if _, err := os.Stat(sysfsEntryPoint); err == nil {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("smbios read timeout: %w", ctx.Err())
		default:
			reader := &sysfsReader{}
			return readFromSource(ctx, reader)
		}
	}

	reader, err := NewDevMemReader()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create devmem reader: %w", err)
	}
	defer reader.Close()

	return readFromSource(ctx, reader)
}

func readFromSource(ctx context.Context, reader Reader) (EntryPoint, []*Table, error) {
	done := make(chan struct{})
	var (
		ep     EntryPoint
		tables []*Table
		err    error
	)
	go func() {
		defer close(done)
		ep, err = reader.readEntryPoint()
		if err != nil {
			return
		}
		var tableAddr, tableLen int
		tableAddr, tableLen = ep.Table()
		tables, err = reader.readTables(tableAddr, tableLen)
	}()

	select {
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("smbios read timeout: %w", ctx.Err())
	case <-done:
		return ep, tables, err
	}
}

func (r *sysfsReader) readTables(tableAddr, tableLen int) ([]*Table, error) {
	file, err := os.Open(sysfsDMI)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", sysfsDMI, err)
	}
	defer file.Close()

	return parseTables(file)
}

func (r *sysfsReader) readEntryPoint() (EntryPoint, error) {
	file, err := os.Open(sysfsEntryPoint)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", sysfsEntryPoint, err)
	}
	defer file.Close()

	return parseEntryPoint(file)
}

func NewDevMemReader() (*devMemReader, error) {
	file, err := os.Open(devMem)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", devMem, err)
	}

	return &devMemReader{file: file}, nil
}

func (r *devMemReader) Close() error {
	if r.file != nil {
		r.mutex.Lock()
		defer r.mutex.Unlock()
		err := r.file.Close()
		r.file = nil
		return err
	}
	return nil
}

func (r *devMemReader) readTables(tableAddr, tableLen int) ([]*Table, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, err := r.file.Seek(int64(tableAddr), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to table address %s failed: %w", devMem, err)
	}

	tables := make([]byte, 0, tableLen)
	if _, err := io.ReadFull(r.file, tables); err != nil {
		return nil, fmt.Errorf("read tables %s failed: %w", devMem, err)
	}

	return parseTables(bytes.NewReader(tables))
}

func (r *devMemReader) readEntryPoint() (EntryPoint, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	epAddr, err := r.findEntryPointAddr()
	if err != nil {
		return nil, err
	}

	if _, err := r.file.Seek(int64(epAddr), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to entry point address %s failed: %w", devMem, err)
	}

	return parseEntryPoint(r.file)

}

func (r *devMemReader) findEntryPointAddr() (int, error) {
	if _, err := r.file.Seek(int64(startAddr), io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek to entry point address %s failed: %w", devMem, err)
	}

	b := make([]byte, 5)
	for i := startAddr; i < endAddr-5; i++ {

		if _, err := io.ReadFull(r.file, b); err != nil {
			return 0, err
		}

		if bytes.HasPrefix(b, []byte("_SM")) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no SMBIOS entry point found in /dev/mem")
}
