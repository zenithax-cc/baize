package pci

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/pkg/execute"
)

// Path constants for sysfs directories.
const (
	sysfsPci string = "/sys/bus/pci/devices"
	sysfsMod string = "/sys/module"
)

// Modalias file parsing constants
// Example format: pci:v00008086d00001533sv00008086sd00000000bc02sc00i00
const (
	modaliasMinLength = 54

	// Field positions in modalias content
	vendorIDStart    = 9
	vendorIDEnd      = 13
	deviceIDStart    = 18
	deviceIDEnd      = 22
	subVendorIDStart = 28
	subVendorIDEnd   = 32
	subDeviceIDStart = 38
	subDeviceIDEnd   = 42
	classIDStart     = 44
	classIDEnd       = 46
	subClassIDStart  = 48
	subClassIDEnd    = 50
	progIFStart      = 51
	progIFEnd        = 53
)

// Predefined errors
var (
	ErrNilPCI        = errors.New("nil PCI object")
	ErrEmptyAddr     = errors.New("empty PCI address")
	ErrInvalidAddr   = errors.New("invalid PCI address format")
	ErrModaliasShort = errors.New("modalias content too short")
)

// PCI address format validation regex
// Format: DDDD:BB:DD.F (domain:bus:device.function)
// NVMe devices: DDDDD:BB:DD.F (domain:bus:function)
var pciAddrRegex = regexp.MustCompile(`^[0-9a-fA-F]{4,5}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-7]$`)

// Driver name validation regex (alphanumeric, underscore, hyphen only)
var driverNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// PCI IDs database lazy initialization
var (
	pciidsInstance *PCIIDs
	pciidsOnce     sync.Once
	pciidsErr      error
)

// getPCIIDs returns the PCI IDs database instance with lazy initialization
func getPCIIDs() (*PCIIDs, error) {
	pciidsOnce.Do(func() {
		pciidsInstance, pciidsErr = NewPCIIDs()
	})
	return pciidsInstance, pciidsErr
}

// New creates a new PCIe object with the given address
func New(addr string) *PCI {
	return &PCI{
		PCIAddr: addr,
	}
}

// Validate checks if the PCI address format is valid
func (p *PCI) Validate() error {
	if p == nil {
		return ErrNilPCI
	}

	if p.PCIAddr == "" {
		return ErrEmptyAddr
	}

	if !pciAddrRegex.MatchString(p.PCIAddr) {
		return fmt.Errorf("%w: %s", ErrInvalidAddr, p.PCIAddr)
	}

	return nil
}

// Collect gathers all PCI device information
func (p *PCI) Collect() error {
	// Validate address format first
	if err := p.Validate(); err != nil {
		return err
	}

	devicePath := filepath.Join(sysfsPci, p.PCIAddr)
	if _, err := os.Stat(devicePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("device not found: %s", p.PCIAddr)
		}
		return fmt.Errorf("accessing device path: %w", err)
	}

	// Collect information from various sources
	// Non-fatal errors are collected but don't stop the process
	var collectionErrors []string

	// 1. Parse modalias (core info, failure is fatal)
	if err := p.parseModalias(devicePath); err != nil {
		return fmt.Errorf("parsing modalias: %w", err)
	}

	// 2. Parse driver info (non-fatal)
	if err := p.parseDriver(devicePath); err != nil {
		collectionErrors = append(collectionErrors, fmt.Sprintf("driver: %v", err))
	}

	// 3. Parse link info (non-fatal)
	if err := p.parseLink(devicePath); err != nil {
		collectionErrors = append(collectionErrors, fmt.Sprintf("link: %v", err))
	}

	// 4. Parse other info (non-fatal)
	if err := p.parseOther(devicePath); err != nil {
		collectionErrors = append(collectionErrors, fmt.Sprintf("other: %v", err))
	}

	// Non-fatal errors are silently ignored as missing info is normal
	_ = collectionErrors

	return nil
}

