package cpu

import (
	"context"
	"errors"
	"fmt"

	"github.com/zenithax-cc/baize/pkg/utils"
)

// CPU architecture constants used for vendor-specific logic.
const (
	archARM = "aarch"
	archX86 = "x86"

	// Power state modes for CPU governor.
	powerStatePerformance = "Performance"
	powerStatePowerSaving = "PowerSaving"

	// Hyper-Threading status values reported in output.
	htSupported         = "Supported Enabled"
	htNotSupported      = "Not Supported"
	htSupportedDisabled = "Supported Disabled"

	// Diagnose result constants.
	diagnoseHealthy   = "Healthy"
	diagnoseUnhealthy = "Unhealthy"

	// SMBIOS processor status indicating a populated and enabled socket.
	statusPopulatedEnabled = "Populated, Enabled"
)

// socketIDMap maps common socket designation strings to normalized zero-based
// socket index strings, enabling consistent cross-platform CPU socket matching.
var (
	socketIDMap = map[string]string{
		"P0": "0", "Proc 1": "0", "CPU 1": "0", "CPU01": "0", "CPU1": "0", "Socket 1": "0",
		"P1": "1", "Proc 2": "1", "CPU 2": "1", "CPU02": "1", "CPU2": "1", "Socket 2": "1",
	}
)

// New creates and returns a new CPU instance with default values:
// HyperThreading is pre-set to "Supported Enabled" and PowerState to "PowerSaving".
func New() *CPU {
	return &CPU{
		HyperThreading: htSupported,
		PowerState:     powerStatePowerSaving,
		CPUEntries:     make([]*SMBIOSCPUEntry, 0, 2),
	}
}

// Collect gathers all CPU information by invoking multiple sub-collectors
// (lscpu, SMBIOS, turbostat) and then associating per-core data.
// All errors from sub-collectors are joined and returned together.
func (c *CPU) Collect(ctx context.Context) error {
	errs := make([]error, 0, 4)

	// Collect basic CPU info from lscpu command output.
	if err := c.collectFromLscpu(); err != nil {
		errs = append(errs, err)
	}

	// Collect detailed per-socket data from SMBIOS (dmidecode type 4).
	if err := c.collectFromSMBIOS(ctx); err != nil {
		errs = append(errs, err)
	}

	// Collect per-thread frequency and temperature via turbostat.
	if err := c.collectFromTurbostat(ctx); err != nil {
		errs = append(errs, err)
	}

	// Associate threads and temperature readings to their SMBIOS CPU entries.
	if err := c.associateCores(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// Name returns the collector identifier string used for module routing.
func (c *CPU) Name() string {
	return "cpu"
}

// JSON serializes the CPU struct to JSON and writes it to stdout.
func (c *CPU) JSON() error {
	return utils.JSONPrintln(c)
}

// DetailPrintln prints full CPU details including per-thread entries to stdout.
func (c *CPU) DetailPrintln() {
	cpu := struct {
		CPUInfo []*CPU `json:"cpu" name:"CPU INFO" output:"both"`
	}{}

	cpu.CPUInfo = append(cpu.CPUInfo, c)

	utils.SP.Print(cpu, "detail")
}

// BriefPrintln prints a brief CPU summary (key metrics only) to stdout.
func (c *CPU) BriefPrintln() {
	cpu := struct {
		CPUInfo []*CPU `json:"cpu" name:"CPU INFO" output:"both"`
	}{}

	cpu.CPUInfo = append(cpu.CPUInfo, c)

	utils.SP.Print(cpu, "brief")
}

// associateCores links per-thread turbostat data (frequency, temperature) to
// the corresponding SMBIOS CPU entry based on socket designation mapping.
// It also collects vendor-specific per-core temperatures (Intel/AMD).
func (c *CPU) associateCores() error {

	var (
		err     error
		errs    []error
		tempMap map[string]int
	)

	// Collect per-core temperatures according to the CPU vendor.
	switch c.VendorID {
	case "Intel":
		tempMap, err = collectIntelTemperature()
	case "AMD":
		tempMap, err = collectAMDTemperature()
	}

	if err != nil {
		errs = append(errs, err)
	}

	for _, entry := range c.CPUEntries {
		// Resolve the socket designation to a normalized physical ID index.
		id, ok := socketIDMap[entry.SocketDesignation]
		if !ok {
			errs = append(errs, errors.New("socket designation not found"))
			continue
		}
		for _, thread := range c.threads {
			// Assign temperature from per-core key (physicalID-coreID) if available.
			if temp, ok := tempMap[thread.PhysicalID+"-"+thread.CoreID]; ok {
				thread.Temperature = fmt.Sprintf("%d ℃", temp)
			}

			// Fallback: assign package-level temperature keyed by physical socket ID.
			if temp, ok := tempMap[thread.PhysicalID]; ok {
				thread.Temperature = fmt.Sprintf("%d ℃", temp)
			}

			// Attach this thread to the matching SMBIOS CPU entry.
			if thread.PhysicalID == id {
				entry.ThreadEntries = append(entry.ThreadEntries, thread)
			}
		}
	}

	return errors.Join(errs...)
}
