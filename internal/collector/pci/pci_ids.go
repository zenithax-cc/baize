package pci

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zenithax-cc/baize/pkg/utils"
)

// Type Definitions

// PCIIDs represents the PCI IDs database containing classes, vendors, and devices.
type PCIIDs struct {
	Class  map[string]*Class  `json:"class,omitempty"`
	Vendor map[string]*Vendor `json:"vendor,omitempty"`
	Device map[string]*Device `json:"device,omitempty"`
}

// Class represents a PCI device class.
type Class struct {
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	SubClass []*SubClass `json:"sub_class,omitempty"`
}

// SubClass represents a PCI device subclass.
type SubClass struct {
	ID               string              `json:"id,omitempty"`
	Name             string              `json:"name,omitempty"`
	ProgramInterface []*ProgramInterface `json:"program_interface,omitempty"`
}

// ProgramInterface represents a PCI programming interface.
type ProgramInterface struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Vendor represents a PCI vendor.
type Vendor struct {
	ID      string    `json:"id,omitempty"`
	Name    string    `json:"name,omitempty"`
	Devices []*Device `json:"devices,omitempty"`
}

// Device represents a PCI device.
type Device struct {
	ID        string       `json:"id,omitempty"`
	Name      string       `json:"name,omitempty"`
	VendorID  string       `json:"vendor_id,omitempty"`
	Subsystem []*Subsystem `json:"subsystem,omitempty"`

	// subsystemIdx provides O(1) lookup for subsystems by ID.
	// Not exported to JSON as it's an internal optimization.
	subsystemIdx map[string]*Subsystem `json:"-"`
}

// Subsystem represents a PCI subsystem (sub-vendor and sub-device combination).
type Subsystem struct {
	ID          string `json:"id,omitempty"`
	SubVendorID string `json:"sub_vendor_id,omitempty"`
	SubDeviceID string `json:"sub_device_id,omitempty"`
	Name        string `json:"name,omitempty"`
}

// gzip Reader Wrapper - Fixes resource leak

// gzipReadCloser wraps gzip.Reader to ensure both the gzip reader
// and underlying file are properly closed.
// This fixes the critical bug where defer uzip.Close() was called
// inside the function, closing the reader before it could be used.
type gzipReadCloser struct {
	*gzip.Reader
	underlying io.Closer
}

// Close closes both the gzip reader and the underlying reader.
func (g *gzipReadCloser) Close() error {
	gzipErr := g.Reader.Close()
	underlyingErr := g.underlying.Close()
	// Return the first error encountered
	if gzipErr != nil {
		return gzipErr
	}
	return underlyingErr
}

// Configuration and Constants

var (
	// pciidsPath contains standard locations for pci.ids file on Linux systems.
	pciidsPath = []string{
		"/usr/share/misc/pci.ids",
		"/usr/share/misc/pci.ids.gz",
		"/usr/share/hwdata/pci.ids",
		"/usr/share/hwdata/pci.ids.gz",
	}

	// pciidsURL is the upstream source for pci.ids database.
	pciidsURL = "https://raw.githubusercontent.com/pciutils/pciids/master/pci.ids"
)

// Download constraints to prevent resource exhaustion attacks.
const (
	// downloadTimeout prevents indefinite hanging on slow/unresponsive servers.
	downloadTimeout = 30 * time.Second

	// maxDownloadSize prevents memory exhaustion from malicious responses.
	// The pci.ids file is typically ~1.5MB, so 10MB provides ample headroom.
	maxDownloadSize = 10 * 1024 * 1024
)

// Estimated capacities based on actual pci.ids file statistics.
// These values minimize map resizing during parsing.
const (
	estimatedClasses   = 32    // ~22 classes in pci.ids
	estimatedVendors   = 3000  // ~2800 vendors
	estimatedDevices   = 40000 // ~35000 devices
	estimatedSubsystem = 8     // Average subsystems per device
)

