package raid

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
	"github.com/zenithax-cc/baize/internal/collector/pci"
)

type vendorID string

const (
	vendorLSI     vendorID = "1000"
	vendorHPE     vendorID = "103c"
	vendorIntel   vendorID = "8086"
	vendorAdaptec vendorID = "9005"
)

const (
	smartctl   = "/usr/sbin/smartctl"
	storcli    = "/usr/local/bin/storcli"
	hpssacli   = "/usr/local/beidou/tool/hpssacli"
	arcconf    = "/usr/local/hwtool/tool/arcconf"
	mdadm      = "/usr/sbin/mdadm"
	sysBusPath = "/sys/bus/pci/devices"
)

var vendorHandlerMap = map[vendorID]func(int, *controller) error{
	vendorLSI:     fromStorcli,
	vendorHPE:     fromHpssacli,
	vendorAdaptec: fromArcconf,
	vendorIntel:   fromMdadm,
}

var (
	pdMap      = map[string]*physicalDrive{}
	pdMapMutex sync.RWMutex
)

func New() *Controllers {
	return &Controllers{
		Controller: make([]*controller, 0, 2),
		NVMe:       make([]*nvme, 0, 8),
	}
}

func (c *Controllers) Collect() error {
	var errs []error

	if err := c.collectNVMe(); err != nil {
		errs = append(errs, err)
	}

	if err := c.collectController(); err != nil {
		errs = append(errs, err)
	}

	return utils.CombineErrors(errs)
}

func (c *Controllers) collectController() error {
	pciBus, err := pci.GetSerialRAIDControllerPCIBus()
	if err != nil {
		return err
	}
	var errs []error
	num := len(pciBus)
	for _, bus := range pciBus {
		p := pci.New(bus)
		if err := p.Collect(); err != nil {
			errs = append(errs, err)
			continue
		}
		ctr := &controller{
			PCIe: p,
		}

		vendor := ctr.PCIe.VendorID
		if handler, ok := vendorHandlerMap[vendorID(vendor)]; ok {
			if err := handler(num, ctr); err != nil {
				errs = append(errs, err)
			}
		}

		c.Controller = append(c.Controller, ctr)
	}

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}

	return nil
}

func (c *Controllers) collectNVMe() error {
	var errs []error
	pciBus, err := pci.GetNVMeControllerPCIBus()
	if err != nil {
		if len(pciBus) == 0 {
			return nil
		}
		return err
	}

	for _, bus := range pciBus {
		n := &nvme{
			PCIe: pci.New(bus),
		}

		if err := n.PCIe.Collect(); err != nil {
			errs = append(errs, err)
		}

		if err := n.nvmePhysicalDrive(); err != nil {
			errs = append(errs, err)
		}

		c.NVMe = append(c.NVMe, n)
	}

	if len(errs) > 0 {
		return utils.CombineErrors(errs)
	}

	return nil
}

func (c *Controllers) String() string {
	return ""
}

func (c *Controllers) Json() {
	if err := c.Collect(); err != nil {
		fmt.Printf("Failed to collect raid information: %v", err)
	}

	j, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		fmt.Printf("marshal raid information to json failed: %v", err)
	}

	fmt.Println(string(j))
}

func (n *nvme) nvmePhysicalDrive() error {
	dirs, err := utils.ReadDir(filepath.Join(sysBusPath, n.PCIe.PCIeAddr, "nvme"))
	if err != nil || len(dirs) == 0 {
		return err
	}

	pd := &physicalDrive{
		MappingFile:  fmt.Sprintf("/dev/%s", dirs[0].Name()),
		Interface:    "U.2",
		MediumType:   "NVMe SSD",
		RotationRate: "Solid State Device",
		FormFactor:   "2.5 inch",
	}

	err = pd.getSmartctlData("nvme", "", "")
	n.physicalDrive = *pd

	if _, ok := pdMap[pd.MappingFile]; !ok {
		pdMap[pd.MappingFile] = pd
	}

	return err
}
