package raid

import (
	"context"
	"fmt"

	"github.com/zenithax-cc/baize/internal/collector/pci"
	"github.com/zenithax-cc/baize/pkg/utils"
)

type vendorID string

const (
	VendorLSI     vendorID = "1000"
	VendorHPE     vendorID = "103C"
	VendorIntel   vendorID = "8086"
	VendorAdaptec vendorID = "9005"

	sysfsDevicesPath = "/sys/bus/pci/devices"
)

type vendorCtrl struct {
	id vendorID
	fn func(context.Context, int, *controller) error
}

var ctrlCollect = []vendorCtrl{
	{id: VendorLSI, fn: collectLSI},
	{id: VendorHPE, fn: collectHPE},
	{id: VendorIntel, fn: collectIntel},
	{id: VendorAdaptec, fn: collectAdaptec},
}

func New() *Controllers {
	return &Controllers{
		Controller: make([]*controller, 0, 2),
		NVMe:       make([]*nvme, 0, 8),
	}
}

func (c *Controllers) Collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	errs := make([]error, 0, 2)
	if err := c.collectNVMe(ctx); err != nil {
		errs = append(errs, fmt.Errorf("collect NVMe failed: %w", err))
	}

	if err := c.collectController(ctx); err != nil {
		errs = append(errs, fmt.Errorf("collect controller failed: %w", err))
	}

	return utils.CombineErrors(errs)
}

func (c *Controllers) collectController(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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
		p := pci.New(ctrl)
		if err := p.Collect(); err != nil {
			errs = append(errs, fmt.Errorf("collect controller %s pci failed: %w", ctrl, err))
			continue
		}

		ctr := &controller{
			PCIe: p,
		}

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

func (c *Controllers) collectNVMe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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
		p := pci.New(n)
		if err := p.Collect(); err != nil {
			errs = append(errs, fmt.Errorf("collect NVMe %s pci failed: %w", n, err))
			continue
		}

		nv := &nvme{
			PCIe: p,
			physicalDrive: physicalDrive{
				RotationRate: "SSD",
				MediaType:    "NVMe SSD",
				FormFactor:   "2.5 inch",
			},
		}

		if err := nv.collect(); err != nil {
			errs = append(errs, fmt.Errorf("collect NVMe %s failed: %w", n, err))
		}

		c.NVMe = append(c.NVMe, nv)
	}

	return nil
}
