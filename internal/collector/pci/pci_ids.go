package pci

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
)

type PCIIDs struct {
	Class  map[string]*class  `json:"Class,omitempty"`
	Vendor map[string]*vendor `json:"Vendor,omitempty"`
	Device map[string]*device `json:"Device,omitempty"`
}

type class struct {
	ID       string      `json:"ID,omitempty"`
	Name     string      `json:"Name,omitempty"`
	SubClass []*subClass `json:"SubClass,omitempty"`
}

type subClass struct {
	ID               string              `json:"ID,omitempty"`
	Name             string              `json:"Name,omitempty"`
	ProgramInterface []*programInterface `json:"ProgramInterface,omitempty"`
}

type programInterface struct {
	ID   string `json:"ID,omitempty"`
	Name string `json:"Name,omitempty"`
}

type vendor struct {
	ID      string    `json:"ID,omitempty"`
	Name    string    `json:"Name,omitempty"`
	Devices []*device `json:"Devices,omitempty"`
}

type device struct {
	ID        string       `json:"ID,omitempty"`
	Name      string       `json:"Name,omitempty"`
	VendorID  string       `json:"VendorID,omitempty"`
	Subsystem []*subsystem `json:"Subsystem,omitempty"`
}

type subsystem struct {
	ID          string `json:"ID,omitempty"`
	SubVendorID string `json:"SubVendorID,omitempty"`
	SubDeviceID string `json:"SubDeviceID,omitempty"`
	Name        string `json:"Name,omitempty"`
}

var (
	pciidsPath = []string{
		"/usr/share/misc/pci.ids",
		"/usr/share/misc/pci.ids.gz",
		"/usr/share/hwdata/pci.ids",
		"/usr/share/hwdata/pci.ids.gz",
	}
	pciidsURL = "https://raw.githubusercontent.com/pciutils/pciids/master/pci.ids"

	instance    *PCIIDs
	pciidsOnece sync.Once
)

func NewPCIIDs() *PCIIDs {
	pciidsOnece.Do(func() {
		file := getPCIIDsFile()
		if file == "" {
			return
		}
		content, err := getPCIIDsContent(file)
		if err != nil {
			return
		}
		instance, err = parsePCIIDsContent(content)
		if err != nil {
			return
		}
	})

	return instance
}

// GetPCIIDs returns the obtained pci.ids file.
func getPCIIDsFile() string {
	for _, file := range pciidsPath {
		if utils.FileExists(file) {
			return file
		}
	}
	tmpPath := "/tmp/pci.ids"
	resp, err := http.Get(pciidsURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	err = os.WriteFile(tmpPath, respBody, 0644)
	if err != nil {
		return ""
	}
	return tmpPath
}

// getPCIIDsContent returns the content of the pci.ids file.
func getPCIIDsContent(file string) (io.ReadCloser, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open file %s error: %w", file, err)
	}
	if strings.HasSuffix(file, ".gz") {
		uzip, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("read gzip file %s error: %w", file, err)
		}
		defer uzip.Close()
		return uzip, nil
	}
	return f, nil
}

