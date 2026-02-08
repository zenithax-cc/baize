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
	"github.com/zenithax-cc/baize/pkg/utils"
)

const (
	storcli = "/usr/local/bin/storcli"
)

type lsiController struct {
	ctrl *controller
	cid  string
}

//var pdRegexp = regexp.MustCompile(`^(.+):(\d+)`)

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

	err := lsiCtr.collect(ctx)
	lsiCtr.associate()

	return err
}

func (lc *lsiController) isFound(i int) bool {
	pcieAddr := lc.ctrl.PCIe.PCIAddr[2 : len(lc.ctrl.PCIe.PCIAddr)-3]
	for j := 0; j < i+1; j++ {
		output := execute.Command(storcli, "/c"+strconv.Itoa(j), "show")
		if output.Err != nil {
			continue
		}

		if len(output.Stdout) > 0 && bytes.Contains(output.Stdout, []byte(pcieAddr)) {
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

	pdFields := []field{
		{"Shield Counter", &res.ShieldCounter},
		{"Media Error Count", &res.MediaErrorCount},
		{"Other Error Count", &res.OtherErrorCount},
		{"Predictive Failure Count", &res.PredictiveFailureCount},
		{"Drive Temperature", &res.Temperature},
		{"S.M.A.R.T alert flagged by drive", &res.SmartAlert},
		{"SN", &res.SN},
		{"WWN", &res.WWN},
		{"Firmware Revision", &res.FirmwareVersion},
		{"Device Speed", &res.DeviceSpeed},
		{"Link Speed", &res.LinkSpeed},
		{"Logical Sector Size", &res.LogicalSectorSize},
		{"Physical Sector Size", &res.PhysicalSectorSize},
	}

	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine("=")
		if !hasMore {
			break
		}
		if v == "" {
			continue
		}

		for _, f := range pdFields {
			if f.key == k {
				*f.value = v
			}
		}
	}

	errs := make([]error, 0, 2)
	if err := scanner.Err(); err != nil {
		errs = append(errs, err)
	}

	if err := res.collectSMARTData(SMARTConfig{Option: "megaraid", DeviceID: res.DeviceId, ControllerID: lc.cid}); err != nil {
		errs = append(errs, err)
	}

	lc.ctrl.PhysicalDrives = append(lc.ctrl.PhysicalDrives, res)

	return errors.Join(errs...)
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

func (lc *lsiController) collectCtrlLD(ctx context.Context, vds []*vdList) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(vds) == 0 {
		return nil
	}

	if lc.ctrl.LogicalDrives == nil {
		lc.ctrl.LogicalDrives = make([]*logicalDrive, 0, len(vds))
	}

	errs := make([]error, 0, len(vds))

	for _, vd := range vds {
		if err := lc.parseCtrlLD(ctx, vd); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (lc *lsiController) parseCtrlLD(ctx context.Context, vd *vdList) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ld := &logicalDrive{
		Type:     vd.Level,
		State:    vd.State,
		Capacity: vd.Size,
		Consist:  vd.Consist,
		Access:   vd.Access,
		Cache:    vd.Cache,
	}

	dg, vid, found := strings.Cut(vd.DGVD, "/")
	if !found {
		return fmt.Errorf("unexcepted DG/VD: %s", vd.DGVD)
	}
	ld.DG = dg
	ld.VD = vid
	ld.Location = "/c" + lc.cid + "/v" + vid

	data, err := storcliCmd(ctx, ld.Location, "show", "all")
	if err != nil {
		return err
	}

	fields := []field{
		{"Strip Size", &ld.StripSize},
		{"Number of Blocks", &ld.NumberOfBlocks},
		{"Number of Drives Per Span", &ld.NumberOfDrivesPerSpan},
		{"OS Drive Name", &ld.MappingFile},
		{"Creation Date", &ld.CreateTime},
		{"SCSI NAA Id", &ld.ScsiNaaId},
	}
	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine("=")
		if !hasMore {
			break
		}
		// if v == "" && pdRegexp.MatchString(k) {
		// 	parts := strings.Fields(k)
		// 	pd := "/c" + lc.cid + "/e" + strings.ReplaceAll(parts[0], ":", "/s")
		// 	ld.pds = append(ld.pds, pd)
		// 	continue
		// }

		if v == "" {
			continue
		}

		for _, f := range fields {
			if f.key == k {
				*f.value = v
			}
		}
	}

	lc.ctrl.LogicalDrives = append(lc.ctrl.LogicalDrives, ld)

	return scanner.Err()
}

func (lc *lsiController) collectCtrlEnclosure(ctx context.Context, ens []*enclosureList) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(ens) == 0 {
		return nil
	}

	if lc.ctrl.Backplanes == nil {
		lc.ctrl.Backplanes = make([]*enclosure, 0, len(ens))
	}

	errs := make([]error, 0, len(ens))

	for _, en := range ens {
		if err := lc.parseCtrlEnclosure(ctx, en); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (lc *lsiController) parseCtrlEnclosure(ctx context.Context, en *enclosureList) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	enl := &enclosure{
		ID:                 strconv.Itoa(en.EID),
		State:              en.State,
		Slots:              strconv.Itoa(en.Slots),
		Location:           fmt.Sprintf("/c%s/e%d", lc.cid, en.EID),
		PhysicalDriveCount: strconv.Itoa(en.PD),
	}

	data, err := storcliCmd(ctx, enl.Location, "show", "all")
	if err != nil {
		return err
	}

	fields := []field{
		{"Connerctor Name", &enl.ConnectorName},
		{"Enclosure Type", &enl.EnclosureType},
		{"Enclosure Serial Number", &enl.EnclosureSerialNumber},
		{"Device Type", &enl.DeviceType},
		{"Vendor Identification", &enl.Vendor},
		{"Product Identification", &enl.ProductIdentification},
		{"Product Revision Level", &enl.ProductRevisionLevel},
	}

	scanner := utils.NewScanner(bytes.NewReader(data))
	for {
		k, v, hasMore := scanner.ParseLine("=")
		if !hasMore {
			break
		}
		if v == "" {
			continue
		}

		for _, f := range fields {
			if f.key == k {
				*f.value = v
			}
		}
	}

	lc.ctrl.Backplanes = append(lc.ctrl.Backplanes, enl)

	return scanner.Err()
}

func (lc *lsiController) collectCtrlBattery(ctx context.Context, bbus []*cacheVaultInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(bbus) == 0 {
		return nil
	}

	if lc.ctrl.Battery == nil {
		lc.ctrl.Battery = make([]*battery, 0, len(bbus))
	}

	for _, bbu := range bbus {
		cachevault := &battery{
			Model:         bbu.Model,
			State:         bbu.State,
			Temperature:   bbu.Temp,
			RetentionTime: bbu.RetentionTime,
			Mode:          bbu.Mode,
			MfgDate:       bbu.MfgDate,
		}

		lc.ctrl.Battery = append(lc.ctrl.Battery, cachevault)
	}

	return nil
}

func (lc *lsiController) associate() {
	// for _, ld := range lc.ctrl.LogicalDrives {
	// 	for _, disk := range ld.pds {
	// 		for _, pd := range lc.ctrl.PhysicalDrives {
	// 			if disk == pd.Location {
	// 				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
	// 				break
	// 			}
	// 		}
	// 	}
	// }

	for _, ld := range lc.ctrl.LogicalDrives {
		for _, pd := range lc.ctrl.PhysicalDrives {
			if ld.DG == pd.DG {
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
		}
	}
}
