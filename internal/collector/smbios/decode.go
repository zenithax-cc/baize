package smbios

import (
	"context"
	"fmt"
	"sync"
)

type parserFunc func(*Table) (any, error)

type Decoder struct {
	EntryPoint EntryPoint
	tables     map[TableType][]*Table
	parsers    map[TableType]parserFunc
	parsedData map[TableType][]any
	mutex      sync.RWMutex
}

func NewDecoder() *Decoder {
	return &Decoder{
		tables:     make(map[TableType][]*Table),
		parsers:    make(map[TableType]parserFunc),
		parsedData: make(map[TableType][]any),
	}
}

func (d *Decoder) RegisterParser(t TableType, p parserFunc) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.parsers[t] = p
}

func (d *Decoder) RegisterDefaultParsers() {
	// bios解析函数注册
	d.RegisterParser(BIOS, func(t *Table) (interface{}, error) {
		return parseType0BIOS(t)
	})
	// system解析函数注册
	d.RegisterParser(System, func(t *Table) (interface{}, error) {
		return parseType1System(t)
	})
	// baseboard解析函数注册
	d.RegisterParser(BaseBoard, func(t *Table) (interface{}, error) {
		return parseType2BaseBoard(t)
	})

	// chassis解析函数注册
	d.RegisterParser(Chassis, func(t *Table) (interface{}, error) {
		return parseType3Chassis(t)
	})

	// processor解析函数注册
	d.RegisterParser(Processor, func(t *Table) (interface{}, error) {
		return parseType4Processor(t)
	})

	// memorydevice解析函数注册
	d.RegisterParser(MemoryDevice, func(t *Table) (interface{}, error) {
		return parseType17MemoryDevice(t)
	})
}

func (d *Decoder) initialize() error {
	// 获取SMBIOS数据
	ep, tables, err := smbiosReader(context.TODO())
	if err != nil {
		return fmt.Errorf("get SMBIOS data failed: %w", err)
	}

	d.EntryPoint = ep
	for _, t := range tables {
		type_ := TableType(t.Type)
		d.tables[type_] = append(d.tables[type_], t)
	}

	d.RegisterDefaultParsers()

	return nil
}

func New() (*Decoder, error) {
	d := NewDecoder()
	if err := d.initialize(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Decoder) getParsedData(t TableType) ([]any, error) {
	d.mutex.RLock()
	if data, exists := d.parsedData[t]; exists {
		d.mutex.RUnlock()
		return data, nil
	}
	d.mutex.RUnlock()

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if data, exists := d.parsedData[t]; exists {
		return data, nil
	}

	tables, exists := d.tables[t]
	if !exists || len(tables) == 0 {
		return nil, fmt.Errorf("table type %d not found", t)
	}

	parser, exists := d.parsers[t]
	if !exists {
		return nil, fmt.Errorf("no parser registered for table type %d", t)
	}
	res := make([]any, 0, len(tables))
	var parseErr []error

	for _, table := range tables {
		r, err := parser(table)
		if err != nil {
			parseErr = append(parseErr, err)
			continue
		}
		if r != nil {
			res = append(res, r)
		}
	}

	d.parsedData[t] = res

	if len(parseErr) > 0 {
		for _, err := range parseErr {
			fmt.Printf("Warning: %s\n", err)
		}
	}

	return res, nil
}

func GetTypeData[T any](d *Decoder, t TableType) ([]T, error) {

	data, err := d.getParsedData(t)

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

var SMBIOS *Decoder

func init() {
	var err error
	SMBIOS, err = New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize SMBIOS decoder: %v", err))
	}
}

func kgmt(v uint64) string {
	switch {
	case v >= 1024*1024*1024*1024 && v%(1024*1024*1024*1024) == 0:
		return fmt.Sprintf("%d TB", v/(1024*1024*1024*1024))
	case v >= 1024*1024*1024 && v%(1024*1024*1024) == 0:
		return fmt.Sprintf("%d GB", v/(1024*1024*1024))
	case v >= 1024*1024 && v%(1024*1024) == 0:
		return fmt.Sprintf("%d MB", v/(1024*1024))
	case v >= 1024 && v%1024 == 0:
		return fmt.Sprintf("%d KB", v/1024)
	default:
		return fmt.Sprintf("%d B", v)
	}
}