// parseModalias extracts device information from the modalias file
func (p *PCI) parseModalias(devicePath string) error {
	modaliasPath := filepath.Join(devicePath, "modalias")

	content, err := os.ReadFile(modaliasPath)
	if err != nil {
		return fmt.Errorf("read modalias file: %w", err)
	}

	if len(content) < modaliasMinLength {
		return fmt.Errorf("%w: expected at least %d bytes, got %d", ErrModaliasShort, modaliasMinLength, len(content))
	}

	// Extract fields directly using slice indices for deterministic behavior
	p.VendorID = string(content[vendorIDStart:vendorIDEnd])
	p.DeviceID = string(content[deviceIDStart:deviceIDEnd])
	p.SubVendorID = string(content[subVendorIDStart:subVendorIDEnd])
	p.SubDeviceID = string(content[subDeviceIDStart:subDeviceIDEnd])
	p.ClassID = string(content[classIDStart:classIDEnd])
	p.SubClassID = string(content[subClassIDStart:subClassIDEnd])
	p.ProgIfID = string(content[progIFStart:progIFEnd])

	// Build combined ID
	p.PCIID = p.VendorID + ":" + p.DeviceID + ":" + p.SubVendorID + ":" + p.SubDeviceID

	// Resolve device names from PCI IDs database
	// Name resolution failure should not block info collection
	_ = p.resolveNames()

	return nil
}

// resolveNames looks up device names from the PCI IDs database
func (p *PCI) resolveNames() error {
	ids, err := getPCIIDs()
	if err != nil {
		return fmt.Errorf("get pci ids: %w", err)
	}

	vendorDeviceID := p.VendorID + ":" + p.DeviceID
	subVendorDeviceID := p.SubVendorID + ":" + p.SubDeviceID

	p.Vendor = ids.GetVendorName(p.VendorID)
	p.Device = ids.GetDeviceName(vendorDeviceID)
	p.SubVendor = ids.GetVendorName(p.SubVendorID)
	p.SubDevice = ids.GetSubsystemName(vendorDeviceID, subVendorDeviceID)

	return nil
}

// parseDriver extracts driver information
// Fixed: version/srcversion are read from /sys/module/{driver}/ not device path
func (p *PCI) parseDriver(devicePath string) error {
	// Read driver symlink
	driverLink := filepath.Join(devicePath, "driver")
	linkTarget, err := os.Readlink(driverLink)
	if err != nil {
		if os.IsNotExist(err) {
			// Device has no bound driver, this is normal
			return nil
		}
		return fmt.Errorf("reading driver link: %w", err)
	}

	driverName := filepath.Base(linkTarget)
	if driverName == "" || driverName == "." {
		return nil
	}

	// Validate driver name format for security
	if !driverNameRegex.MatchString(driverName) {
		return fmt.Errorf("invalid driver name format: %s", driverName)
	}

	p.Driver.DriverName = driverName

	// Read version info from /sys/module/{driver}/
	// Fixed: original code incorrectly read from device path
	modulePath := filepath.Join(sysfsMod, driverName)

	// Read version (optional)
	if version, err := readFileContent(filepath.Join(modulePath, "version")); err == nil {
		p.Driver.DriverVer = version
	}

	// Read source version (optional)
	if srcVersion, err := readFileContent(filepath.Join(modulePath, "srcversion")); err == nil {
		p.Driver.SrcVer = srcVersion
	}

	// Get driver file path (optional, failure is not an error)
	p.Driver.FileName = p.getDriverFile(driverName)

	return nil
}

// getDriverFile retrieves the driver module file path using modinfo
func (p *PCI) getDriverFile(driverName string) string {
	res := execute.Command("modinfo", "-n", driverName)
	if res.Err != nil {
		// modinfo may fail for built-in drivers, handle silently
		return ""
	}

	return strings.TrimSpace(string(res.Stdout))
}

// parseLink extracts PCIe link information
func (p *PCI) parseLink(devicePath string) error {
	// Define files to read and their target fields
	linkFiles := []struct {
		name   string
		target *string
	}{
		{"max_link_speed", &p.Link.MaxSpeed},
		{"max_link_width", &p.Link.MaxWidth},
		{"current_link_speed", &p.Link.CurrSpeed},
		{"current_link_width", &p.Link.CurrWidth},
	}

	for _, f := range linkFiles {
		if content, err := readFileContent(filepath.Join(devicePath, f.name)); err == nil {
			*f.target = content
		}
		// File not found is silently skipped
	}

	return nil
}

// parseOther extracts other PCIe information
func (p *PCI) parseOther(devicePath string) error {
	otherFiles := []struct {
		name   string
		target *string
	}{
		{"numa_node", &p.Numa},
		{"revision", &p.Revision},
	}

	for _, f := range otherFiles {
		if content, err := readFileContent(filepath.Join(devicePath, f.name)); err == nil {
			*f.target = content
		}
	}

	return nil
}

// readFileContent reads a file and returns its trimmed content
func readFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}
