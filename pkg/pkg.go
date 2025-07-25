package pkg

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

func printJson(mode string, c any) {
	if c == nil {
		fmt.Printf("%s information is not collected yet\n", mode)
		return
	}
	js, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		fmt.Printf("Failed to marshal %s information to JSON: %v\n", mode, err)
		return
	}

	fmt.Println(string(js))
}

func printSeparator(key, value string, isColon bool, i int) string {
	separator := strings.Repeat("    ", i)
	labelWidth := max(10, 40-i*4)
	if isColon {
		return fmt.Sprintf("%s%-*s: %v\n", separator, labelWidth, key, value)
	}
	return fmt.Sprintf("%s%s%s\n", separator, key, value)
}

type parseConfig struct {
	fields     []string
	fieldMap   map[string]bool
	labelWidth int
	maxDepth   int
	maxItems   int
	separator  string
	indent     int
	colorRule  map[string]map[string]string
}

func selectFields(v any, fields []string, indent int, color map[string]map[string]string) *strings.Builder {
	var sb strings.Builder

	fieldMap := make(map[string]bool, len(fields))
	for _, field := range fields {
		fieldMap[field] = true
	}

	config := parseConfig{
		fields:     fields,
		fieldMap:   fieldMap,
		labelWidth: max(10, 40-indent*4),
		maxDepth:   10,
		maxItems:   50,
		separator:  strings.Repeat("    ", indent),
		indent:     indent,
		colorRule:  color,
	}

	parseStruct(v, &sb, &config)
	return &sb
}

func parseStruct(v any, sb *strings.Builder, config *parseConfig) {
	if config.maxDepth <= 0 {
		sb.WriteString(config.separator + "...(max depth reached)\n")
		return
	}

	if v == nil {
		sb.WriteString(config.separator + "<nil>\n")
		return
	}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			sb.WriteString(config.separator + "<nil>\n")
			return
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	nextConfig := parseConfig{
		fields:     config.fields,
		fieldMap:   config.fieldMap,
		labelWidth: max(10, config.labelWidth-4),
		maxDepth:   config.maxDepth - 1,
		maxItems:   config.maxItems,
		separator:  config.separator + "    ",
		indent:     config.indent + 1,
		colorRule:  config.colorRule,
	}

	for _, fieldName := range config.fields {
		fieldVal := val.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}

		fieldType, ok := typ.FieldByName(fieldName)
		if !ok || !fieldType.IsExported() {
			continue
		}

		if utils.IsEmpty(fieldVal.Interface()) {
			continue
		}
		switch fieldVal.Kind() {
		case reflect.Struct:
			parseStruct(fieldVal.Interface(), sb, &nextConfig)
		case reflect.Pointer:
			if !fieldVal.IsNil() {
				parseStruct(fieldVal.Interface(), sb, &nextConfig)
			}
		case reflect.Slice, reflect.Array:
			sliceLen := fieldVal.Len()

			if sliceLen == 0 {
				continue
			}

			itemsToShow := min(sliceLen, config.maxItems)

			for i := 0; i < itemsToShow; i++ {
				elem := fieldVal.Index(i)
				elemType := elem.Kind()
				sb.WriteString("\n")
				switch elemType {
				case reflect.Struct:
					parseStruct(elem.Interface(), sb, &nextConfig)
				case reflect.Pointer:
					if !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
						parseStruct(elem.Interface(), sb, &nextConfig)
					}
				default:
					writeColor(sb, &nextConfig, fieldName, elem.Interface())
				}
			}
		case reflect.Map:
			mapLen := fieldVal.Len()
			if mapLen == 0 {
				continue
			}
			writeColor(sb, config, fieldName, fieldVal.Interface())
		default:
			writeColor(sb, config, fieldName, fieldVal.Interface())
		}
	}
}

func getColorValue(fieldName, value string, colorRules map[string]map[string]string) string {
	if rule, exists := colorRules[fieldName]; exists {
		if color, exists := rule[value]; exists {
			return color + value + colorReset
		}

		if color, exists := rule["default"]; exists {
			return color + value + colorReset
		}
	}

	return value
}

func writeColor(sb *strings.Builder, config *parseConfig, fieldName string, value any) {
	valueStr := fmt.Sprintf("%v", value)
	colorValue := getColorValue(fieldName, valueStr, config.colorRule)
	fmt.Fprintf(sb, "%s%-*s: %s\n", config.separator, config.labelWidth, fieldName, colorValue)
}
