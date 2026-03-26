// Package raid provides functionality for collecting storage controller and NVMe
// device information. It supports LSI (Broadcom), HPE, Intel VROC, and Adaptec
// RAID controllers, as well as direct-attached NVMe drives via PCI enumeration.
package raid

import (
	"context"
	"errors"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/pci"
	"github.com/zenithax-cc/baize/pkg/utils"
)

// vendorID represents a PCI vendor ID string (4-digit hex, uppercase).
type vendorID string

// Supported RAID controller vendor PCI IDs.
const (
	VendorLSI     vendorID = "1000" // Broadcom / LSI
	VendorHPE     vendorID = "103C" // Hewlett-Packard Enterprise (Smart Array)
	VendorIntel   vendorID = "8086" // Intel VROC (Virtual RAID on CPU)
	VendorAdaptec vendorID = "9005" // Microchip / Adaptec

	// sysfsDevicesPath is the sysfs PCI device directory used for device enumeration.
	sysfsDevicesPath = "/sys/bus/pci/devices"
)

// vendorCtrl associates a PCI vendor ID with its vendor-specific collect function.
type vendorCtrl struct {
	id vendorID
	fn func(context.Context, int, *controller) error
}

// ctrlCollect is the ordered list of supported RAID controller vendors and their
// respective collection handlers. Each entry is tried when a matching PCI vendor ID
// is detected.
var ctrlCollect = []vendorCtrl{
	{id: VendorLSI, fn: collectLSI},
	{id: VendorHPE, fn: collectHPE},
	{id: VendorIntel, fn: collectIntel},
	{id: VendorAdaptec, fn: collectAdaptec},
}

// New creates and returns a new Controllers instance with pre-allocated slices
// for RAID controllers and NVMe devices.
func New() *Controllers {
	return &Controllers{
		Controller: make([]*controller, 0, 2),
		NVMe:       make([]*nvme, 0, 8),
	}
}

// Collect discovers and collects information for all NVMe drives and RAID controllers
// present on the system. Both collection paths run independently; errors are joined.
func (c *Controllers) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	errs := make([]error, 0, 2)

	// Collect NVMe drives first (they are independent of RAID controllers).
	if err := c.collectNVMe(ctx); err != nil {
		errs = append(errs, fmt.Errorf("collect NVMe failed: %w", err))
	}

	// Collect vendor RAID controllers detected via PCI bus enumeration.
	if err := c.collectController(ctx); err != nil {
		errs = append(errs, fmt.Errorf("collect controller failed: %w", err))
	}

	return utils.CombineErrors(errs)
}

// collectController enumerates RAID controller PCI devices, resolves their vendor,
// and delegates to the appropriate vendor-specific collection function.
func (c *Controllers) collectController(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Discover all serial-attached RAID controller PCI bus addresses.
	ctrls, err := pci.GetSerialRAIDPCIBus()
	if err != nil {
		return err
	}

	ctrlCount := len(ctrls)
	if ctrlCount == 0 {
		return nil
	}

	errs := make([]error, 0, ctrlCount)
	for _, ctrl := range ctrls {
		// Collect PCI device information (vendor ID, device ID, etc.).
		p := pci.New(ctrl)
		if err := p.Collect(); err != nil {
			errs = append(errs, fmt.Errorf("collect controller %s pci failed: %w", ctrl, err))
			continue
		}

		ctr := &controller{
			PCIe: p,
		}

		// Match vendor ID and invoke the corresponding vendor handler.
		for _, h := range ctrlCollect {
			if h.id == vendorID(ctr.PCIe.VendorID) {
				if err := h.fn(ctx, ctrlCount, ctr); err != nil {
					errs = append(errs, fmt.Errorf("handle %s controller failed: %w", ctrl, err))
				}
			}
		}

		c.Controller = append(c.Controller, ctr)
	}

	return utils.CombineErrors(errs)
}

// collectNVMe enumerates NVMe PCI devices and collects SMART data for each drive.
// Default media type and form factor are applied before SMART collection.
func (c *Controllers) collectNVMe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Discover all NVMe PCI bus addresses.
	nvmes, err := pci.GetNVMePCIBus()
	if err != nil {
		return err
	}

	nvmeCount := len(nvmes)
	if nvmeCount == 0 {
		return nil
	}

	errs := make([]error, 0, nvmeCount)
	for _, n := range nvmes {
		// Collect PCI device metadata for this NVMe address.
		p := pci.New(n)
		if err := p.Collect(); err != nil {
			errs = append(errs, fmt.Errorf("collect NVMe %s pci failed: %w", n, err))
			continue
		}

		// Initialize NVMe with default physical drive attributes.
		nv := &nvme{
			PCIe: p,
			physicalDrive: physicalDrive{
				RotationRate: "SSD",
				MediaType:    "NVMe SSD",
				FormFactor:   "2.5 inch",
			},
		}

		// Collect SMART data and namespace information via smartctl.
		if err := nv.collect(); err != nil {
			errs = append(errs, fmt.Errorf("collect NVMe %s failed: %w", n, err))
		}

		c.NVMe = append(c.NVMe, nv)
	}

	return errors.Join(errs...)
}

// Name returns the collector identifier used for module routing.
func (c *Controllers) Name() string {
	return "raid"
}

// JSON serializes the Controllers struct to JSON and writes it to stdout.
func (c *Controllers) JSON() error {
	return utils.JSONPrintln(c)
}

// DetailPrintln prints full RAID controller and drive details to stdout.
func (c *Controllers) DetailPrintln() {
	utils.PrinterInstance.Print(c, "RAID INFO")
}

// BriefPrintln prints a brief RAID and NVMe summary to stdout.
func (c *Controllers) BriefPrintln() {
	utils.PrinterInstance.Print(c, "RAID INFO")
}
