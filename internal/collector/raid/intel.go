package raid

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
)

const (
	procMdstat = "/proc/mdstat"
	mdadm      = "/usr/sbin/mdadm"
)

type intelController struct {
	ctrl    *controller
	pds     []string
	pciAddr string
}

var (
	intelControllers []*intelController
	intelOnece       sync.Once
)

func collectIntel(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

func (ic *intelController) collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	errs := make([]error, 0, 4)
	if err := ic.collectCtrlCard(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := ic.collectCtrlPD(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := ic.collectCtrlLD(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func mdadmCMD(ctx context.Context, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	output := execute.Command(mdadm, args...)
	if output.Err != nil {
		return nil, output.Err
	}

	return output.Stdout, nil
}

func (ic *intelController) collectCtrlCard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := mdadmCMD(ctx, "--detail-platform")
	if err != nil {
		return err
	}

	ctrls := bytes.SplitSeq(data, []byte("\n\n"))
	var errs []error

	for ctrl := range ctrls {
		if err := ic.parseCtrlCard(ctx, bytes.TrimSpace(ctrl)); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (ic *intelController) parseCtrlCard(ctx context.Context, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}

		if v == "" {
			continue
		}

		switch {
		case k == "RAID Levels":
			ic.ctrl.RaidLevelSupported = v
		case k == "Max Disks":
			ic.ctrl.SupportedDrives = v
		case k == "I/O Controller":
			if ic.pciAddr != "" {
				continue
			}
			ic.pciAddr = filepath.Base(strings.Fields(v)[0])
		case k == "NVMe under VMD":
			disk := strings.Fields(v)[0]
			ic.pds = append(ic.pds, disk)
		case strings.HasPrefix(k, "Port"):
			if !strings.Contains(v, "no device attached") {
				disk := strings.Fields(v)[0]
				ic.pds = append(ic.pds, k+" "+disk)
			}
		}
	}

	return scanner.Err()
}

func (ic *intelController) collectCtrlPD(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	pds := utils.GetBlockByLsblk()
	if len(pds) == 0 {
		return nil
	}

	errs := make([]error, 0, len(pds))
	for _, pd := range pds {
		if err := ic.parseCtrlPD(ctx, pd); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (ic *intelController) parseCtrlPD(ctx context.Context, pd string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	res := &physicalDrive{
		MappingFile: "/dev/" + pd,
	}

	err := res.collectSMARTData(SMARTConfig{Option: "jbod", BlockDevice: res.MappingFile})

	return err
}

func (ic *intelController) collectCtrlLD(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.Open(procMdstat)
	if err != nil {
		return err
	}
	defer file.Close()

	var errs []error
	scanner := utils.NewScanner(file)
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}
		if v == "" || strings.HasPrefix(v, "active") {
			continue
		}

		if err := ic.parseCtrlLD(ctx, k); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (ic *intelController) parseCtrlLD(ctx context.Context, md string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ld := &logicalDrive{
		MappingFile: "/dev/" + md,
	}

	data, err := mdadmCMD(ctx, "--detail", ld.MappingFile)
	if err != nil {
		return err
	}

	fields := []field{
		{"Raid Level", &ld.Type},
		{"Array Size", &ld.Capacity},
		{"Total Devices", &ld.NumberOfDrives},
		{"State", &ld.State},
		{"Consistency Policy", &ld.Cache},
		{"UUID", &ld.ScsiNaaId},
	}

	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}

		if v == "" && strings.Contains(k, "/dev/") {
			idx := strings.IndexByte(k, '/')
			println(v[idx:])
			ld.pds = append(ld.pds, v[idx:])
		}

		for _, f := range fields {
			if f.key == k {
				if k == "Array Size" {
					v = strings.Fields(v)[0]
				}
				*f.value = v
				break
			}
		}
	}

	ic.ctrl.LogicalDrives = append(ic.ctrl.LogicalDrives, ld)

	return scanner.Err()
}
