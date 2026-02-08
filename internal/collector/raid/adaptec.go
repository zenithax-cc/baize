package raid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
)

const (
	snFile  = "/host0/scsi_host/host0/serial_number"
	arcconf = "/usr/local/hwtool/tool/arcconf"
)

type adaptecController struct {
	ctrl *controller
	cid  string
}

type field struct {
	key   string
	value *string
}

func collectAdaptec(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	arcCtr := &adaptecController{
		ctrl: c,
	}

	output := execute.Command("dmidecode", "-s", "system-manufacturer")
	if output.Err != nil {
		return output.Err
	}

	if bytes.HasPrefix(bytes.TrimSpace(output.Stdout), []byte("HP")) {
		return collectHPE(ctx, i, c)
	}

	if !arcCtr.isFound(i) {
		return fmt.Errorf("adaptec controller %s not found", c.PCIe.PCIAddr)
	}

	err := arcCtr.collect(ctx)
	arcCtr.associate()

	return err
}

func (ac *adaptecController) isFound(i int) bool {
	sn, err := os.ReadFile(sysfsDevicesPath + ac.ctrl.PCIe.PCIAddr + snFile)
	if err != nil {
		return false
	}

	for j := 0; j < i+1; j++ {
		output := execute.Command(arcconf, "GETCONFIG", strconv.Itoa(i), "AD")
		if output.Err != nil {
			continue
		}
		if len(output.Stdout) > 0 && bytes.Contains(output.Stdout, bytes.TrimSpace(sn)) {
			ac.cid = strconv.Itoa(j)
			return true
		}
	}

	return false
}

func arcconfCmd(ctx context.Context, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	output := execute.CommandWithContext(ctx, arcconf+" GETCONFIG", args...)
	if output.Err != nil {
		return nil, output.Err
	}

	return output.Stdout, nil
}

func (ac *adaptecController) collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var errs []error
	if err := ac.parseCtrlCard(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := ac.collectCtrlPD(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := ac.collectCtrlLD(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (ac *adaptecController) parseCtrlCard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := arcconfCmd(ctx, ac.cid, "AD")
	if err != nil {
		return err
	}

	ctrlFields := []field{
		{"Controller Status", &ac.ctrl.ControllerStatus},
		{"Controller Mode", &ac.ctrl.CurrentPersonality},
		{"Controller Model", &ac.ctrl.ProductName},
		{"Installed memory", &ac.ctrl.CacheSize},
		{"BIOS", &ac.ctrl.BiosVersion},
		{"Firmware", &ac.ctrl.FwVersion},
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

		if k == "Logical devices/Failed/Degraded" {
			val := strings.SplitN(v, "/", 3)
			if len(val) >= 3 {
				ac.ctrl.NumberOfRaid = val[0]
				ac.ctrl.FailedRaid = val[1]
				ac.ctrl.DegradedRaid = val[2]
			}
		}

		for _, field := range ctrlFields {
			if field.key == k {
				*field.value = v
			}
		}
	}

	return scanner.Err()
}

func (ac *adaptecController) collectCtrlPD(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := arcconfCmd(ctx, ac.cid, "PD")
	if err != nil {
		return err
	}

	pds := bytes.Split(data, []byte("\n\n"))
	errs := make([]error, 0, len(pds))
	for _, pd := range pds {
		if !bytes.Contains(pd, []byte("Device is a Hard drive")) {
			continue
		}
		if err := ac.parseCtrlPD(ctx, pd); err != nil {
			errs = append(errs, err)
		}
	}

	return nil
}

func (ac *adaptecController) parseCtrlPD(ctx context.Context, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	res := &physicalDrive{}
	pdFields := []field{
		{"State", &res.State},
		{"Block Size", &res.PhysicalSectorSize},
		{"Transfer Speed", &res.LinkSpeed},
		{"Vendor", &res.Vendor},
		{"Model", &res.ModelName},
		{"Firmware", &res.FirmwareVersion},
		{"Serial Number", &res.SN},
		{"World-wide name", &res.WWN},
		{"Write cache", &res.WriteCache},
		{"S.M.A.R.T.", &res.SmartAlert},
	}

	errs := make([]error, 0, 2)

	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}
		if v == "" {
			continue
		}

		if k == "Reported Location" {
			val := strings.Split(v, ",")
			if len(val) >= 2 {
				res.EnclosureId = strings.Fields(val[0])[1]
				res.SlotId = strings.Fields(val[1])[1]
				res.Location = fmt.Sprintf("/c%s/e%s/s%s", ac.cid, res.EnclosureId, res.SlotId)
			}
			continue
		}

		for _, field := range pdFields {
			if field.key == k {
				*field.value = v
			}
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, err)
	}

	cid, _ := strconv.Atoi(ac.cid)
	if err := res.collectSMARTData(SMARTConfig{
		Option:      "aacraid",
		BlockDevice: "/dev" + utils.GetOneBlock(),
		DeviceID:    fmt.Sprintf("%d,%s,%s", cid, res.EnclosureId, res.SlotId),
	}); err != nil {
		errs = append(errs, err)
	}

	ac.ctrl.PhysicalDrives = append(ac.ctrl.PhysicalDrives, res)

	return errors.Join(errs...)
}

func (ac *adaptecController) collectCtrlLD(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := arcconfCmd(ctx, ac.cid, "LD")
	if err != nil {
		return err
	}

	lds := bytes.Split(data, []byte("\n\n"))
	errs := make([]error, 0, len(lds))

	for _, ld := range lds {
		if !bytes.Contains(ld, []byte("Logical Device number")) {
			continue
		}
		if err := ac.parseCtrlLD(ctx, ld); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (ac *adaptecController) parseCtrlLD(ctx context.Context, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	res := &logicalDrive{}
	ldFields := []field{
		{"Logical Device name", &res.Location},
		{"RAID Level", &res.Type},
		{"State of Logical Drive", &res.State},
		{"Size", &res.Capacity},
	}

	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine(":")
		if !hasMore {
			break
		}
		if strings.HasPrefix(k, "Logical Device number") {
			parts := strings.Fields(k)
			res.VD = parts[len(parts)-1]
			continue
		}

		if v == "" {
			continue
		}

		if strings.HasPrefix(k, "Segment ") {
			parts := strings.Fields(v)
			res.pds = append(res.pds, parts[len(parts)-1])
			continue
		}

		for _, field := range ldFields {
			if field.key == k {
				*field.value = v
			}
		}
	}

	ac.ctrl.LogicalDrives = append(ac.ctrl.LogicalDrives, res)

	return scanner.Err()
}

func (ac *adaptecController) associate() {
	for _, ld := range ac.ctrl.LogicalDrives {
		for _, disk := range ld.pds {
			for _, pd := range ac.ctrl.PhysicalDrives {
				if disk == pd.SN {
					ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
					break
				}
			}
		}
	}
}
