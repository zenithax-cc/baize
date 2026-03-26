package utils

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

// OutputMode 输出模式
type OutputMode int

const (
	OutputBoth   OutputMode = iota // 输出 both 和 detail
	OutputSimple                   // 只输出 both
)

// ANSI 颜色码
const (
	Reset      = "\033[0m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
	BoldCyan   = "\033[1;36m"
	BoldWhite  = "\033[1;37m"
)

// Printer console 输出器
type Printer struct {
	mode       OutputMode
	lineWidth  int
	labelWidth int
}

// NewPrinter 创建输出器
func NewPrinter() *Printer {
	return &Printer{
		lineWidth:  50,
		labelWidth: 24,
	}
}

var PrinterInstance = NewPrinter()

// SetLineWidth 设置分隔线宽度
func (p *Printer) SetLineWidth(w int) *Printer {
	p.lineWidth = w
	return p
}

// SetLabelWidth 设置标签对齐宽度
func (p *Printer) SetLabelWidth(w int) *Printer {
	p.labelWidth = w
	return p
}

// Print 输出 struct 到 console
func (p *Printer) Print(v interface{}, title string) {
	fmt.Println("jjj")
	p.printSection(v, title, 0)
}

// printSection 输出一个 section（带标题）
func (p *Printer) printSection(v interface{}, title string, depth int) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	// 收集需要输出的字段
	lines := p.collectFields(val, depth+1)
	if len(lines) == 0 {
		return
	}

	// 打印标题
	p.printTitle(title, depth)

	// 打印字段
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
}

// printTitle 打印标题栏
func (p *Printer) printTitle(title string, depth int) {
	indent := strings.Repeat("    ", depth)
	//sep := indent + strings.Repeat("─", p.lineWidth)
	//fmt.Println(sep)
	fmt.Printf("%s%s[%s]%s\n", indent, BoldCyan, strings.ToUpper(title), Reset)
	//fmt.Println(sep)
	fmt.Println()
}

// collectFields 递归收集 struct 中需要输出的字段行
func (p *Printer) collectFields(val reflect.Value, indentLevel int) []string {
	var lines []string
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// 跳过未导出字段
		if !field.IsExported() {
			continue
		}
		name := field.Tag.Get("name")
		output := field.Tag.Get("output")
		colorTag := field.Tag.Get("color")

		// 处理嵌套 struct（匿名嵌入）
		if field.Anonymous {
			embedded := fieldVal
			if embedded.Kind() == reflect.Ptr {
				if embedded.IsNil() {
					continue
				}
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				if name == "" {
					name = field.Name
				}
				subLines := p.collectFields(embedded, indentLevel+1)
				if len(subLines) > 0 {
					// 嵌入的 struct 作为子 section
					sectionLines := p.formatSubSection(name, subLines, indentLevel)
					lines = append(lines, sectionLines...)
				}
				continue
			}
		}

		// 处理 struct 类型字段（非匿名）
		actualVal := fieldVal
		actualKind := actualVal.Kind()
		if actualKind == reflect.Ptr {
			if actualVal.IsNil() {
				continue
			}
			actualVal = actualVal.Elem()
			actualKind = actualVal.Kind()
		}

		// 处理 slice 字段
		if actualKind == reflect.Slice {
			if name == "" {
				continue
			}
			sliceLines := p.collectSliceFields(actualVal, name, output, indentLevel)
			lines = append(lines, sliceLines...)
			continue
		}

		// 处理嵌套 struct 字段（非匿名）
		if actualKind == reflect.Struct {
			if name == "" {
				name = field.Name
			}
			subLines := p.collectFields(actualVal, indentLevel+1)
			if len(subLines) > 0 {
				sectionLines := p.formatSubSection(name, subLines, indentLevel)
				lines = append(lines, sectionLines...)
			}
			continue
		}

		// 普通字段：检查是否需要输出
		if !p.shouldOutput(output) {
			continue
		}
		if name == "" {
			continue
		}

		// 获取字符串值
		strVal := fmt.Sprintf("%v", actualVal.Interface())
		if strVal == "" {
			continue
		}

		// 格式化输出行
		line := p.formatLine(name, strVal, colorTag, indentLevel)
		lines = append(lines, line)
	}

	return lines
}

// collectSliceFields 收集 slice 类型字段
func (p *Printer) collectSliceFields(sliceVal reflect.Value, name, output string, indentLevel int) []string {
	var lines []string

	if !p.shouldOutput(output) {
		return lines
	}

	if sliceVal.Len() == 0 {
		return lines
	}

	// 检查 slice 元素类型
	elemType := sliceVal.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() == reflect.Struct {
		// slice of struct: 每个元素作为子块输出
		for j := 0; j < sliceVal.Len(); j++ {
			elem := sliceVal.Index(j)
			if elem.Kind() == reflect.Ptr {
				if elem.IsNil() {
					continue
				}
				elem = elem.Elem()
			}
			subName := fmt.Sprintf("%s #%d", name, j+1)

			// 尝试从 struct 中找到一个合适的标识名
			identifier := p.findIdentifier(elem)
			if identifier != "" {
				subName = fmt.Sprintf("%s [%s]", name, identifier)
			}

			subLines := p.collectFields(elem, indentLevel+2)
			if len(subLines) > 0 {
				sectionLines := p.formatSubSection(subName, subLines, indentLevel+1)
				lines = append(lines, sectionLines...)
			}
		}
	}

	return lines
}

