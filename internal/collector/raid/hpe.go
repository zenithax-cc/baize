package raid

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
)

const (
	hpssacli = "/usr/local/beidou/tool/hpssacli"
)

type hpeController struct {
	ctrl      *controller
	cid       string
	failedPDs uint32
}

var (
	hpePDRegex        = regexp.MustCompile(`physicaldrive (\d+I:\d+:\d+) \(port.*?\)`)
	hpeLDRegex        = regexp.MustCompile(`logicaldrive (\d+) \(.*?\)`)
	hpeEnclosureRegex = regexp.MustCompile(`Internal Drive Cage at Port (\d+I), Box (\d+), ([A-Za-z]+)`)
)

func collectHPE(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	hpeCtr := &hpeController{
		ctrl: c,
	}

	if !hpeCtr.isFound(i) {
		return fmt.Errorf("hpe controller %s not found", c.PCIe.PCIAddr)
	}

	err := hpeCtr.collect(ctx)

	hpeCtr.associate()

	return err
}

func (h *hpeController) isFound(num int) bool {
	for i := 0; i < num; i++ {
		cmd := fmt.Sprintf("%s ctrl slot=%d show | grep -i %s", hpssacli, i, h.ctrl.PCIe.PCIAddr)
		output := execute.ShellCommand(cmd)
		if output.Err == nil && len(output.Stdout) > 0 {
			h.cid = strconv.Itoa(i)
			return true
		}
	}

	return false
}

func hpssacliCmd(ctx context.Context, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	output := execute.CommandWithContext(ctx, hpssacli, args...)
	if output.Err != nil {
		return nil, output.Err
	}

	return output.Stdout, nil
}

func (h *hpeController) collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := hpssacliCmd(ctx, "ctrl", "slot="+h.cid, "show", "config")
	if err != nil {
		return err
	}

	errs := make([]error, 0, 4)
	if err := h.parseCtrlCard(ctx); err != nil {
		errs = append(errs, err)
	}

	pds := hpePDRegex.FindAllSubmatch(data, -1)
	if err := h.collectCtrlPD(ctx, pds); err != nil {
		errs = append(errs, err)
	}

	lds := hpeLDRegex.FindAllSubmatch(data, -1)
	if err := h.collectCtrlLD(ctx, lds); err != nil {
		errs = append(errs, err)
	}

	els := hpeEnclosureRegex.FindAllSubmatch(data, -1)
	if err := h.collectCtrlEnclosure(ctx, els); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (h *hpeController) parseCtrlCard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := hpssacliCmd(ctx, "ctrl", "slot="+h.cid, "show")
	if err != nil {
		return fmt.Errorf("controller %s: %w", h.cid, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	fieldMap := map[string]*string{
		"Controller Status": &h.ctrl.ControllerStatus,
		"Controller Mode":   &h.ctrl.CurrentPersonality,
		"Firmware Version":  &h.ctrl.Firmware,
		"Total Cache Size":  &h.ctrl.CacheSize,
		"Interface":         &h.ctrl.HostInterface,
		"Serial Number":     &h.ctrl.SerialNumber,
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Smart ") || strings.HasPrefix(line, "HPE Smart Array") {
			h.ctrl.ProductName = line
			continue
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "Battery/Capacitor Status" {
			h.ctrl.Battery = append(h.ctrl.Battery, &battery{State: value})
		}

		if field, ok := fieldMap[key]; ok {
			*field = value
		}
	}

	return scanner.Err()
}

func (h *hpeController) collectCtrlPD(ctx context.Context, pds [][][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(pds) == 0 {
		return nil
	}

	errs := make([]error, 0, len(pds))
	h.ctrl.PhysicalDrives = make([]*physicalDrive, 0, len(pds)/2)
	for _, pd := range pds {
		if err := h.parseCtrlPD(ctx, pd[1]); err != nil {
			errs = append(errs, fmt.Errorf("parse %s pd: %w", string(pd[1]), err))
			continue
		}
	}

	return utils.CombineErrors(errs)
}

func (h *hpeController) parseCtrlPD(ctx context.Context, p []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := hpssacliCmd(ctx, "ctrl", "slot="+h.cid, "pd", string(p), "show")
	if err != nil {
		return err
	}

	pd := &physicalDrive{
		Location: string(p),
	}

	fieldFunc := map[string]func(string){
		"Port": func(v string) { pd.EnclosureId = v },
		"Box":  func(v string) { pd.EnclosureId += ":" + v },
		"Bay": func(v string) {
			pd.SlotId = v
			did, _ := strconv.Atoi(v)
			pd.DeviceId = strconv.Itoa(did - 1)
		},
		"Status":                  func(v string) { pd.State = v },
		"Interface Type":          func(v string) { pd.ProtocolType = v },
		"Size":                    func(v string) { pd.Capacity = v },
		"Firmware Revision":       func(v string) { pd.FirmwareVersion = v },
		"Serial Number":           func(v string) { pd.SN = v },
		"WWID":                    func(v string) { pd.WWN = v },
		"Model":                   func(v string) { pd.ModelName = v },
		"Current Temperature (C)": func(v string) { pd.Temperature = v + " ℃" },
		"PHY Transfer Rate":       func(v string) { pd.DeviceSpeed = v },
		"Logical/Physical Block Size": func(v string) {
			if parts := strings.Split(v, "/"); len(parts) == 2 {
				pd.LogicalSectorSize = parts[0] + " B"
				pd.PhysicalSectorSize = parts[1] + " kB"
			}
		},
	}

	var errs []error
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if handler, ok := fieldFunc[key]; ok {
			handler(value)
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, err)
	}

	if err := h.hpeSMART(pd); err != nil {
		errs = append(errs, err)
	}

	h.ctrl.PhysicalDrives = append(h.ctrl.PhysicalDrives, pd)

	return errors.Join(errs...)
}

func (h *hpeController) hpeSMART(pd *physicalDrive) error {

	if pd.State == "Failed" {
		atomic.AndUint32(&h.failedPDs, 1)
		return nil
	}

	if block := utils.GetBlockByWWN(pd.WWN); block != "" {
		pd.MappingFile = block
		if err := pd.collectSMARTData(SMARTConfig{Option: "jbod"}); err != nil {
			return err
		}
		return nil
	}

	block := "/dev/" + utils.GetOneBlock()

	if did, err := strconv.Atoi(pd.DeviceId); err == nil {
		useID := did - int(atomic.LoadUint32(&h.failedPDs))
		if err := pd.collectSMARTData(SMARTConfig{Option: "cciss", BlockDevice: block, DeviceID: strconv.Itoa(useID)}); err != nil {
			return err
		}
	}

	return nil
}

func (h *hpeController) collectCtrlLD(ctx context.Context, lds [][][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(lds) == 0 {
		return nil
	}

	errs := make([]error, 0, len(lds))
	h.ctrl.LogicalDrives = make([]*logicalDrive, 0, len(lds)/2)

	for _, ld := range lds {
		if err := h.parseCtrlLD(ctx, ld[1]); err != nil {
			errs = append(errs, fmt.Errorf("parse ld %s: %w", string(ld[1]), err))
		}
	}

	return utils.CombineErrors(errs)
}

func (h *hpeController) parseCtrlLD(ctx context.Context, ld []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := hpssacliCmd(ctx, "ctrl", "slot="+h.cid, "ld", string(ld), "show")
	if err != nil {
		return err
	}

	res := &logicalDrive{
		Location: fmt.Sprintf("/c%s/v%s", h.cid, ld),
	}

	fieldsMap := map[string]func(string){
		"Size":              func(v string) { res.Capacity = v },
		"Fault Tolerance":   func(v string) { res.Type = "RAID " + v },
		"Strip Size":        func(v string) { res.StripSize = v },
		"Status":            func(v string) { res.State = v },
		"Caching":           func(v string) { res.Cache = v },
		"Unique Identifier": func(v string) { res.ScsiNaaId = v },
		"Disk Name":         func(v string) { res.MappingFile = v },
	}

	var array string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "physicaldrive") {
			parts := strings.Fields(line)
			res.pds = append(res.pds, parts[1])
			continue
		}

		if strings.HasPrefix(line, "Array") {
			array = line
			continue
		}

		if key, value, found := strings.Cut(line, ":"); found {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if fn, exists := fieldsMap[key]; exists {
				fn(value)
			}
		}
	}

	var errs []error

	if err := scanner.Err(); err != nil {
		errs = append(errs, err)
	}

	if len(res.pds) == 0 && array != "" {
		if err := parseArrayPD(res, h.cid, array); err != nil {
			errs = append(errs, err)
		}
	}

	h.ctrl.LogicalDrives = append(h.ctrl.LogicalDrives, res)

	return utils.CombineErrors(errs)
}

