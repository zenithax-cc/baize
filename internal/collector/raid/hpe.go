package raid

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/zenithax-cc/baize/common/utils"
)

type hpeController struct {
	*controller
}

var (
	hpePDRegex        = regexp.MustCompile(`physicaldrive (\d+I:\d+:\d+) \(port.*?\)`)
	hpeLDRegex        = regexp.MustCompile(`logicaldrive (\d+) \(.*?\)`)
	hpeBackplaneRegex = regexp.MustCompile(`Internal Drive Cage at Port (\d+I), Box (\d+), ([A-Za-z]+)`)

	failedPDCounter int64
)

func hpeHandle(ctx context.Context, ctrNum int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	hpeCtr := &hpeController{
		controller: c,
	}

	if err := hpeCtr.findController(ctx, ctrNum); err != nil {
		return err
	}

	err := hpeCtr.loadControllerData(ctx)

	*c = *hpeCtr.controller

	return err
}

func (hc *hpeController) findController(ctx context.Context, ctrNum int) error {
	for i := 0; i < ctrNum; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if hc.isControllerAtSlot(i) {
			hc.ID = strconv.Itoa(i)
			return nil
		}
	}

	return fmt.Errorf("HPE controller %s not found", hc.PCIe.PCIeAddr)
}

func (hc *hpeController) isControllerAtSlot(slot int) bool {
	cmd := fmt.Sprintf("%s ctrl slot=%d show | grep -i %s", hpssacliPath, slot, hc.PCIe.PCIeAddr)
	output, err := utils.Run.Command("bash", "-c", cmd)
	return err == nil && len(output) > 0
}

func executeHpssacli(cmd ...string) ([]byte, error) {
	output, err := utils.Run.Command(hpssacliPath, cmd...)
	if err != nil {
		return nil, fmt.Errorf("hpssacli command failed: %w", err)
	}

	return output, nil
}

func (hc *hpeController) loadControllerData(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	output, err := executeHpssacli("ctrl", fmt.Sprintf("slot=%s", hc.ID), "show")
	if err != nil {
		return err
	}

	var multiErr utils.MultiError
	if err := hc.parseControllerData(ctx, output); err != nil {
		multiErr.Add(err)
	}

	pdMatch := hpePDRegex.FindAllStringSubmatch(string(output), -1)
	if len(pdMatch) > 0 {
		hc.PhysicalDrives = make([]*physicalDrive, len(pdMatch))
		for _, pd := range pdMatch {
			if err := hc.parsePhysicalDrive(ctx, pd[1]); err != nil {
				multiErr.Add(err)
			}
		}
	}

	vdMatch := hpeLDRegex.FindAllStringSubmatch(string(output), -1)
	if len(vdMatch) > 0 {
		hc.LogicalDrives = make([]*logicalDrive, len(vdMatch))
		for _, ld := range vdMatch {
			if err := hc.parseLogicalDrive(ctx, ld[1]); err != nil {
				multiErr.Add(err)
			}
		}
	}

	backplaneMatch := hpeBackplaneRegex.FindAllStringSubmatch(string(output), -1)
	if len(backplaneMatch) > 0 {
		hc.Backplanes = make([]*backplane, len(backplaneMatch))
		for _, bp := range backplaneMatch {
			if err := hc.parseBackplane(bp); err != nil {
				multiErr.Add(err)
			}
		}
	}

	return multiErr.Unwrap()
}

type fieldHandlers map[string]func(string)

func (hc *hpeController) parseControllerData(ctx context.Context, info []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	handlers := fieldHandlers{
		"Controller Status": func(v string) { hc.ControllerStatus = v },
		"Controller Mode":   func(v string) { hc.CurrentPersonality = v },
		"Firmware Version":  func(v string) { hc.Firmware = v },
		"Total Cache Size":  func(v string) { hc.CacheSize = v },
		"Interface":         func(v string) { hc.HostInterface = v },
		"Serial Number":     func(v string) { hc.SerialNumber = v },
		"Battery/Capacitor Status": func(v string) {
			hc.Battery = append(hc.Battery, &battery{State: v})
		},
	}

	return hc.parseWithHandler(info, handlers)
}

func (hc *hpeController) parseWithHandler(info []byte, handlers fieldHandlers) error {
	scanner := scannerPool.Get().(*bufio.Scanner)
	defer scannerPool.Put(scanner)

	*scanner = *bufio.NewScanner(bytes.NewReader(info))

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

		if handler, exists := handlers[key]; exists {
			handler(value)
		}

	}

	return scanner.Err()
}

