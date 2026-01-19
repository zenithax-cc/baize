package smbios

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
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

func parseType(t *Table, offset int, complete bool, sp any) (int, error) {
	var (
		ok bool
		sv reflect.Value
	)

	if sv, ok = sp.(reflect.Value); !ok {
		sv = reflect.Indirect(reflect.ValueOf(sp))
	}

	svtn := sv.Type().Name()

	i := 0
	for ; i < sv.NumField() && offset < len(t.FormattedArea); i++ {
		f := sv.Type().Field(i)
		fv := sv.Field(i)
		ft := fv.Type()
		tags := f.Tag.Get(tagKey)
		ignore := false

		for tag := range strings.SplitSeq(tags, tagSeparator) {
			parts := strings.Split(tag, tagValueSeparator)
			switch parts[0] {
			case tagIgnore:
				ignore = true
			case tagSkip:
				numBytes, _ := strconv.Atoi(parts[1])
				offset += numBytes
			}
		}

		if ignore {
			continue
		}

		var fErr error
		switch ft.Kind() {
		case reflect.Uint8:
			v, err := t.GetByteAt(offset)
			fErr = err
			fv.SetUint(uint64(v))
			offset++
		case reflect.Uint16:
			v, err := t.GetWordAt(offset)
			fErr = err
			fv.SetUint(uint64(v))
			offset += 2
		case reflect.Uint32:
			v, err := t.GetDwordAt(offset)
			fErr = err
			fv.SetUint(uint64(v))
			offset += 4
		case reflect.Uint64:
			v, err := t.GetQwordAt(offset)
			fErr = err
			fv.SetUint(uint64(v))
			offset += 8
		case reflect.String:
			v, err := t.GetStringAt(offset)
			fErr = err
			fv.SetString(v)
			offset++
		default:
			if reflect.PointerTo(ft).Implements(fieldParserType) {
				offset, err := fv.Addr().Interface().(fieldParser).parseField(t, offset)
				if err != nil {
					return offset, fmt.Errorf("%s.%s: %w", svtn, f.Name, err)
				}
				break
			}

			if fv.Kind() == reflect.Struct {
				offset, err := parseType(t, offset, true, fv)
				if err != nil {
					return offset, err
				}
				break
			}

			return offset, fmt.Errorf("%s.%s: %v %s", svtn, f.Name, ErrUnsurpportedType, fv.Kind())
		}
		if fErr != nil {
			return offset, fmt.Errorf("parse %s.%s failed: %w", svtn, f.Name, fErr)
		}
	}

	if complete && i < sv.NumField() {
		return offset, fmt.Errorf("%w: %s incomplete,got %d of %d fields", io.ErrUnexpectedEOF, svtn, i, sv.NumField())
	}

	for ; i < sv.NumField(); i++ {
		f := sv.Type().Field(i)
		fv := sv.Field(i)
		ft := sv.Type()
		tags := f.Tag.Get(tagKey)

		ignore := false
		var defValue uint64
		for tag := range strings.SplitSeq(tags, tagSeparator) {
			parts := strings.Split(tag, tagValueSeparator)
			switch parts[0] {
			case tagIgnore:
				ignore = true
			case tagSkip:
				numBytes, _ := strconv.Atoi(parts[1])
				offset += numBytes
			case tagDefault:
				defValue, _ = strconv.ParseUint(parts[1], 0, 64)
			}
		}
		if ignore {
			continue
		}

		switch fv.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fv.SetUint(defValue)
			offset += int(ft.Size())
		case reflect.Struct:
			off, err := parseType(t, offset, false, fv)
			if err != nil {
				return off, err
			}
		}
	}

	return offset, nil
}
