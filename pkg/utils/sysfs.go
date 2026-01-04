package utils

import (
	"os"
	"strconv"
	"strings"
)

// ReadSysfsFile reads a file from the sysfs and returns its contents as a string.
func ReadSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadSysfsInt reads a file from the sysfs and returns its contents as an integer.
func ReadSysfsInt(path string) (int, error) {
	data, err := ReadSysfsFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(data)
}

// ReadSysfsUint64 reads a file from the sysfs and returns its contents as an unsigned 64-bit integer.
func ReadSysfsUint64(path string) (uint64, error) {
	data, err := ReadSysfsFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(data, 10, 64)
}