func countLeadingTabs(s string) int {
	count := 0
	for _, c := range s {
		if c == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

func extractIDAndName(s string, idLen int) (string, string) {
	if len(s) < idLen {
		return s, ""
	}
	id := strings.TrimSpace(s[:idLen])
	name := strings.TrimSpace(s[idLen+1:])
	return id, name
}

const (
	stateNone = iota
	stateClass
	stateVendor
	stateDevice
	stateSubClass
	stateProgramInterface
	stateSubsystem
)

// parsePCIIDsContent parses the content of the pci.ids file.
func parsePCIIDsContent(irc io.ReadCloser) (*PCIIDs, error) {
	if irc == nil {
		return nil, fmt.Errorf("invalid io.ReadCloser")
	}
	defer irc.Close()

	scan := bufio.NewScanner(irc)

	res := &PCIIDs{
		Class:  make(map[string]*class, 20),
		Vendor: make(map[string]*vendor, 200),
		Device: make(map[string]*device, 1000),
	}

	var (
		state            = stateNone
		currentClass     *class
		currentVendor    *vendor
		currentDevice    *device
		currentSubClass  *subClass
		currentSubsystem *subsystem
	)

	linenum := 0
	for scan.Scan() {
		linenum++
		line := scan.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tabCount := countLeadingTabs(line)
		switch {
		// Syntax:
		// C class   class_name
		// 		subclass    subclass_name       <-- single tab
		// 			prog-if  prog-if_name   <-- two tabs
		case strings.HasPrefix(line, "C "):
			state = stateClass
			id, name := extractIDAndName(line[2:], 2)
			currentClass = &class{
				ID:       id,
				Name:     name,
				SubClass: make([]*subClass, 0, 10),
			}
			res.Class[id] = currentClass
		case tabCount == 1 && state == stateClass:
			line = strings.TrimLeft(line, "\t")
			id, name := extractIDAndName(line, 2)
			currentSubClass = &subClass{
				ID:               id,
				Name:             name,
				ProgramInterface: make([]*programInterface, 0, 10),
			}
			currentClass.SubClass = append(currentClass.SubClass, currentSubClass)
		case tabCount == 2 && state == stateClass:
			line = strings.TrimLeft(line, "\t")
			id, name := extractIDAndName(line, 2)
			programInterface := &programInterface{
				ID:   id,
				Name: name,
			}
			currentSubClass.ProgramInterface = append(currentSubClass.ProgramInterface, programInterface)

		// Syntax:
		// vendor  vendor_name
		// 		device  device_name             <-- single tab
		// 			subvendor subdevice  subsystem_name <-- two tabs
		case tabCount == 0 && !strings.HasPrefix(line, "C "):
			state = stateVendor
			id, name := extractIDAndName(line, 4)
			currentVendor = &vendor{
				ID:      id,
				Name:    name,
				Devices: make([]*device, 0, 16),
			}
			res.Vendor[id] = currentVendor

		case tabCount == 1 && state == stateVendor:
			line = strings.TrimLeft(line, "\t")
			id, name := extractIDAndName(line, 4)
			deviceFullID := fmt.Sprintf("%s:%s", currentVendor.ID, id)
			currentDevice = &device{
				ID:        deviceFullID,
				Name:      name,
				VendorID:  currentVendor.ID,
				Subsystem: make([]*subsystem, 0, 16),
			}
			currentVendor.Devices = append(currentVendor.Devices, currentDevice)
			res.Device[deviceFullID] = currentDevice
		case tabCount == 2 && state == stateVendor:
			line = strings.TrimLeft(line, `\t`)
			parts := strings.SplitN(line, "  ", 2)
			if len(parts) < 2 || len(parts[0]) < 9 {
				continue
			}
			idParts := strings.Fields(parts[0])
			if len(idParts) != 2 {
				continue
			}
			currentSubsystem = &subsystem{
				ID:          fmt.Sprintf("%s:%s", idParts[0], idParts[1]),
				SubVendorID: idParts[0],
				SubDeviceID: idParts[1],
				Name:        strings.TrimSpace(parts[1]),
			}
			currentDevice.Subsystem = append(currentDevice.Subsystem, currentSubsystem)
		}
	}

	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("error scanning pci.ids file: %w", err)
	}
	return res, nil
}

func (p *PCIIDs) FindDeviceByID(id string) *device {
	if p == nil || p.Device == nil {
		return nil
	}
	if device, ok := p.Device[id]; ok {
		return device
	}
	return nil
}

func (p *PCIIDs) FindVendorByID(id string) *vendor {
	if p == nil || p.Vendor == nil {
		return nil
	}
	if vendor, ok := p.Vendor[id]; ok {
		return vendor
	}
	return nil
}

func (p *PCIIDs) FindClassByID(id string) *class {
	if p == nil || p.Class == nil {
		return nil
	}
	if class, ok := p.Class[id]; ok {
		return class
	}
	return nil
}

func (p *PCIIDs) getDeviceNameByID(id string) string {
	device := p.FindDeviceByID(id)
	if device == nil {
		return "Unknown"
	}
	return device.Name
}

func (p *PCIIDs) getSubDeviceNameByID(did, sid string) string {
	device := p.FindDeviceByID(did)
	if device == nil {
		return "Unknown"
	}
	for _, subDevice := range device.Subsystem {
		if subDevice.ID == sid {
			return subDevice.Name
		}
	}
	return "Unknown"
}

func (p *PCIIDs) getVendorNameByID(id string) string {
	vendor := p.FindVendorByID(id)
	if vendor == nil {
		return "Unknown"
	}
	return vendor.Name
}

func (p *PCIIDs) getClassNameByID(id string) string {
	class := p.FindClassByID(id)
	if class == nil {
		return "Unknown"
	}
	return class.Name
}
