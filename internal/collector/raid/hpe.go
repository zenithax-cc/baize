package raid

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

type hpeController controller

var (
	hpePDRegex        = regexp.MustCompile(`physicaldrive (\d+I:\d+:\d+) \(port.*?\)`)
	hpeLDRegex        = regexp.MustCompile(`logicaldrive (\d+) \(.*?\)`)
	hpeBackplaneRegex = regexp.MustCompile(`Internal Drive Cage at Port (\d+I), Box (\d+), ([A-Za-z]+)`)
	failedPD          = 0
)

func hpeHandle(ctx context.Context, ctrNum int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	hpeCtr := (*hpeController)(c)

	if err := hpeCtr.checkController(ctrNum); err != nil {
		return err
	}

	err := hpeCtr.getController()

	*c = (controller)(*hpeCtr)
	return err
}

func (hc *hpeController) checkController(ctrNum int) error {
	for i := 0; i <= ctrNum; i++ {
		output, err := utils.Run.Command("bash", "-c", fmt.Sprintf("%s ctrl slot=%d show | grep -i %s", hpssacli, i, hc.PCIe.PCIeAddr))
		if err == nil && len(output) > 0 {
			hc.ID = strconv.Itoa(i)
			return nil
		}
	}
	return fmt.Errorf("not found HPE controller %s", hc.PCIe.PCIeAddr)
}

func (hc *hpeController) getController() error {
	output, err := utils.Run.Command(hpssacli, "ctrl", fmt.Sprintf("slot=%s", hc.ID), "show", "config")
	if err != nil {
		return fmt.Errorf("hpssacli failed: %w", err)
	}

	var errs []error
	if err := hc.parseController(); err != nil {
		errs = append(errs, err)
	}

	pdMatch := hpePDRegex.FindAllStringSubmatch(string(output), -1)
	for _, pd := range pdMatch {
		if err := hc.parsePhysicalDrive(pd[1]); err != nil {
			errs = append(errs, err)
		}
	}

	ldMatch := hpeLDRegex.FindAllStringSubmatch(string(output), -1)
	for _, ld := range ldMatch {
		if err := hc.parseLogicalDrive(ld[1]); err != nil {
			errs = append(errs, err)
		}
	}

	backplaneMatch := hpeBackplaneRegex.FindAllStringSubmatch(string(output), -1)
	for _, bp := range backplaneMatch {
		if err := hc.parseBackplane(bp); err != nil {
			errs = append(errs, err)
		}
	}

	return utils.CombineErrors(errs)
}

func (hc *hpeController) parseController() error {
	output, err := utils.Run.Command(hpssacli, "ctrl", fmt.Sprintf("slot=%s", hc.ID), "show")
	if err != nil {
		return fmt.Errorf("hpssacli failed: %w", err)
	}

	fieldMap := map[string]*string{
		"Interface":         &hc.HostInterface,
		"Serial Number":     &hc.SerialNumber,
		"Controller Status": &hc.ControllerStatus,
		"Firmware Version":  &hc.Firmware,
		"Total Cache Size":  &hc.CacheSize,
		"Controller Mode":   &hc.CurrentPersonality,
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Smart ") || strings.HasPrefix(line, "HPE Smart Array") {
			hc.ProductName = line
			continue
		}

		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}

		if ptr, exists := fieldMap[key]; exists {
			*ptr = value
		}

		if key == "Battery/Capacitor Status" {
			hc.Battery = append(hc.Battery, &battery{
				State: value,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning controller: %w", err)
	}

	return nil
}

func (hc *hpeController) parsePhysicalDrive(slot string) error {
	output, err := utils.Run.Command(hpssacli, "ctrl", fmt.Sprintf("slot=%s", hc.ID), "pd", slot, "show")
	if err != nil {
		return fmt.Errorf("hpssacli %s failed: %w", slot, err)
	}

	var errs []error
	pd := &physicalDrive{
		Location: slot,
	}
	fieldMap := map[string]func(string){
		"Port": func(v string) { pd.EnclosureId = v },
		"Box":  func(v string) { pd.EnclosureId += ":" + v },
		"Bay": func(v string) {
			pd.SlotId = v
			did, err := strconv.Atoi(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("convert %s to int: %w", v, err))
				return
			}
			pd.DeviceId = strconv.Itoa(did - 1)
		},
		"Status":                  func(v string) { pd.State = v },
		"Interface Type":          func(v string) { pd.Interface = v },
		"Size":                    func(v string) { pd.Capacity = v },
		"Firmware Revision":       func(v string) { pd.Firmware = v },
		"Serial Number":           func(v string) { pd.SN = v },
		"WWID":                    func(v string) { pd.WWN = v },
		"Model":                   func(v string) { pd.Model = v },
		"Current Temperature (C)": func(v string) { pd.Temperature = v + " ℃" },
		"PHY Transfer Rate":       func(v string) { pd.DeviceSpeed = v },
		"Logical/Physical Block Size": func(v string) {
			block := strings.Split(v, "/")
			pd.LogicalSectorSize = block[0] + " B"
			pd.PhysicalSectorSize = block[1] + " kB"
		},
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}

		if handler, exists := fieldMap[key]; exists {
			handler(value)
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error scanning physical drive: %w", err))
	}

	if pd.State == "Failed" {
		failedPD++
	} else {
		if did, err := strconv.Atoi(pd.DeviceId); err == nil {
			if err := pd.getSmartctlData("hpe", "", fmt.Sprintf("%d", did-failedPD)); err != nil {
				errs = append(errs, err)
			}
		}
	}

	pdMapMutex.Lock()
	if _, ok := pdMap[pd.Location]; !ok {
		pdMap[pd.Location] = pd
	}
	pdMapMutex.Unlock()

	hc.PhysicalDrives = append(hc.PhysicalDrives, pd)

	return utils.CombineErrors(errs)
}

