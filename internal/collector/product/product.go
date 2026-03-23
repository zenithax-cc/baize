package product

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/zenithax-cc/baize/pkg/utils"
)

type collectTask struct {
	name string
	fn   func() error
}

func New() *Product {
	return &Product{}
}

func (p *Product) Collect(ctx context.Context) error {
	tasks := []collectTask{
		{name: "kernel", fn: p.collectKernel},
		{name: "distribution", fn: p.collectDistribution},
		{name: "bios", fn: p.collectBIOS},
		{name: "system", fn: p.collectSystem},
		{name: "baseboard", fn: p.collectBaseBoard},
		{name: "chassis", fn: p.collectChassis},
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

			select {
			case <-ctx.Done():
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", t.name, ctx.Err()))
				mu.Unlock()
				return
			default:
			}
			if err := t.fn(); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", t.name, err))
				mu.Unlock()
			}
		}(task)
	}

	wg.Wait()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (p *Product) Name() string {
	return "product"
}

func (p *Product) JSON() error {
	return utils.JSONPrintln(p)
}

// DetailPrintln prints full product details (all SMBIOS sub-sections) to stdout.
func (p *Product) DetailPrintln() {
	p.BriefPrintln()
}

// BriefPrintln prints a concise server identity summary to stdout.
func (p *Product) BriefPrintln() {
	brief := ProductBrief{
		Manufacturer: p.System.Manufacturer,
		ProductName:  p.System.ProductName,
		SerialNumber: p.System.SerialNumber,
		UUID:         p.System.UUID,
		AssetTag:     p.Chassis.AssetTag,
		ChassisType:  p.Chassis.Type,
		HostName:     p.OS.HostName,
		BIOSVersion:  p.BIOS.Version,
		BIOSDate:     p.BIOS.ReleaseDate,
	}

	// Build a user-friendly OS display string.
	if p.OS.PrettyName != "" {
		brief.OS = p.OS.PrettyName
	} else if p.OS.Distr != "" {
		brief.OS = p.OS.Distr + " " + p.OS.MinorVersion
	}

	brief.Kernel = p.OS.KernelRelease

	wrapper := struct {
		Products []*ProductBrief `name:"PRODUCT INFO" output:"both"`
	}{
		Products: []*ProductBrief{&brief},
	}

	utils.SP.Print(wrapper, "brief")
}
