package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ParseKeyValue parses a string into a map of key-value pairs.
func ParseKeyValue(text string, sep string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, sep, 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}

	}
	return result
}

func ParseKeyValueFromBytes(data []byte, sep string, res map[string]*string) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		key, value, ok := strings.Cut(line, sep)
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if ptr, exists := res[key]; exists && *ptr == "" {
			*ptr = value
		}
	}

	return scanner.Err()
}

// ParseLineKeyValue parses a line into a key-value pair.
func ParseLineKeyValue(line, sep string) (string, string, bool) {
	idx := strings.Index(line, sep)
	if idx <= 0 {
		return "", "", false
	}

	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}

// FileExists checks if the file exists and is a regular file
func FileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func PathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)

	return err == nil || os.IsExist(err)
}

func ReadLinesOffsetN(path string, offset, n int64) ([]string, error) {
	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: %d", offset)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", path, err)
	}
	defer file.Close()

	capcity := n
	if n <= 0 {
		capcity = 64
	}

	maxLineSize := 1 << 20 // 1MB
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	scanner.Buffer(buf, maxLineSize)

	// Skip lines until the offset
	for i := int64(0); i < offset; i++ {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("skip lines in %s: %w", path, err)
			}

			return []string{}, fmt.Errorf("file %s has less than %d lines", path, offset)
		}
	}

	res := make([]string, 0, capcity)
	for scanner.Scan() {
		res = append(res, scanner.Text())
		if n > 0 && int64(len(res)) >= n {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read lines in %s: %w", path, err)
	}

	return res, nil
}

func ReadOneLineFile(path string) (string, error) {
	lines, err := ReadLinesOffsetN(path, 0, 1)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(lines[0]), nil
}

func ReadLines(path string) ([]string, error) {
	return ReadLinesOffsetN(path, 0, -1)
}

func CombineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	validErrors := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			validErrors = append(validErrors, err)
		}
	}

	switch len(validErrors) {
	case 0:
		return nil
	case 1:
		return validErrors[0]
	default:
		return errors.Join(validErrors...)
	}
}

func HasPrefix(str string, taget []string) bool {
	if len(taget) == 0 {
		return false
	}

	for _, prefix := range taget {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}

	return false
}

func FillField(s string, t *string) {
	if s == "" || *t != "" {
		return
	}

	*t = s
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
	{KB, "KB"},
	{MB, "MB"},
	{GB, "GB"},
	{TB, "TB"},
}

func KGMT(v uint64) string {
	for _, f := range sizeFormat {
		if v >= f.unit && v%(f.unit) == 0 {
			return fmt.Sprintf("%d %s", v/f.unit, f.suffix)
		}
	}

	return fmt.Sprintf("%d B", v)
}