// findIdentifier 尝试从 struct 中找到标识字段（如 Name, Model, Device 等）
func (p *Printer) findIdentifier(val reflect.Value) string {
	typ := val.Type()
	identifierFields := []string{"Name", "Model", "Device", "Locator", "DeviceLocator"}
	for _, idField := range identifierFields {
		for i := 0; i < typ.NumField(); i++ {
			if typ.Field(i).Name == idField {
				fv := val.Field(i)
				if fv.Kind() == reflect.String && fv.String() != "" {
					return fv.String()
				}
			}
		}
	}
	return ""
}

// shouldOutput 根据 output tag 和当前模式判断是否输出
func (p *Printer) shouldOutput(output string) bool {
	if output == "" {
		return false
	}
	switch output {
	case "both":
		return true
	case "detail":
		return p.mode == OutputBoth
	case "simple":
		return true
	default:
		return false
	}
}

// formatLine 格式化一行输出
func (p *Printer) formatLine(name, value, colorTag string, indentLevel int) string {
	indent := strings.Repeat("    ", indentLevel)
	// 右侧补空格对齐
	padding := p.labelWidth - utf8.RuneCountInString(name)
	if padding < 1 {
		padding = 1
	}
	label := name + strings.Repeat(" ", padding)

	colorCode := p.resolveColor(colorTag, value)
	resetCode := ""
	if colorCode != "" {
		resetCode = Reset
	}

	return fmt.Sprintf("%s%s: %s%s%s", indent, label, colorCode, value, resetCode)
}

// formatSubSection 格式化子 section
func (p *Printer) formatSubSection(name string, subLines []string, indentLevel int) []string {
	var lines []string
	indent := strings.Repeat("    ", indentLevel)

	// 子标题
	lines = append(lines, fmt.Sprintf("%s%s%s:%s", indent, BoldWhite, name, Reset))
	lines = append(lines, subLines...)
	lines = append(lines, "") // 空行分隔
	return lines
}

// resolveColor 根据 color tag 解析颜色
func (p *Printer) resolveColor(colorTag, value string) string {
	if colorTag == "" {
		return ""
	}

	switch colorTag {
	case "defaultGreen":
		return Green
	case "defaultRed":
		return Red
	case "defaultYellow":
		return Yellow
	case "defaultBlue":
		return Blue
	case "defaultCyan":
		return Cyan
	case "Diagnose":
		return p.diagnoseColor(value)
	case "red":
		return Red
	case "green":
		return Green
	case "yellow":
		return Yellow
	case "blue":
		return Blue
	case "cyan":
		return Cyan
	case "magenta":
		return Magenta
	case "boldRed":
		return BoldRed
	case "boldGreen":
		return BoldGreen
	case "boldYellow":
		return BoldYellow
	case "boldBlue":
		return BoldBlue
	case "boldCyan":
		return BoldCyan
	default:
		return ""
	}
}

// diagnoseColor 根据诊断值返回颜色
func (p *Printer) diagnoseColor(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "healthy"),
		strings.Contains(lower, "normal"),
		strings.Contains(lower, "ok"),
		strings.Contains(lower, "pass"),
		strings.Contains(lower, "good"):
		return Green
	case strings.Contains(lower, "warning"),
		strings.Contains(lower, "warn"),
		strings.Contains(lower, "degrad"):
		return Yellow
	case strings.Contains(lower, "critical"),
		strings.Contains(lower, "error"),
		strings.Contains(lower, "fail"),
		strings.Contains(lower, "abnormal"),
		strings.Contains(lower, "bad"):
		return BoldRed
	default:
		return Yellow
	}
}

// PrintAll 输出顶层 struct，自动展开嵌套 struct 字段为独立 section
func (p *Printer) PrintAll(v interface{}) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !field.IsExported() {
			continue
		}

		name := field.Tag.Get("name")
		if name == "" {
			name = field.Name
		}

		actual := fieldVal
		if actual.Kind() == reflect.Ptr {
			if actual.IsNil() {
				continue
			}
			actual = actual.Elem()
		}

		switch actual.Kind() {
		case reflect.Struct:
			p.printSection(actual.Interface(), name+" Info", 0)
		case reflect.Slice:
			if actual.Len() > 0 {
				p.printSliceSection(actual, name, 0)
			}
		default:
			// 普通字段直接输出
			output := field.Tag.Get("output")
			colorTag := field.Tag.Get("color")
			if p.shouldOutput(output) && name != "" {
				strVal := fmt.Sprintf("%v", actual.Interface())
				if strVal != "" {
					fmt.Println(p.formatLine(name, strVal, colorTag, 1))
				}
			}
		}
	}
}

// printSliceSection 输出 slice 类型的 section
func (p *Printer) printSliceSection(sliceVal reflect.Value, title string, depth int) {
	if sliceVal.Len() == 0 {
		return
	}

	indent := strings.Repeat("    ", depth)
	sep := indent + strings.Repeat("─", p.lineWidth)
	fmt.Println(sep)
	fmt.Printf("%s%s[%s]%s\n", indent, BoldCyan, strings.ToUpper(title), Reset)
	fmt.Println(sep)
	fmt.Println()

	for i := 0; i < sliceVal.Len(); i++ {
		elem := sliceVal.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Struct {
			identifier := p.findIdentifier(elem)
			subName := fmt.Sprintf("%s #%d", title, i+1)
			if identifier != "" {
				subName = fmt.Sprintf("%s [%s]", title, identifier)
			}

			subLines := p.collectFields(elem, depth+2)
			if len(subLines) > 0 {
				innerIndent := strings.Repeat("    ", depth+1)
				fmt.Printf("%s%s%s:%s\n", innerIndent, BoldWhite, subName, Reset)
				for _, line := range subLines {
					fmt.Println(line)
				}
				fmt.Println()
			}
		}
	}
}
