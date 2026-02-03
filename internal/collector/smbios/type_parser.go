package smbios

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

type fieldParser interface {
	parseField(t *Table, offset int) (int, error)
}

var (
	fieldTagKey             = "smbios"
	fieldParseInterfaceType = reflect.TypeOf((*fieldParser)(nil)).Elem()
)

func parseType(t *Table, offset int, complete bool, sp any) (int, error) {
	var (
		err error
		ok  bool
		sv  reflect.Value
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
		tags := f.Tag.Get(fieldTagKey)
		ignore := false
		for _, tag := range strings.Split(tags, ",") {
			tp := strings.Split(tag, "=")
			switch tp[0] {
			case "-":
				ignore = true
			case "skip":
				numBytes, _ := strconv.Atoi(tp[1])
				offset += numBytes
			}
		}
		if ignore {
			continue
		}
		var gErr error
		switch ft.Kind() {
		case reflect.Uint8:
			v, err := t.GetByteAt(offset)
			gErr = err
			fv.SetUint(uint64(v))
			offset++
		case reflect.Uint16:
			v, err := t.GetWordAt(offset)
			gErr = err
			fv.SetUint(uint64(v))
			offset += 2
		case reflect.Uint32:
			v, err := t.GetDwordAt(offset)
			gErr = err
			fv.SetUint(uint64(v))
			offset += 4
		case reflect.Uint64:
			v, err := t.GetQwordAt(offset)
			gErr = err
			fv.SetUint(uint64(v))
			offset += 8
		case reflect.String:
			v, err := t.GetStringAt(offset)
			gErr = err
			fv.SetString(v)
			offset++
		default:
			if reflect.PointerTo(ft).Implements(fieldParseInterfaceType) {
				offset, err = fv.Addr().Interface().(fieldParser).parseField(t, offset)
				if err != nil {
					return offset, fmt.Errorf("%s.%s: %w", svtn, f.Name, err)
				}
				break
			}
			if fv.Kind() == reflect.Struct {
				offset, err = parseType(t, offset, true /* complete */, fv)
				if err != nil {
					return offset, err
				}
				break
			}
			return offset, fmt.Errorf("%s.%s: unsupported type %s", svtn, f.Name, fv.Kind())
		}
		if gErr != nil {
			return offset, fmt.Errorf("failed to parse %s.%s: %w", svtn, f.Name, gErr)
		}
	}
	if complete && i < sv.NumField() {
		return offset, fmt.Errorf("%w: %s incomplete, got %d of %d fields", io.ErrUnexpectedEOF, svtn, i, sv.NumField())
	}

	// Fill in defaults
	for ; i < sv.NumField(); i++ {
		f := sv.Type().Field(i)
		fv := sv.Field(i)
		ft := fv.Type()
		tags := f.Tag.Get(fieldTagKey)
		// fmt.Printf("XX %02Xh f %s t %s k %s %s\n", off, f.Name, f.Type.Name(), fv.Kind(), tags)
		// Check tags first
		ignore := false
		var defValue uint64
		for _, tag := range strings.Split(tags, ",") {
			tp := strings.Split(tag, "=")
			switch tp[0] {
			case "-":
				ignore = true
			case "skip":
				numBytes, _ := strconv.Atoi(tp[1])
				offset += numBytes
			case "default":
				defValue, _ = strconv.ParseUint(tp[1], 0, 64)
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
			off, err := parseType(t, offset, false /* complete */, fv)
			if err != nil {
				return off, err
			}
		}
	}

	return offset, nil
}
