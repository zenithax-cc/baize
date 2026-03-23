package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// ANSI color codes for terminal output.
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// StructPrinter renders struct data to the terminal in a human-readable
// key-value format, driven by struct field tags.
type StructPrinter struct {
	indent     int
	labelWidth int
}

// SP is the global StructPrinter instance used by all modules.
var SP = NewStructPrinter()

// NewStructPrinter creates a StructPrinter with default formatting parameters.
func NewStructPrinter() *StructPrinter {
	return &StructPrinter{
		indent:     4,
		labelWidth: 28,
	}
}

// formatValue applies optional color rules to the display value.
func (sp *StructPrinter) formatValue(colorRule string, value interface{}) string {
	strValue := fmt.Sprintf("%v", value)
	if colorRule == "" {
		return strValue
	}

	color := sp.getColor(colorRule, strValue)
	if color == "" {
		return strValue
	}

	return fmt.Sprintf("%s%s%s", color, strValue, ColorReset)
}

// getColor returns the ANSI color code for a given rule and value.
func (sp *StructPrinter) getColor(colorRule, value string) string {
	switch colorRule {
	case "trueGreen":
		if value == "true" {
			return ColorGreen
		}
		return ColorRed
	case "DefaultGreen", "defaultGreen":
		if value != "" {
			return ColorGreen
		}
	case "powerGreen":
		if value == "Performance" {
			return ColorGreen
		}
		return ColorYellow
	case "Diagnose":
		switch {
		case value == "Healthy" || value == "OK":
			return ColorGreen
		case value == "Unhealthy" || value == "WARNING" || strings.HasPrefix(value, "Unhealthy"):
			return ColorRed
		default:
			return ColorYellow
		}
	}
	return ""
}

// printField prints a single labeled value line with proper indentation and alignment.
func (sp *StructPrinter) printField(indent int, label string, value any, colorRule string) {
	indentStr := strings.Repeat(" ", indent*sp.indent)
	formattedValue := sp.formatValue(colorRule, value)
	fmt.Printf("%s%-*s: %v\n", indentStr, sp.labelWidth-indent*sp.indent, label, formattedValue)
}

// printHeader prints a module section header with visual separators.
func (sp *StructPrinter) printHeader(indent int, label string) {
	indentStr := strings.Repeat(" ", indent*sp.indent)
	line := strings.Repeat("─", 50)
	fmt.Printf("\n%s%s%s%s\n", indentStr, ColorCyan, line, ColorReset)
	fmt.Printf("%s%s[%s]%s\n", indentStr, ColorBold, label, ColorReset)
	fmt.Printf("%s%s%s%s\n", indentStr, ColorCyan, line, ColorReset)
}

// printStructHeader prints a sub-section header for slice elements (e.g., each DIMM, each NIC).
func (sp *StructPrinter) printStructHeader(indent int, label string, value string) {
	indentStr := strings.Repeat(" ", indent*sp.indent)
	fmt.Printf("\n%s%s%-*s%s: %s\n", indentStr, ColorBold, sp.labelWidth-indent*sp.indent, label, ColorReset, value)
}

// Print is the main entry point for rendering a struct to the terminal.
func (sp *StructPrinter) Print(v any, outputType string) {
	sp.printValue(reflect.ValueOf(v), outputType, 0, true)
}

// printValue recursively traverses struct fields and prints them according
// to their output/name/color tags.
func (sp *StructPrinter) printValue(v reflect.Value, outputType string, indent int, isRoot bool) {
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

		name, colorRule, ok := parseFieldTag(field, outputType)
		if !ok {
			continue
		}

		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			for j := 0; j < value.Len(); j++ {
				elem := value.Index(j)
				if elem.Kind() == reflect.Ptr {
					if elem.IsNil() {
						continue
					}
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					elemType := elem.Type()
					if elemType.NumField() > 0 {
						elemName := elemType.Field(0).Tag.Get("name")
						if elemName == "" {
							continue
						}
						sp.printStructHeader(indent+1, elemName, fmt.Sprintf("%v", elem.Field(0).Interface()))
					}
					sp.printRemainingFields(elem, outputType, indent+2)
				}

				if elem.Kind() == reflect.String {
					sp.printField(indent+2, name, elem.Interface(), colorRule)
				}
			}
		case reflect.Struct:
			sp.printValue(value, outputType, indent+1, false)
		default:
			if IsEmpty(value) {
				continue
			}
			sp.printField(indent, name, value.Interface(), colorRule)
		}
	}
}

// printRemainingFields prints all fields of a struct except the first one
// (which is typically used as the header/identifier).
func (sp *StructPrinter) printRemainingFields(v reflect.Value, outputType string, indent int) {
	t := v.Type()
	for i := 1; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		name, colorRule, ok := parseFieldTag(field, outputType)
		if !ok {
			continue
		}

		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			for j := 0; j < value.Len(); j++ {
				elem := value.Index(j)
				if elem.Kind() == reflect.Ptr {
					if elem.IsNil() {
						continue
					}
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					elemType := elem.Type()
					if elemType.NumField() > 0 {
						elemName := elemType.Field(0).Tag.Get("name")
						if elemName == "" {
							continue
						}
						sp.printStructHeader(indent, elemName, fmt.Sprintf("%v", elem.Field(0).Interface()))
					}
					sp.printRemainingFields(elem, outputType, indent+1)
				}
			}
		case reflect.Struct:
			sp.printRemainingFields(value, outputType, indent+1)
		case reflect.Ptr:
			if !value.IsNil() {
				sp.printRemainingFields(value.Elem(), outputType, indent+1)
			}
		default:
			if IsEmpty(value) {
				continue
			}
			sp.printField(indent, name, value.Interface(), colorRule)
		}
	}
}

// parseFieldTag extracts display metadata from struct field tags.
// Returns the display name, color rule, and whether the field should be shown
// for the given outputType ("brief" or "detail").
func parseFieldTag(field reflect.StructField, outputType string) (string, string, bool) {
	name := field.Tag.Get("name")
	color := field.Tag.Get("color")
	output := field.Tag.Get("output")
	var ot string
	switch output {
	case "both":
		ot = outputType
	case "brief":
		ot = "brief"
	case "detail":
		ot = "detail"
	}

	if ot != outputType || name == "" {
		return "", "", false
	}

	return name, color, true
}
