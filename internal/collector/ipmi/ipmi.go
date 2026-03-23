// Package ipmi provides functionality for collecting IPMI (Intelligent Platform
// Management Interface) hardware information from the local BMC via ipmitool.
//
// It concurrently collects:
//   - BMC device info and LAN configuration (ipmitool bmc info / lan print)
//   - All sensor readings grouped by type (ipmitool sensor)
//   - Power supply status and system power consumption (ipmitool sdr / dcmi)
//   - Filtered System Event Log entries (ipmitool sel elist)
//
// If ipmitool is not present or the BMC is unreachable, all sub-tasks are
// skipped gracefully and a non-fatal error is returned.
package ipmi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/pkg/utils"
)

// collectTask pairs a human-readable task name with its collection function.
type collectTask struct {
	name string
	fn   func(context.Context) error
}

// New creates and returns an initialised IPMI collector.
func New() *IPMI {
	return &IPMI{}
}

// Collect runs all IPMI sub-collectors concurrently.
// Individual sub-task errors are joined and returned together; they do not
// prevent other sub-tasks from completing.
func (m *IPMI) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tasks := []collectTask{
		{name: "bmc", fn: m.collectBMC},
		{name: "sensors", fn: m.collectSensors},
		{name: "power", fn: m.collectPowerSupplies},
		{name: "sel", fn: m.collectSEL},
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(t collectTask) {
			defer wg.Done()

			// Respect context cancellation before launching each sub-task.
			select {
			case <-ctx.Done():
				mu.Lock()
				errs = append(errs, fmt.Errorf("ipmi %s: %w", t.name, ctx.Err()))
				mu.Unlock()
				return
			default:
			}

			if err := t.fn(ctx); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("ipmi %s: %w", t.name, err))
				mu.Unlock()
			}
		}(task)
	}

	wg.Wait()

	// Run diagnosis after all data is collected.
	m.diagnose()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// diagnose evaluates collected data to produce a top-level health summary.
// It sets Diagnose to "OK" when no issues are found, or to a brief description
// of the most-severe problem detected.
func (m *IPMI) diagnose() {
	var issues []string

	// Check for critical/warning SEL entries.
	criticalCount, warningCount := 0, 0
	for _, e := range m.SEL {
		switch e.Severity {
		case "Critical":
			criticalCount++
		case "Warning":
			warningCount++
		}
	}
	if criticalCount > 0 {
		issues = append(issues, fmt.Sprintf("%d critical SEL event(s)", criticalCount))
	}
	if warningCount > 0 {
		issues = append(issues, fmt.Sprintf("%d warning SEL event(s)", warningCount))
	}

	// Check for sensors in non-ok status.
	sensorIssues := m.checkSensorStatus()
	issues = append(issues, sensorIssues...)

	// Check PSU health.
	for _, psu := range m.PowerSupplies {
		lower := strings.ToLower(psu.Status)
		if strings.Contains(lower, "fail") || strings.Contains(lower, "absent") {
			issues = append(issues, fmt.Sprintf("PSU %s: %s", psu.Name, psu.Status))
		}
	}

	if len(issues) == 0 {
		m.Diagnose = "OK"
	} else {
		m.Diagnose = "WARNING"
		m.DiagnoseDetail = strings.Join(issues, "; ")
	}
}

// checkSensorStatus scans all sensor groups for non-"ok" statuses and
// returns a slice of human-readable issue descriptions.
func (m *IPMI) checkSensorStatus() []string {
	if m.Sensors == nil {
		return nil
	}

	var issues []string
	allSensors := make([]*Sensor, 0, 32)
	allSensors = append(allSensors, m.Sensors.Temperature...)
	allSensors = append(allSensors, m.Sensors.Voltage...)
	allSensors = append(allSensors, m.Sensors.Fan...)
	allSensors = append(allSensors, m.Sensors.Current...)
	allSensors = append(allSensors, m.Sensors.Other...)

	for _, s := range allSensors {
		status := strings.ToLower(s.Status)
		// "cr" = critical, "nc" = non-critical, "nr" = non-recoverable
		if status == "cr" || status == "nr" || status == "lnc" || status == "unc" {
			issues = append(issues, fmt.Sprintf("sensor %q: %s (%s)", s.Name, s.Value, s.Status))
		}
	}
	return issues
}

// Name returns the module identifier used by the collector Manager.
func (m *IPMI) Name() string {
	return "ipmi"
}

// JSON serializes the IPMI struct to indented JSON and writes it to stdout.
func (m *IPMI) JSON() error {
	return utils.JSONPrintln(m)
}

// BriefPrintln prints a brief IPMI summary (BMC info + diagnosis) to stdout.
func (m *IPMI) BriefPrintln() {
	// Build a flat brief view for the IPMI module.
	type IPMIBrief struct {
		FirmwareRevision string `name:"BMC Firmware" output:"both"`
		IPMIVersion      string `name:"IPMI Version" output:"both"`
		ManagementIP     string `name:"Management IP" output:"both" color:"DefaultGreen"`
		MACAddress       string `name:"MAC Address" output:"both"`
		Diagnose         string `name:"Diagnose" output:"both" color:"Diagnose"`
		DiagnoseDetail   string `name:"Diagnose Detail" output:"both" color:"Diagnose"`
	}

	brief := &IPMIBrief{
		FirmwareRevision: m.BMC.FirmwareRevision,
		IPMIVersion:      m.BMC.IPMIVersion,
		ManagementIP:     m.BMC.ManagementIP,
		MACAddress:       m.BMC.MACAddress,
		Diagnose:         m.Diagnose,
		DiagnoseDetail:   m.DiagnoseDetail,
	}

	wrapper := struct {
		Items []*IPMIBrief `name:"IPMI INFO" output:"both"`
	}{
		Items: []*IPMIBrief{brief},
	}

	utils.SP.Print(wrapper, "brief")
}

// DetailPrintln prints full IPMI details (sensors, SEL, PSU) to stdout.
func (m *IPMI) DetailPrintln() {
	wrapper := struct {
		Items []*IPMI `name:"IPMI INFO" output:"both"`
	}{
		Items: []*IPMI{m},
	}

	utils.SP.Print(wrapper, "detail")
}
