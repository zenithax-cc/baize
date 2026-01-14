package smbios

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type fieldParser interface {
	parseField(t *Table, offset int) (int, error)
}

const (
	tagKey            = "smbios"
	tagSkip           = "skip"
	tagIgnore         = "ignore"
	tagDefault        = "default"
	tagSeparator      = ","
	tagValueSeparator = "="
)

var (
	ErrUnsurpportedType = errors.New("unsupported field type")
	ErrIncompleteData   = errors.New("incomplete data")

	fieldParserType = reflect.TypeOf((*fieldParser)(nil)).Elem()
)

type fieldTag struct {
	ignore       bool
	skip         int
	defaultValue uint64
	hasDefault   bool
}

func parseFieldTag(tagStr string) (fieldTag, error) {
	var ft fieldTag
	if tagStr == "" {
		return ft, nil
	}

	for _, t := range strings.Split(tagStr, tagSeparator) {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}

		if tag == tagIgnore {
			ft.ignore = true
			continue
		}

		parts := strings.SplitN(tag, tagValueSeparator, 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case tagSkip:
			numBytes, _ := strconv.Atoi(value)
			ft.skip = numBytes
		case tagDefault:
			val, _ := strconv.ParseUint(value, 10, 64)
			ft.defaultValue = val
			ft.hasDefault = true
		}
	}

	return ft, nil
}

type typeInfo struct {
	name   string
	fields []fieldInfo
}

type fieldInfo struct {
	name          string
	index         int
	kind          reflect.Kind
	size          int
	tag           fieldTag
	isStruct      bool
	isFieldParser bool
	structInfo    *typeInfo
}

var typeCache = struct {
	sync.RWMutex
	m map[reflect.Type]*typeInfo
}{
	m: make(map[reflect.Type]*typeInfo),
}

func getTypeInfo(t reflect.Type) (*typeInfo, error) {
	typeCache.RLock()
	info, ok := typeCache.m[t]
	typeCache.RUnlock()

	if ok {
		return info, nil
	}

	typeCache.Lock()
	defer typeCache.Unlock()

	if info, ok = typeCache.m[t]; ok {
		return info, nil
	}

	info, err := buildTypeInfo(t)
	if err != nil {
		return nil, err
	}

	typeCache.m[t] = info
	return info, nil
}

func buildTypeInfo(t reflect.Type) (*typeInfo, error) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", t.Kind())
	}

	info := &typeInfo{
		name:   t.Name(),
		fields: make([]fieldInfo, 0, t.NumField()),
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if !f.IsExported() {
			continue
		}

		ft, err := parseFieldTag(f.Tag.Get(tagKey))
		if err != nil {
			return nil, fmt.Errorf("field %s.%s: %w", t.Name(), f.Name, err)
		}

		fi := fieldInfo{
			name:  f.Name,
			index: i,
			kind:  f.Type.Kind(),
			tag:   ft,
		}

		switch fi.kind {
		case reflect.Uint8:
			fi.size = 1
		case reflect.Uint16:
			fi.size = 2
		case reflect.Uint32:
			fi.size = 4
		case reflect.Uint64:
			fi.size = 8
		case reflect.String:
			fi.size = 1
		case reflect.Struct:
			fi.isStruct = true
		default:
			if reflect.PointerTo(f.Type).Implements(fieldParserType) {
				fi.isFieldParser = true
			} else {
				return nil, fmt.Errorf("field: %s.%s: %w: %s",
					t.Name(), f.Name, ErrUnsurpportedType, f.Type)
			}
		}

		info.fields = append(info.fields, fi)
	}

	return info, nil
}

type ParseOptions struct {
	RequireComplete bool
}

var DefaultParseOptions = ParseOptions{
	RequireComplete: true,
}

func parseStruct(t *Table, dest any, opts ...ParseOptions) error {
	opt := DefaultParseOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("dest must be a non-nil pointer")
	}

	_, err := parseValue(t, 0, opt.RequireComplete, rv.Elem())

	return err
}

func parseValue(t *Table, offset int, complete bool, v reflect.Value) (int, error) {
	info, err := getTypeInfo(v.Type())
	if err != nil {
		return offset, err
	}

	dataLen := len(t.FormattedArea)
	parsedCount := 0

	for _, f := range info.fields {
		if f.tag.ignore {
			continue
		}

		offset += f.tag.skip
		if offset >= dataLen {
			break
		}

		fv := v.Field(f.index)

		var parseErr error
		offset, parseErr = parseFieldOffset(t, offset, f, fv)
		if parseErr != nil {
			return offset, fmt.Errorf("field %s.%s: %w", info.name, f.name, parseErr)
		}

		parsedCount++
	}

	requiredFields := countRequiredFields(info.fields)
	if complete && parsedCount < requiredFields {
		return offset, fmt.Errorf("%w: %s got %d of %d fields", ErrIncompleteData,
			info.name, parsedCount, requiredFields)
	}

	for i := parsedCount; i < len(info.fields); i++ {
		f := info.fields[i]
		if f.tag.ignore {
			continue
		}

		fv := v.Field(f.index)
		offset = setDefaultValue(fv, f, offset)
	}

	return offset, nil
}

func countRequiredFields(fields []fieldInfo) int {
	count := 0
	for _, f := range fields {
		if !f.tag.ignore {
			count++
		}
	}

	return count
}

func setDefaultValue(fv reflect.Value, f fieldInfo, offset int) int {
	offset += f.tag.skip
	switch f.kind {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if f.tag.hasDefault {
			fv.SetUint(f.tag.defaultValue)
		}
		return offset + f.size
	case reflect.Struct:
		info, err := getTypeInfo(fv.Type())
		if err != nil {
			return offset
		}
		for _, subFi := range info.fields {
			if subFi.tag.ignore {
				continue
			}

			offset = setDefaultValue(fv.Field(subFi.index), subFi, offset)
		}
		return offset
	default:
		return offset + f.size
	}
}

func parseFieldOffset(t *Table, offset int, f fieldInfo, fv reflect.Value) (int, error) {
	switch f.kind {
	case reflect.Uint8:
		v, err := t.GetByteAt(offset)
		if err != nil {
			return offset, err
		}
		fv.SetUint(uint64(v))
		return offset + 1, nil
	case reflect.Uint16:
		v, err := t.GetWordAt(offset)
		if err != nil {
			return offset, err
		}
		fv.SetUint(uint64(v))
		return offset + 2, nil
	case reflect.Uint32:
		v, err := t.GetDwordAt(offset)
		if err != nil {
			return offset, err
		}
		fv.SetUint(uint64(v))
		return offset + 4, nil
	case reflect.Uint64:
		v, err := t.GetQwordAt(offset)
		if err != nil {
			return offset, err
		}
		fv.SetUint(uint64(v))
		return offset + 8, nil
	case reflect.String:
		v, err := t.GetStringAt(offset)
		if err != nil {
			return offset, err
		}
		fv.SetString(v)
		return offset + 1, nil
	case reflect.Struct:
		return parseValue(t, offset, true /* complete */, fv)
	default:
		if f.isFieldParser {
			return fv.Addr().Interface().(fieldParser).parseField(t, offset)
		}
		return offset, fmt.Errorf("%w: %s", ErrUnsurpportedType, f.kind)
	}
}

func parseType(t *Table, offset int, complete bool, sp any) (int, error) {
	var v reflect.Value
	if rv, ok := sp.(reflect.Value); !ok {
		v = reflect.Indirect(reflect.ValueOf(sp))
	} else {
		v = rv
	}

	return parseValue(t, offset, complete, v)
}