// Singleton Instance Management

var (
	instance   *PCIIDs
	initErr    error
	instanceMu sync.Once
)

// Predefined errors for better error handling.
var (
	ErrPCIIDsNotFound   = errors.New("pci.ids file not found in standard paths")
	ErrDownloadFailed   = errors.New("failed to download pci.ids from remote")
	ErrNilReader        = errors.New("nil reader provided to parser")
	ErrNotInitialized   = errors.New("PCIIDs database not initialized")
	ErrInvalidSubsystem = errors.New("invalid subsystem format")
)

// NewPCIIDs returns the singleton PCIIDs instance.
// Thread-safe initialization using sync.Once.
// Returns error if initialization fails.
func NewPCIIDs() (*PCIIDs, error) {
	instanceMu.Do(func() {
		file, err := getPCIIDsFile()
		if err != nil {
			initErr = fmt.Errorf("locating pci.ids: %w", err)
			return
		}

		content, err := getPCIIDsContent(file)
		if err != nil {
			initErr = fmt.Errorf("reading pci.ids: %w", err)
			return
		}

		instance, initErr = parsePCIIDsContent(content)
		if initErr != nil {
			initErr = fmt.Errorf("parsing pci.ids: %w", initErr)
		}
	})

	if instance == nil {
		return nil, initErr
	}
	return instance, nil
}

// MustNewPCIIDs returns the singleton PCIIDs instance or panics on error.
// Useful for initialization in main() or init() where errors are fatal.
func MustNewPCIIDs() *PCIIDs {
	pci, err := NewPCIIDs()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize PCIIDs: %v", err))
	}
	return pci
}

// File Location and Download

// getPCIIDsFile finds the pci.ids file in standard paths or downloads it.
// Returns the path to the file or an error if not found/downloadable.
func getPCIIDsFile() (string, error) {
	// Try standard system paths first - most efficient option
	for _, file := range pciidsPath {
		if utils.FileExists(file) {
			return file, nil
		}
	}

	// Fall back to downloading from upstream
	return downloadPCIIDs()
}

// downloadPCIIDs fetches pci.ids from the upstream URL.
// Implements security best practices:
// - Request timeout to prevent hanging
// - Response size limit to prevent memory exhaustion
// - Secure temporary file creation
func downloadPCIIDs() (string, error) {
	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pciidsURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{
		// No additional timeout needed - context handles it
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	// Verify successful response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: HTTP status %d", ErrDownloadFailed, resp.StatusCode)
	}

	// Limit response size to prevent memory exhaustion attacks
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	// Create secure temporary file
	// os.CreateTemp creates file with 0600 permissions, preventing symlink attacks
	tmpFile, err := os.CreateTemp("", "pci.ids.*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write and close in single operation for atomicity
	if _, err := tmpFile.Write(respBody); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath) // Cleanup on error
		return "", fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return tmpPath, nil
}

// File Content Reading

// getPCIIDsContent opens the pci.ids file and returns a reader.
// Handles both plain text and gzip compressed files.
// The caller is responsible for closing the returned reader.
func getPCIIDsContent(file string) (io.ReadCloser, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", file, err)
	}

	// Handle gzip compressed files
	if strings.HasSuffix(file, ".gz") {
		gzReader, err := gzip.NewReader(f)
		if err != nil {
			f.Close() // Critical: close file handle on error
			return nil, fmt.Errorf("creating gzip reader for %s: %w", file, err)
		}
		// Return wrapper that closes both gzip reader and underlying file
		return &gzipReadCloser{
			Reader:     gzReader,
			underlying: f,
		}, nil
	}

	return f, nil
}

// Parsing Utilities

