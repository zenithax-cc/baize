package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	timeOut    = 120 * time.Second
	ErrTimeOut = "command timed out"
)

type RunShell struct{}
type RunSheller interface {
	Command(name string, args ...string) ([]byte, error)
	CommandContext(ctx context.Context, name string, args ...string) ([]byte, error)
}

var Run RunSheller = &RunShell{}

// Command 执行shell命令
func (r *RunShell) Command(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()
	return r.CommandContext(ctx, name, args...)
}

// CommandContext 执行shell命令,超时退出
func (r *RunShell) CommandContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command not found: %s", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		return buf.Bytes(), fmt.Errorf("start command error: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		err = ctx.Err()
	}
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			switch ctxErr {
			case context.DeadlineExceeded:
				return buf.Bytes(), fmt.Errorf("%s: %w", ErrTimeOut, ctx.Err())
			case context.Canceled:
				return buf.Bytes(), fmt.Errorf("command was canceled:%w", ctxErr)
			default:
				return buf.Bytes(), fmt.Errorf("command context error: %w", ctx.Err())
			}
		}
		return buf.Bytes(), fmt.Errorf("command error: %w,%v", err, args)

	}
	return buf.Bytes(), nil
}

var (
	unitIndexMap = map[string]int{"B": 0, "KB": 1, "MB": 2, "GB": 3, "TB": 4, "PB": 5, "EB": 6, "ZB": 7, "YB": 8}
	decimalUnit  = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// binaryUnit   = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

// ConvertUnit 转换单位
func ConvertUnit(f float64, u string, isBinary bool) (string, error) {
	if f < 0 {
		return "", fmt.Errorf("f must be greater than or equal to 0")
	}
	index, ok := unitIndexMap[strings.ToUpper(u)]
	if !ok {
		return "", fmt.Errorf("unit must be B, KB, MB, GB, TB, PB, EB, ZB, YB")
	}
	var unit []string
	base := 1000
	if isBinary {
		//		unit = binaryUnit
		base = 1024
	}
	unit = decimalUnit
	//	} else {
	//		unit = decimalUnit
	//	}
	for f >= float64(base) && index < len(unit)-1 {
		f /= float64(base)
		index++
	}
	return fmt.Sprintf("%.2f %s", f, unit[index]), nil
}

// ReadOneLineFile 读取仅一行内容的文件，删除/n回车符
func ReadOneLineFile(file string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func ReadLines(file string) ([]string, error) {
	return ReadLinesOffsetN(file, 0, -1)
}

func ReadLinesOffsetN(file string, offset uint, n int) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open file %s failed: %w", file, err)
	}
	defer f.Close()

	initialCapacity := n
	if n < 0 {
		initialCapacity = 64
	}
	res := make([]string, 0, initialCapacity)

	scanner := bufio.NewScanner(f)
	for i := uint(0); i < offset; i++ {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error while skipping lines in file %s: %w", file, err)
			}
			return nil, fmt.Errorf("file %s has less than %d lines", file, offset)
		}
	}

	linesRead := 0
	for scanner.Scan() {
		res = append(res, strings.TrimSpace(scanner.Text()))
		linesRead++
		if n >= 0 && linesRead >= n {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error while reading file %s: %w", file, err)
	}
	return res, nil
}

// SplitTrimSpace 按sep分割字符串，并删除分割字符串首尾空格
func SplitTrimSpace(str string, sep string) []string {
	return SplitNTrimSpace(str, sep, -1)
}

func SplitNTrimSpace(str string, sep string, n int) []string {
	if len(str) == 0 || len(sep) == 0 {
		return nil
	}
	splitN := n
	if n < 0 {
		splitN = -1
	}
	parts := strings.SplitN(str, sep, splitN)
	res := make([]string, 0, len(parts))
	for i := range parts {
		trimed := strings.TrimSpace(parts[i])
		if trimed != "" {
			res = append(res, trimed)
		}
	}
	return res
}

// Cut 查找字符串中第一个sep分隔符，返回分隔符前后的字符串
func Cut(str, sep string) (string, string, bool) {
	key, value, found := strings.Cut(str, sep)
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

// IsEmpty 判断结构体是否为空
func IsEmpty(v any) bool {
	visited := make(map[uintptr]bool)
	return isEmptyWithVisited(v, visited)
}

func isEmptyWithVisited(v any, visited map[uintptr]bool) bool {
	if v == nil {
		return true
	}

	switch v := v.(type) {
	case string:
		return v == ""
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0
	case bool:
		return !v
	case []any:
		return len(v) == 0
	case map[any]any:
		return len(v) == 0
	}

	value := reflect.ValueOf(v)
	for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return true
		}

		if value.Kind() == reflect.Ptr {
			ptr := value.Pointer()
			if visited[ptr] {
				return false
			}
			visited[ptr] = true
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return value.IsZero()
	}

	return isStructEmpty(value, visited)
}

func isStructEmpty(value reflect.Value, visited map[uintptr]bool) bool {
	numField := value.NumField()
	structType := value.Type()

	for i := 0; i < numField; i++ {
		field := value.Field(i)
		fieldType := structType.Field(i)

		if fieldType.Anonymous {
			if !isEmptyWithVisited(field.Interface(), visited) {
				return false
			}
			continue
		}

		if field.CanInterface() {
			if !isEmptyWithVisited(field.Interface(), visited) {
				return false
			}
		} else {
			if !field.IsZero() {
				return false
			}
		}
	}
	return true
}

// PathExists checks if the path exists
func PathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)

	return err == nil
}

// FileExists checks if the file exists and is a regular file
func FileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirExists checks if the path exists and is a directory
func DirExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ReadDir 读取目录
func ReadDir(path string) ([]os.DirEntry, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	dir, err := os.ReadDir(path)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not exists", path)
		}
		return nil, fmt.Errorf("read %s failed: %w", path, err)
	}
	return dir, nil
}

// Atoi converts a string to an int, returning -1 if conversion fails
func Atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return i
}

func CombineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	var sb strings.Builder

	for i, err := range errs {
		if err != nil {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(err.Error())
		}
	}

	if sb.Len() == 0 {
		return nil
	}

	return fmt.Errorf("%s", sb.String())
}
