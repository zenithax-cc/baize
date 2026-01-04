package utils

import (
	"bufio"
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