// countLeadingTabs counts the number of leading tab characters in a string.
// Optimized to use byte comparison instead of rune iteration.
func countLeadingTabs(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

// extractIDAndName splits a line into ID and name components.
// The ID is expected to be at the start with fixed length.
// Handles edge cases where name may be missing.
func extractIDAndName(s string, idLen int) (id, name string) {
	s = strings.TrimSpace(s)
	sLen := len(s)

	// Handle short strings
	if sLen < idLen {
		return s, ""
	}

	id = s[:idLen]

	// Handle case where there's no name after ID
	if sLen <= idLen {
		return id, ""
	}

	// Skip any whitespace between ID and name
	name = strings.TrimSpace(s[idLen:])
	return id, name
}

// buildDeviceID creates a composite device ID from vendor and device IDs.
// Using string concatenation is faster than fmt.Sprintf for simple cases.
func buildDeviceID(vendorID, deviceID string) string {
	// Pre-allocate exact size: vendorID + ":" + deviceID
	var b strings.Builder
	b.Grow(len(vendorID) + 1 + len(deviceID))
	b.WriteString(vendorID)
	b.WriteByte(':')
	b.WriteString(deviceID)
	return b.String()
}

// buildSubsystemID creates a composite subsystem ID.
func buildSubsystemID(subVendorID, subDeviceID string) string {
	var b strings.Builder
	b.Grow(len(subVendorID) + 1 + len(subDeviceID))
	b.WriteString(subVendorID)
	b.WriteByte(':')
	b.WriteString(subDeviceID)
	return b.String()
}

// Parser State Machine

// Parser states - only define what's actually used
const (
	stateNone = iota
	stateClass
	stateVendor
)

// parsePCIIDsContent parses the pci.ids file format.
// The file format is:
//
//	C class   class_name
//	    subclass  subclass_name       <- single tab
//	        prog-if  prog-if_name     <- two tabs
//	vendor  vendor_name
//	    device  device_name           <- single tab
//	        subvendor subdevice  name <- two tabs
func parsePCIIDsContent(reader io.ReadCloser) (*PCIIDs, error) {
	if reader == nil {
		return nil, ErrNilReader
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)

	// Pre-allocate maps with estimated sizes to minimize resizing
	result := &PCIIDs{
		Class:  make(map[string]*Class, estimatedClasses),
		Vendor: make(map[string]*Vendor, estimatedVendors),
		Device: make(map[string]*Device, estimatedDevices),
	}

	var (
		state           = stateNone
		currentClass    *Class
		currentVendor   *Vendor
		currentDevice   *Device
		currentSubClass *SubClass
	)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		tabCount := countLeadingTabs(line)

		switch {
		// Class definition: "C xx  class_name"
		case len(line) > 2 && line[0] == 'C' && line[1] == ' ':
			state = stateClass
			id, name := extractIDAndName(line[2:], 2)
			currentClass = &Class{
				ID:       id,
				Name:     name,
				SubClass: make([]*SubClass, 0, 10),
			}
			result.Class[id] = currentClass

		// SubClass: single tab under class
		case tabCount == 1 && state == stateClass && currentClass != nil:
			trimmedLine := line[1:] // Remove leading tab
			id, name := extractIDAndName(trimmedLine, 2)
			currentSubClass = &SubClass{
				ID:               id,
				Name:             name,
				ProgramInterface: make([]*ProgramInterface, 0, 5),
			}
			currentClass.SubClass = append(currentClass.SubClass, currentSubClass)

		// Program Interface: two tabs under class
		case tabCount == 2 && state == stateClass && currentSubClass != nil:
			trimmedLine := line[2:] // Remove two leading tabs
			id, name := extractIDAndName(trimmedLine, 2)
			progIf := &ProgramInterface{ID: id, Name: name}
			currentSubClass.ProgramInterface = append(currentSubClass.ProgramInterface, progIf)

		// Vendor definition: no tabs, not a class
		case tabCount == 0 && line[0] != 'C':
			state = stateVendor
			id, name := extractIDAndName(line, 4)
			currentVendor = &Vendor{
				ID:      id,
				Name:    name,
				Devices: make([]*Device, 0, 16),
			}
			result.Vendor[id] = currentVendor

		// Device: single tab under vendor
		case tabCount == 1 && state == stateVendor && currentVendor != nil:
			trimmedLine := line[1:]
			id, name := extractIDAndName(trimmedLine, 4)
			deviceFullID := buildDeviceID(currentVendor.ID, id)
			currentDevice = &Device{
				ID:           deviceFullID,
				Name:         name,
				VendorID:     currentVendor.ID,
				Subsystem:    make([]*Subsystem, 0, estimatedSubsystem),
				subsystemIdx: make(map[string]*Subsystem, estimatedSubsystem),
			}
			currentVendor.Devices = append(currentVendor.Devices, currentDevice)
			result.Device[deviceFullID] = currentDevice

		// Subsystem: two tabs under vendor (fixed: use "\t" not `\t`)
		case tabCount == 2 && state == stateVendor && currentDevice != nil:
			trimmedLine := line[2:] // Remove two leading tabs
			parts := strings.SplitN(trimmedLine, "  ", 2)
			if len(parts) < 2 {
				continue
			}
			idParts := strings.Fields(parts[0])
			if len(idParts) != 2 {
				continue
			}
			subsysID := buildSubsystemID(idParts[0], idParts[1])
			subsys := &Subsystem{
				ID:          subsysID,
				SubVendorID: idParts[0],
				SubDeviceID: idParts[1],
				Name:        strings.TrimSpace(parts[1]),
			}
			currentDevice.Subsystem = append(currentDevice.Subsystem, subsys)
			currentDevice.subsystemIdx[subsysID] = subsys // O(1) lookup index
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning pci.ids: %w", err)
	}

	return result, nil
}

// Lookup Methods

// FindDeviceByID returns the device with the given ID, or nil if not found.
func (p *PCIIDs) FindDeviceByID(id string) *Device {
	if p == nil || p.Device == nil {
		return nil
	}

	if p.Device[id] == nil {
		return p.Device[strings.ToLower(id)]
	}

	return p.Device[id]
}

// FindVendorByID returns the vendor with the given ID, or nil if not found.
func (p *PCIIDs) FindVendorByID(id string) *Vendor {
	if p == nil || p.Vendor == nil {
		return nil
	}

	if p.Vendor[id] == nil {
		return p.Vendor[strings.ToLower(id)]
	}

	return p.Vendor[id]
}

// FindClassByID returns the class with the given ID, or nil if not found.
func (p *PCIIDs) FindClassByID(id string) *Class {
	if p == nil || p.Class == nil {
		return nil
	}

	if p.Class[id] == nil {
		return p.Class[strings.ToLower(id)]
	}

	return p.Class[id]
}

// GetDeviceName returns the device name or "Unknown" if not found.
func (p *PCIIDs) GetDeviceName(id string) string {
	if device := p.FindDeviceByID(id); device != nil {
		return device.Name
	}
	return "Unknown"
}

// GetSubsystemName returns the subsystem name using O(1) indexed lookup.
func (p *PCIIDs) GetSubsystemName(deviceID, subsystemID string) string {
	device := p.FindDeviceByID(deviceID)
	if device == nil {
		return "Unknown"
	}
	// Use indexed map for O(1) lookup instead of O(n) linear search
	if sub, ok := device.subsystemIdx[subsystemID]; ok {
		return sub.Name
	}
	return "Unknown"
}

// GetVendorName returns the vendor name or "Unknown" if not found.
func (p *PCIIDs) GetVendorName(id string) string {
	if vendor := p.FindVendorByID(id); vendor != nil {
		return vendor.Name
	}
	return "Unknown"
}

// GetClassName returns the class name or "Unknown" if not found.
func (p *PCIIDs) GetClassName(id string) string {
	if class := p.FindClassByID(id); class != nil {
		return class.Name
	}
	return "Unknown"
}