func (hc *hpeController) parsePhysicalDrive(ctx context.Context, slot string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	output, err := executeHpssacli("ctrl", fmt.Sprintf("slot=%s", hc.ID), "pd", slot, "show")
	if err != nil {
		return fmt.Errorf("%s %w", slot, err)
	}

	var multiErr utils.MultiError

	pd := &physicalDrive{
		Location: slot,
	}

	handlers := fieldHandlers{
		"Port": func(v string) { pd.EnclosureId = v },
		"Box":  func(v string) { pd.EnclosureId += ":" + v },
		"Bay": func(v string) {
			pd.SlotId = v
			if did, err := strconv.Atoi(v); err != nil {
				multiErr.Add(fmt.Errorf("convert %s to int: %w", v, err))
				return
			} else {
				pd.DeviceId = strconv.Itoa(did - 1)
			}
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
			if parts := strings.Split(v, "/"); len(parts) == 2 {
				pd.LogicalSectorSize = parts[0] + " B"
				pd.PhysicalSectorSize = parts[1] + " kB"
			}
		},
	}

	if err := hc.parseWithHandler(output, handlers); err != nil {
		multiErr.Add(fmt.Errorf("scanning physical drive %s error: %w", pd.Location, err))
	}

	if err := updatePDSMART(pd); err != nil {
		multiErr.Add(err)
	}

	pdc.Set(pd.Location, pd)

	hc.PhysicalDrives = append(hc.PhysicalDrives, pd)

	return multiErr.Unwrap()
}

func updatePDSMART(pd *physicalDrive) error {

	if pd.State == "Failed" {
		atomic.AddInt64(&failedPDCounter, 1)
		return nil
	}

	if block := GetBlockByWWN(pd.WWN); block != "" {
		pd.MappingFile = block
		if err := pd.getSmartctlData("vroc", "", ""); err != nil {
			return err
		}
		return nil
	}

	if did, err := strconv.Atoi(pd.DeviceId); err == nil {
		adjustedID := did - int(atomic.LoadInt64(&failedPDCounter))
		if err := pd.getSmartctlData("hpe", "", strconv.Itoa(adjustedID)); err != nil {
			return err
		}
	}

	return nil
}

func (hc *hpeController) parseLogicalDrive(ctx context.Context, slot string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	output, err := executeHpssacli("ctrl", fmt.Sprintf("slot=%s", hc.ID), "ld", slot, "show")
	if err != nil {
		return fmt.Errorf("%s %w", slot, err)
	}

	ld := &logicalDrive{
		Location: fmt.Sprintf("/c%s/v%s", hc.ID, slot),
	}

	handlers := fieldHandlers{
		"Size":              func(v string) { ld.Capacity = v },
		"Fault Tolerance":   func(v string) { ld.Type = "RAID " + v },
		"Strip Size":        func(v string) { ld.StripSize = v },
		"Status":            func(v string) { ld.State = v },
		"Caching":           func(v string) { ld.Cache = v },
		"Unique Identifier": func(v string) { ld.ScsiNaaId = v },
		"Disk Name":         func(v string) { ld.MappingFile = v },
	}

	scanner := scannerPool.Get().(*bufio.Scanner)
	defer scannerPool.Put(scanner)
	*scanner = *bufio.NewScanner(bytes.NewReader(output))

	var array string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "physicaldrive") {
			ll := strings.Fields(line)
			if pd, ok := pdc.Get(ll[1]); ok {
				pd.State = "Online"
				pd.MappingFile = ld.MappingFile
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
			continue
		}

		if strings.HasPrefix(line, "Array") {
			array = line
			continue
		}

		if key, value, found := utils.Cut(line, ":"); found {
			if handler, exists := handlers[key]; exists {
				handler(value)
			}
		}
	}

	var multiErr utils.MultiError
	if err := scanner.Err(); err != nil {
		multiErr.Add(fmt.Errorf("error scanning logical drive: %w", err))
	}

	if len(ld.PhysicalDrives) == 0 && array != "" {
		if err := parseArrayPD(ld, hc.ID, array); err != nil {
			multiErr.Add(err)
		}
	}

	hc.LogicalDrives = append(hc.LogicalDrives, ld)

	return multiErr.Unwrap()
}

func parseArrayPD(ld *logicalDrive, id, array string) error {
	output, err := executeHpssacli("ctrl", fmt.Sprintf("slot=%s", id), array, "pd", "all", "show")
	if err != nil {
		return fmt.Errorf("array %s : %w", array, err)
	}

	scanner := scannerPool.Get().(*bufio.Scanner)
	defer scannerPool.Put(scanner)
	*scanner = *bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "physicaldrive") {
			ll := strings.Fields(line)
			if pd, ok := pdc.Get(ll[1]); ok {
				pd.State = "Online"
				pd.MappingFile = ld.MappingFile
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
			continue
		}
	}
	return scanner.Err()
}

func (hc *hpeController) parseBackplane(bp []string) error {
	if len(bp) != 4 {
		return fmt.Errorf("backplane match error:%v", bp[1:])
	}

	id := fmt.Sprintf("%s:%s", bp[1], bp[2])
	ouput, err := executeHpssacli("ctrl", fmt.Sprintf("slot=%s", hc.ID), "enclosure", id, "show")
	if err != nil {
		return fmt.Errorf("enclosure %s: %w", id, err)
	}

	bplane := &backplane{
		Location: id,
		State:    bp[3],
	}

	handlers := fieldHandlers{
		"Drive Bays": func(v string) { bplane.PhysicalDriveCount = v },
		"Port":       func(v string) { bplane.ID = v },
		"Box":        func(v string) { bplane.Slots = v },
		"Location":   func(v string) { bplane.EnclosureType = v },
	}

	return hc.parseWithHandler(ouput, handlers)
}
