package utils

import (
	"bufio"
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

// FileExists checks if the file exists and is a regular file
func FileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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

	return strings.TrimSpace(lines[0]), err
}

func ReadLines(path string) ([]string, error) {
	return ReadLinesOffsetN(path, 0, -1)
}
