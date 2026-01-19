package smbios

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/zenithax-cc/baize/pkg/utils"
)

var (
	ErrTableNotFound  = errors.New("table type not found")
	ErrParserNotFound = errors.New("no parser registered for table type")
	ErrSMBIOSFailed   = errors.New("failed to read SMBIOS data")
)

type parseFunc func(*Table) (any, error)

var parsers = map[TableType]parseFunc{
	BIOS:         func(t *Table) (any, error) { return parseType0BIOS(t) },
	System:       func(t *Table) (any, error) { return parseType1System(t) },
	BaseBoard:    func(t *Table) (any, error) { return parseType2BaseBoard(t) },
	Chassis:      func(t *Table) (any, error) { return parseType3Chassis(t) },
	Processor:    func(t *Table) (any, error) { return parseType4Processor(t) },
	MemoryDevice: func(t *Table) (any, error) { return parseType17MemoryDevice(t) },
}

type Decoder struct {
	EntryPoint EntryPoint
	tables     map[TableType][]*Table
	parsers    map[TableType]parseFunc
	cache      map[TableType][]any
	mu         sync.RWMutex
}

var decodeOnce sync.Once

func New(ctx context.Context) (*Decoder, error) {
	var d *Decoder
	var err error
	decodeOnce.Do(func() { d, err = getDecoder(ctx) })
	return d, err
}

func getDecoder(ctx context.Context) (*Decoder, error) {
	d := &Decoder{
		tables:  make(map[TableType][]*Table, len(parsers)),
		parsers: maps.Clone(parsers),
		cache:   make(map[TableType][]any, len(parsers)),
	}

	ep, tables, err := readSMBIOS(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSMBIOSFailed, err)
	}

	d.EntryPoint = ep
	for _, t := range tables {
		if _, ok := parsers[TableType(t.Type)]; ok {
			d.tables[TableType(t.Type)] = append(d.tables[TableType(t.Type)], t)
		}
	}

	return d, nil
}

func (d *Decoder) getParserData(t TableType) ([]any, error) {
	d.mu.RLock()
	cache, exists := d.cache[t]
	d.mu.RUnlock()

	if exists {
		return cache, nil
	}

	table, exists := d.tables[t]
	if !exists {
		return nil, errors.New("not supported table type")
	}

	parser, exists := d.parsers[t]
	if !exists {
		return nil, ErrParserNotFound
	}

	res := make([]any, 0, len(table))
	var errs []error
	for _, tb := range table {
		t, err := parser(tb)
		if err != nil {
			errs = append(errs, err)
		}
		if t != nil {
			res = append(res, t)
		}
	}

	return res, utils.CombineErrors(errs)
}

func GetTypeData[T any](d *Decoder, t TableType) ([]T, error) {
	data, err := d.getParserData(t)
	if err != nil {
		return nil, err
	}

	res := make([]T, 0, len(data))
	for _, d := range data {
		if item, ok := d.(T); ok {
			res = append(res, item)
		}
	}

	return res, nil
}

const (
	_         = iota
	KB uint64 = 1 << (iota * 10)
	MB
	GB
	TB
)

var sizeFormat = []struct {
	unit   uint64
	suffix string
}{
	{TB, "TB"},
	{GB, "GB"},
	{MB, "MB"},
	{KB, "KB"},
}

func kgmt(v uint64) string {
	for _, f := range sizeFormat {
		if v >= f.unit && v%(f.unit) == 0 {
			return fmt.Sprintf("%d %s", v/f.unit, f.suffix)
		}
	}

	return fmt.Sprintf("%d B", v)
}