func parseArrayPD(res *logicalDrive, cid, array string) error {
	data, err := hpssacliCmd(context.Background(), "ctrl", fmt.Sprintf("slot=%s", cid), array, "pd", "all", "show")
	if err != nil {
		return fmt.Errorf("%s : %w", array, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "physicaldrive") {
			parts := strings.Fields(line)
			res.pds = append(res.pds, parts[1])
		}
	}

	return scanner.Err()
}

func (h *hpeController) collectCtrlEnclosure(ctx context.Context, els [][][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(els) == 0 {
		return nil
	}

	errs := make([]error, 0, len(els))
	for _, el := range els {

		if err := h.parseCtrlEnclosure(ctx, el); err != nil {
			errs = append(errs, fmt.Errorf("parse enclosure %s: %w", string(el[1]), err))
		}
	}
	return nil
}

func (h *hpeController) parseCtrlEnclosure(ctx context.Context, el [][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(el) != 4 {
		return fmt.Errorf("enclosure match error:%v", el[1:])
	}

	res := &enclosure{
		Location: fmt.Sprintf("%s:%s", el[1], el[2]),
		State:    string(el[3]),
	}

	data, err := hpssacliCmd(ctx, "ctrl", fmt.Sprintf("slot=%s", h.cid), "enclosure", res.Location, "show")
	if err != nil {
		return fmt.Errorf("enclosure %s: %w", res.Location, err)
	}

	fieldsMap := map[string]*string{
		"Drive Bays": &res.PhysicalDriveCount,
		"Port":       &res.ID,
		"Box":        &res.Slots,
		"Location":   &res.EnclosureType,
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if field, exists := fieldsMap[key]; exists {
			*field = value
		}
	}

	h.ctrl.Backplanes = append(h.ctrl.Backplanes, res)
	return scanner.Err()
}

func (h *hpeController) associate() {
	for _, ld := range h.ctrl.LogicalDrives {
		for _, disk := range ld.pds {
			for _, pd := range h.ctrl.PhysicalDrives {
				if disk == pd.Location {
					ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
					break
				}
			}
		}
	}
}
