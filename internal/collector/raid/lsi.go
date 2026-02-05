package raid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	storcli = "/usr/local/bin/storcli"
)

type lsiController struct {
	ctrl *controller
	cid  string
}

func collectLSI(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	lsiCtr := &lsiController{
		ctrl: c,
	}

	if !lsiCtr.isFound(i) {
		return fmt.Errorf("lsi controller %s not found", c.PCIe.PCIAddr)
	}

	if err := lsiCtr.collect(ctx); err != nil {
		return err
	}

	return nil
}

func (lc *lsiController) isFound(i int) bool {
	for j := 0; j < i+1; j++ {
		output := execute.Command(storcli, "/c"+strconv.Itoa(j), "show")
		if output.Err != nil {
			continue
		}

		if len(output.Stdout) > 0 && bytes.Contains(output.Stdout, []byte(lc.ctrl.PCIe.PCIAddr)) {
			lc.cid = strconv.Itoa(j)
			return true
		}
	}

	return false
}

func storcliCmd(ctx context.Context, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	output := execute.Command(storcli, args...)
	if output.Err != nil {
		return nil, output.Err
	}

	return output.Stdout, nil
}

func (lc *lsiController) collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := storcliCmd(ctx, "/c"+lc.cid, "show", "J")
	if err != nil {
		return err
	}

	var js showJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return fmt.Errorf("unmarshal %s show to json: %w", lc.cid, err)
	}

	errs := make([]error, 0, 5)
	if err := lc.parseCtrlCard(ctx); err != nil {
		errs = append(errs, err)
	}

	res := js.Controllers[0].ResponseData
	if err := lc.collectCtrlPD(ctx, res.PDList); err != nil {
		errs = append(errs, err)
	}

	if err := lc.collectCtrlLD(ctx, res.VDList); err != nil {
		errs = append(errs, err)
	}

	if err := lc.collectCtrlEnclosure(ctx, res.EnclosureList); err != nil {
		errs = append(errs, err)
	}

	if err := lc.collectCtrlBattery(ctx, res.CacheVaultInfo); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (lc *lsiController) parseCtrlCard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := storcliCmd(ctx, "/c"+lc.cid, "show", "all", "J")
	if err != nil {
		return err
	}

	var js showAllJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return fmt.Errorf("unmarshal %s show all to json: %w", lc.cid, err)
	}

	res := js.Controllers[0].ResponseData

	if b := res.Basics; b != nil {
		lc.ctrl.ProductName = b.Model
		lc.ctrl.SerialNumber = b.SN
		lc.ctrl.ControllerTime = b.CTD
		lc.ctrl.SasAddress = b.SAS
	}

	if v := res.Version; v != nil {
		lc.ctrl.BiosVersion = v.BiosVersion
		lc.ctrl.FwVersion = v.FirmwareVer
		lc.ctrl.Firmware = v.FirmwarePackge
	}

	if b := res.Bus; b != nil {
		lc.ctrl.HostInterface = b.HostInterface
		lc.ctrl.DeviceInterface = b.DeviceInterface
	}

	if s := res.Status; s != nil {
		lc.ctrl.ControllerStatus = s.ControllerStatus
		lc.ctrl.MemoryCorrectableErrors = strconv.Itoa(s.MemoryCeErr)
		lc.ctrl.MemoryUncorrectableErrors = strconv.Itoa(s.MemoryUeErr)
	}

	if a := res.Adapter; a != nil {
		lc.ctrl.SurpportedJBOD = a.SupportJBOD
		lc.ctrl.ForeignConfigImport = a.ForeignConfigImport
	}

	if h := res.HwCfg; h != nil {
		lc.ctrl.ChipRevision = h.ChipRevision
		lc.ctrl.FrontEndPortCount = strconv.Itoa(h.FrontEndPortCount)
		lc.ctrl.BackendPortCount = strconv.Itoa(h.BackendPortCount)
		lc.ctrl.NVRAMSize = h.NVRAMSize
		lc.ctrl.FlashSize = h.FlashSize
		lc.ctrl.CacheSize = h.OnBoardMemorySize
	}

	if c := res.Capabilities; c != nil {
		lc.ctrl.SupportedDrives = c.SupportedDrives
		lc.ctrl.RaidLevelSupported = c.RaidLevelSupported
		lc.ctrl.EnableJBOD = c.EnableJBOD
	}

	return nil
}

func (lc *lsiController) collectCtrlPD(ctx context.Context, pds []*pdList) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(pds) == 0 {
		return nil
	}

	if lc.ctrl.PhysicalDrives == nil {
		lc.ctrl.PhysicalDrives = make([]*physicalDrive, 0, len(pds))
	}

	errs := make([]error, 0, len(pds))

	for _, pd := range pds {
		if err := lc.parseCtrlPD(ctx, pd); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (lc *lsiController) parseCtrlPD(ctx context.Context, pd *pdList) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	res := &physicalDrive{
		DeviceId:           strconv.Itoa(pd.DID),
		State:              pd.State,
		Capacity:           pd.Size,
		MediaType:          pd.Med,
		ProtocolType:       pd.Intf,
		ModelName:          pd.Model,
		PhysicalSectorSize: pd.SeSz,
		DG:                 parseDG(pd.DG),
	}

	eid, sid, found := strings.Cut(pd.EIDSlt, ":")
	if !found {
		return fmt.Errorf("unexcepted EIDSlt: %s", pd.EIDSlt)
	}
	res.EnclosureId = eid
	res.SlotId = sid
	res.Location = "/c" + lc.cid + "/e" + res.EnclosureId + "/s" + res.SlotId

	data, err := storcliCmd(ctx, res.Location, "show", "all")
	if err != nil {
		return err
	}

	scanner := utils.NewScanner(bytes.NewReader(data))

	return nil
}

func parseDG(dg any) string {
	switch v := dg.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return "Unknown"
	}
}
