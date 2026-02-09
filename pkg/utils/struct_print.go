package utils

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
)

type StructPrinter struct {
	indent     int
	labelWidth int
}

func NewStructPrinter() *StructPrinter {
	return &StructPrinter{
		indent:     4,
		labelWidth: 28,
	}
}

func (sp *StructPrinter) formatValue(colorRule string, value interface{}) string {
	strValue := fmt.Sprintf("%v", value)
	if colorRule == "" {
		return strValue
	}

	color := sp.getColor(colorRule)

	return fmt.Sprintf("%s%s%s", color, strValue, ColorReset)
}

func (sp *StructPrinter) getColor(colorRule string) string {
	return ""
}

func (sp *StructPrinter) printField(indent int, label string, value any, colorRule string) {
	indentStr := strings.Repeat(" ", indent*sp.indent)
	formattedValue := sp.formatValue(colorRule, value)
	fmt.Printf("%s%-*s: %v\n", indentStr, sp.labelWidth-indent*sp.indent, label, formattedValue)
}

func (sp *StructPrinter) printHeader(indent int, label string) {
	indentStr := strings.Repeat(" ", indent*sp.indent)
	fmt.Printf("%s[%s]\n", indentStr, label)
}

func (sp *StructPrinter) printStructHeader(indent int, label string, value string) {
	indentStr := strings.Repeat(" ", indent*sp.indent)
	fmt.Printf("\n%s%-*s: %s\n", indentStr, sp.labelWidth*sp.indent, label, value)
}

func (sp *StructPrinter) Print(v any) {
	sp.printValue(reflect.ValueOf(v), 0, true)
}

func (sp *StructPrinter) printValue(v reflect.Value, indent int, isRoot bool) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()

	if isRoot {
		if nameTag := t.Field(0).Tag.Get("name"); t.NumField() > 0 {
			if t.Field(0).Type.Kind() == reflect.Slice {
				sp.printHeader(indent, nameTag)
			}
		}
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		name := field.Tag.Get("name")
		if name == "" {
			name = field.Name
		}

		colorRule := field.Tag.Get("color")

		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			for j := 0; j < value.Len(); j++ {
				elem := value.Index(j)
				if elem.Kind() == reflect.Struct {
					elemType := elem.Type()
					if elemType.NumField() > 0 {
						elemName := elemType.Field(0).Tag.Get("name")
						if elemName == "" {
							elemName = elemType.Field(0).Name
						}
						sp.printStructHeader(indent+1, elemName, fmt.Sprintf("%v", elem.Field(0).Interface()))
					}
					sp.printRemainingFields(elem, indent+2)
				}
			}
		case reflect.Struct:
			sp.printStructHeader(indent, name, "")
			sp.printValue(value, indent+1, false)
		default:
			sp.printField(indent, name, value.Interface(), colorRule)
		}
	}
}

func (sp *StructPrinter) printRemainingFields(v reflect.Value, indent int) {
	t := v.Type()
	for i := 1; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		name := field.Tag.Get("name")
		if name == "" {
			name = field.Name
		}

		colorRule := field.Tag.Get("color")

		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			for j := 0; j < value.Len(); j++ {
				elem := value.Index(j)
				if elem.Kind() == reflect.Struct {
					elemType := elem.Type()
					if elemType.NumField() > 0 {
						elemName := elemType.Field(0).Tag.Get("name")
						if elemName == "" {
							elemName = elemType.Field(0).Name
						}
						sp.printStructHeader(indent, elemName, fmt.Sprintf("%v", elem.Field(0).Interface()))
					}
				}
				sp.printRemainingFields(elem, indent+1)
			}
		case reflect.Struct:
			sp.printStructHeader(indent, name, "")
			sp.printRemainingFields(value, indent+1)
		default:
			sp.printField(indent, name, value.Interface(), colorRule)
		}
	}
}