func (hc *hpeController) parseLogicalDrive(slot string) error {
	output, err := utils.Run.Command(hpssacli, "ctrl", fmt.Sprintf("slot=%s", hc.ID), "ld", slot, "show")
	if err != nil {
		return fmt.Errorf("hpssacli ld %s failed: %w", slot, err)
	}

	var errs []error
	ld := &logicalDrive{
		Location: fmt.Sprintf("/c%s/v%s", hc.ID, slot),
	}
	fieldMap := map[string]*string{
		"Size":              &ld.Capacity,
		"Fault Tolerance":   &ld.Type,
		"Strip Size":        &ld.StripSize,
		"Status":            &ld.State,
		"Caching":           &ld.Cache,
		"Unique Identifier": &ld.ScsiNaaId,
		"Disk Name":         &ld.MappingFile,
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var array string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "physicaldrive") {
			ll := strings.Fields(line)
			pdMapMutex.Lock()
			if pd, ok := pdMap[ll[1]]; ok {
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
			pdMapMutex.Unlock()
			continue
		}

		if strings.HasPrefix(line, "Array") {
			array = line
			continue
		}

		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}

		if ptr, exists := fieldMap[key]; exists {
			if key == "Fault Tolerance" {
				*ptr = "RAID" + value
			} else {
				*ptr = value
			}
		}
	}

	if len(ld.PhysicalDrives) == 0 {
		if err := parseArrayPD(ld, hc.ID, array); err != nil {
			errs = append(errs, err)
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error scanning logical drive: %w", err))
	}

	hc.LogicalDrives = append(hc.LogicalDrives, ld)

	return utils.CombineErrors(errs)
}

func parseArrayPD(ld *logicalDrive, id, array string) error {
	output, err := utils.Run.Command(hpssacli, "ctrl", fmt.Sprintf("slot=%s", id), array, "pd", "all", "show")
	if err != nil {
		return fmt.Errorf("hpssacli ld %s failed: %w", array, err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var errs []error
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "physicaldrive") {
			ll := strings.Fields(line)
			pdMapMutex.Lock()
			if pd, ok := pdMap[ll[1]]; ok {
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
			pdMapMutex.Unlock()
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error scanning logical drive: %w", err))
	}

	return utils.CombineErrors(errs)
}

func (hc *hpeController) parseBackplane(bp []string) error {
	if len(bp) != 4 {
		return fmt.Errorf("backplane match error:%v", bp[1:])
	}

	id := fmt.Sprintf("%s:%s", bp[1], bp[2])
	ouput, err := utils.Run.Command(hpssacli, "ctrl", fmt.Sprintf("slot=%s", hc.ID), "enclosure", id, "show")
	if err != nil {
		return fmt.Errorf("hpssacli enclosure %s failed: %w", id, err)
	}

	bplane := &backplane{
		Location: id,
		State:    bp[3],
	}
	fieldMap := map[string]*string{
		"Drive Bays": &bplane.PhysicalDriveCount,
		"Port":       &bplane.ID,
		"Box":        &bplane.Slots,
		"Location":   &bplane.EnclosureType,
	}

	scanner := bufio.NewScanner(bytes.NewReader(ouput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, ":")
		if !find {
			continue
		}

		if ptr, exists := fieldMap[key]; exists {
			*ptr = value
		}
	}
	hc.Backplanes = append(hc.Backplanes, bplane)
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning backplane: %w", err)
	}
	return nil
}
