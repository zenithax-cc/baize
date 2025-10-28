package raid

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

type arcController controller

func adaptecHandle(ctx context.Context, ctrNum int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	vendor, err := utils.Run.Command("dmidecode", "-s", "system-manufacturer")
	if err != nil {
		return fmt.Errorf("adaptec dmidecode failed: %w", err)
	}

	if strings.ToLower(strings.TrimSpace(string(vendor))) == "hpe" {
		return hpeHandle(ctx, ctrNum, c)
	}

	arcCtr := (*arcController)(c)
	if err := arcCtr.checkController(ctrNum); err != nil {
		return err
	}
	var errs []error
	if err := arcCtr.getController(); err != nil {
		errs = append(errs, err)
	}

	if err := arcCtr.getPhysicalDrives(); err != nil {
		errs = append(errs, err)
	}

	if err := arcCtr.getLogicalDrives(); err != nil {
		errs = append(errs, err)
	}

	return utils.CombineErrors(errs)
}

func (a *arcController) checkController(ctrNum int) error {
	snFile := "/sys/bus/pci/devices/" + a.PCIe.PCIeAddr + "/host0/scsi_host/host0/serial_number"
	sn, err := os.ReadFile(snFile)
	if err != nil {
		return fmt.Errorf("read %s error: %w", snFile, err)
	}

	a.SerialNumber = string(bytes.TrimSpace(sn))
	for i := 0; i < ctrNum+1; i++ {
		output, err := utils.Run.Command("bash", "-c", fmt.Sprintf("%s GETCONFIG %d AD | grep %s", arcconfPath, i, a.SerialNumber))
		if err == nil && len(output) > 0 {
			a.ID = strconv.Itoa(i)
			return nil
		}
	}

	return fmt.Errorf("not found RAID controller id by: %s", a.PCIe.PCIeAddr)
}

func (a *arcController) getController() error {
	ctr, err := utils.Run.Command(arcconfPath, "GETCONFIG", a.ID, "AD")
	if err != nil {
		return fmt.Errorf("arcconf  failed: %w", err)
	}

	fieldMap := map[string]*string{
		"Controller Status": &a.ControllerStatus,
		"Controller Mode":   &a.CurrentPersonality,
		"Controller Model":  &a.ProductName,
		"Installed memory":  &a.CacheSize,
		"BIOS":              &a.BiosVersion,
		"Firmware":          &a.FwVersion,
	}

	scanner := bufio.NewScanner(bytes.NewReader(ctr))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}
		if key == "Logical devices/Failed/Degraded" {
			val := strings.Split(value, "/")
			a.NumberOfRaid = val[0]
			a.FailedRaid = val[1]
			a.DegradedRaid = val[2]
			continue
		}

		if field, ok := fieldMap[key]; ok {
			*field = value
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning controller: %w", err)
	}

	return nil
}

func (a *arcController) getPhysicalDrives() error {
	pds, err := utils.Run.Command(arcconfPath, "GETCONFIG", a.ID, "PD")
	if err != nil {
		return fmt.Errorf("arcconf pd failed: %w", err)
	}

	pdList := strings.Split(string(pds), "\n\n")

	for _, pd := range pdList {
		if !strings.Contains(pd, "Device is a Hard drive") {
			continue
		}
		a.parsePhysicalDrive(pd)
	}
	return nil
}

func (a *arcController) parsePhysicalDrive(content string) error {
	var res physicalDrive
	fieldMap := map[string]*string{
		"State":           &res.State,
		"Block Size":      &res.PhysicalSectorSize,
		"Transfer Speed":  &res.LinkSpeed,
		"Vendor":          &res.Vendor,
		"Model":           &res.ModelName,
		"Firmware":        &res.FirmwareVersion,
		"Serial Number":   &res.SN,
		"World-wide name": &res.WWN,
		"Write cache":     &res.WriteCache,
		"S.M.A.R.T.":      &res.SmartAlert,
	}
	var errs []error
	h, _ := strconv.Atoi(a.ID)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}
		if key == "Reported Location" {
			val := strings.Split(value, ",")
			res.EnclosureId = strings.Fields(val[0])[1]
			res.SlotId = strings.Fields(val[1])[1]
			res.Location = fmt.Sprintf("/c%s/e%s/s%s", a.ID, res.EnclosureId, res.SlotId)
			hlid := fmt.Sprintf("%d,%s,%s", h-1, res.EnclosureId, res.SlotId)
			if err := res.collectSMARTData(SMARTConfig{Option: "aacraid", DeviceID: hlid}); err != nil {
				errs = append(errs, err)
			}
			continue
		}

		if field, ok := fieldMap[key]; ok {
			*field = value
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error scanning physical drive: %w", err))
	}

	a.PhysicalDrives = append(a.PhysicalDrives, &res)
	pdc.Set(res.SN, &res)

	return utils.CombineErrors(errs)
}

func (a *arcController) getLogicalDrives() error {
	lds, err := utils.Run.Command(arcconfPath, "GETCONFIG", a.ID, "LD")
	if err != nil {
		return fmt.Errorf("arcconf ld failed: %w", err)
	}
	ldList := strings.Split(string(lds), "\n\n")
	var errs []error
	for _, ld := range ldList {
		if !strings.Contains(ld, "Logical Device number") {
			continue
		}
		if err := a.parseLogicalDrive(ld); err != nil {
			errs = append(errs, err)
		}
	}

	return utils.CombineErrors(errs)
}

func (a *arcController) parseLogicalDrive(content string) error {
	var res logicalDrive
	fieldMap := map[string]*string{
		"Logical Device name":    &res.Location,
		"RAID Level":             &res.Type,
		"State of Logical Drive": &res.State,
	}
	var errs []error
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Logical Device number") {
			ldn := strings.Fields(line)
			res.VD = ldn[len(ldn)-1]
			continue
		}

		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}

		if strings.HasPrefix(key, "Segment ") {
			val := strings.Fields(value)
			if pd, ok := pdc.Get(val[len(val)-1]); ok {
				res.PhysicalDrives = append(res.PhysicalDrives, pd)
			}
			continue
		}

		if key == "Size" {
			val := strings.Fields(value)
			if len(val) != 2 {
				res.Capacity = value
			}
			s, err := strconv.ParseFloat(val[0], 64)
			if err != nil {
				errs = append(errs, fmt.Errorf("convert size to float64: %w", err))
				continue
			}
			size, err := utils.ConvertUnit(s, val[1], true)
			if err != nil {
				errs = append(errs, fmt.Errorf("convert size to human failed: %w", err))
				continue
			}
			res.Capacity = size
		}

		if field, ok := fieldMap[key]; ok {
			*field = value
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error scanning logical drive: %w", err))
	}

	a.LogicalDrives = append(a.LogicalDrives, &res)

	return utils.CombineErrors(errs)
}
