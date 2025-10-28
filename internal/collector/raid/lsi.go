package raid

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
)

type lsiController struct {
	*controller
}

var (
	lsiPDRegex = regexp.MustCompile(`^(\d+):(\d+)`)

	scannerPool = sync.Pool{
		New: func() any {
			return bufio.NewScanner(bytes.NewReader(nil))
		},
	}
)

type LsiErr struct {
	Operation string
	Details   string
	Err       error
}

func (e *LsiErr) Error() string {
	return fmt.Sprintf("LSI %s failed: %s - %v", e.Operation, e.Details, e.Err)
}

func (e *LsiErr) Unwrap() error {
	return e.Err
}

func lsiHandle(ctx context.Context, ctrNum int, c *controller) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	lsiCtr := &lsiController{controller: c}

	if err := lsiCtr.findController(ctx, ctrNum); err != nil {
		return &LsiErr{Operation: "find controller", Details: c.PCIe.PCIeID, Err: err}
	}

	if err := lsiCtr.loadControllerData(ctx); err != nil {
		return &LsiErr{Operation: "load controller data", Details: c.PCIe.PCIeID, Err: err}
	}

	*c = *lsiCtr.controller
	return nil
}

// findController finds the controller with the given PCIe address.
func (lc *lsiController) findController(ctx context.Context, ctrNum int) error {
	for i := 0; i < ctrNum; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if lc.isControllerAtSlot(i) {
			lc.ID = strconv.Itoa(i)
			return nil
		}
	}

	return fmt.Errorf("LSI controller %s not found", lc.PCIe.PCIeAddr)
}

func (lc *lsiController) isControllerAtSlot(slot int) bool {
	pcieAddr := strings.TrimPrefix(lc.PCIe.PCIeAddr, "00")
	cmd := fmt.Sprintf("%s /c%d show | grep %s", storcliPath, slot, pcieAddr)
	output, err := utils.Run.Command("bash", "-c", cmd)
	return err == nil && len(output) > 0
}

// executeStorcli executes the storcli command and returns the output.
func executeStorcli(cmd []string) ([]byte, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("storcli command is empty")
	}

	output, err := utils.Run.Command(storcliPath, cmd...)

	if err != nil {
		return nil, fmt.Errorf("storcli failed: %w", err)
	}

	return output, nil
}

// loadControllerData loads the controller data from the storcli command.
func (lc *lsiController) loadControllerData(ctx context.Context) error {
	var multiErr utils.MultiError

	// HBA卡获取physical drives信息
	if lc.PCIe.SubClassID == "07" {
		if err := lc.loadBasicInfo(ctx); err != nil {
			multiErr.Add(err)
		}
	}

	if err := lc.loadDetailedInfo(ctx); err != nil {
		multiErr.Add(err)
	}

	return multiErr.Unwrap()
}

// loadBasicInfo loads the basic information of the controller.
func (lc *lsiController) loadBasicInfo(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	output, err := executeStorcli([]string{"/c" + lc.ID, "show", "J"})
	if err != nil {
		return err
	}

	return lc.parseControllerJSON(output)
}

// loadDetailedInfo loads the detailed information of the controller.
func (lc *lsiController) loadDetailedInfo(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	output, err := executeStorcli([]string{"/c" + lc.ID, "show", "all", "J"})
	if err != nil {
		return err
	}

	return lc.parseControllerJSON(output)
}

// parseControllerJSON parses the controller JSON output.
func (lc *lsiController) parseControllerJSON(output []byte) error {
	var lsiCtr StorcliRes
	if err := json.Unmarshal(output, &lsiCtr); err != nil {
		return fmt.Errorf("unmarshal lsi controller json error: %w", err)
	}

	if len(lsiCtr.Controllers) != 1 {
		return fmt.Errorf("expected one controller,but got %d", len(lsiCtr.Controllers))
	}

	return lc.parseControllerData(lsiCtr.Controllers[0].ResponseData)
}

// parseControllerData parses the controller data.
func (lc *lsiController) parseControllerData(data *ResponseData) error {
	var multiErr utils.MultiError

	lc.populateBasicInfo(data)

	if len(data.PDList) > 0 {
		if err := lc.parsePhysicalDriveList(data.PDList); err != nil {
			multiErr.Add(err)
		}
	}

	if len(data.VDList) > 0 {
		for _, vd := range data.VDList {
			if err := lc.parseVirtualDrive(vd); err != nil {
				multiErr.Add(fmt.Errorf("parse virtual drive %s error: %w", vd.DGVD, err))
			}
		}
	}

	lc.processEnclosureList(data.EnclosureList, &multiErr)
	lc.processBatteries(data.CachevaultInfo)

	return multiErr.Unwrap()
}

// populateBasicInfo populates the basic information of the controller.
func (lc *lsiController) populateBasicInfo(data *ResponseData) {

	lc.NumberOfRaid = strconv.Itoa(data.VirtualDrives)
	lc.NumberOfDisk = strconv.Itoa(data.PhysicalDrives)
	lc.NumberOfBackplane = strconv.Itoa(data.Enclosures)

	if data.Basics != nil {
		lc.ProductName = data.Basics.Model
		lc.SerialNumber = data.Basics.SN
		lc.ControllerTime = data.Basics.CTD
		lc.SasAddress = data.Basics.SAS
	}

	if data.Version != nil {
		lc.BiosVersion = data.Version.BiosVersion
		lc.FwVersion = data.Version.FirmwareVer
		lc.Firmware = data.Version.FirmwarePackge
	}

	if data.Bus != nil {
		lc.HostInterface = data.Bus.HostInterface
		lc.DeviceInterface = data.Bus.DeviceInterface
	}

	if data.Status != nil {
		lc.ControllerStatus = data.Status.ControllerStatus
		lc.MemoryCorrectableErrors = strconv.Itoa(data.Status.MemoryCeErr)
		lc.MemoryUncorrectableErrors = strconv.Itoa(data.Status.MemoryUeErr)
	}

	if data.Adapter != nil {
		lc.SurpportedJBOD = data.Adapter.SupportJBOD
		lc.ForeignConfigImport = data.Adapter.ForeignConfigImport
	}

	if data.HwCfg != nil {
		lc.ChipRevision = data.HwCfg.ChipRevision
		lc.FrontEndPortCount = strconv.Itoa(data.HwCfg.FrontEndPortCount)
		lc.BackendPortCount = strconv.Itoa(data.HwCfg.BackendPortCount)
		lc.NVRAMSize = data.HwCfg.NVRAMSize
		lc.FlashSize = data.HwCfg.FlashSize
		lc.CacheSize = data.HwCfg.OnBoardMemorySize
	}

	if data.Capabilities != nil {
		lc.SupportedDrives = data.Capabilities.SupportedDrives
		lc.RaidLevelSupported = data.Capabilities.RaidLevelSupported
		lc.EnableJBOD = data.Capabilities.EnableJBOD
	}
}

// parsePhysicalDriveList parses the physical drive list.
func (lc *lsiController) parsePhysicalDriveList(pdList []*pdList) error {
	var multiErr utils.MultiError

	for _, pd := range pdList {
		drive, err := lc.parsePhysicalDrive(pd)
		if err != nil {
			multiErr.Add(fmt.Errorf("parse physical drive %d error: %w", pd.DID, err))
		}
		lc.PhysicalDrives = append(lc.PhysicalDrives, drive)
		pdc.Set(drive.Location, drive)
	}

	return multiErr.Unwrap()
}

var physicalDriveFieldMap = map[string]func(*physicalDrive, string){
	"Shield Couter":                    func(pd *physicalDrive, val string) { pd.ShieldCounter = val },
	"Media Error Count":                func(pd *physicalDrive, val string) { pd.MediaErrorCount = val },
	"Other Error Count":                func(pd *physicalDrive, val string) { pd.OtherErrorCount = val },
	"Predictive Failure Count":         func(pd *physicalDrive, val string) { pd.PredictiveFailureCount = val },
	"Drive Temperature":                func(pd *physicalDrive, val string) { pd.Temperature = val },
	"S.M.A.R.T alert flagged by drive": func(pd *physicalDrive, val string) { pd.SmartAlert = val },
	"SN":                               func(pd *physicalDrive, val string) { pd.SN = val },
	"WWN":                              func(pd *physicalDrive, val string) { pd.WWN = val },
	"Firmware Revision":                func(pd *physicalDrive, val string) { pd.FirmwareVersion = val },
	"Device Speed":                     func(pd *physicalDrive, val string) { pd.DeviceSpeed = val },
	"Link Speed":                       func(pd *physicalDrive, val string) { pd.LinkSpeed = val },
	"Write Cache":                      func(pd *physicalDrive, val string) { pd.WriteCache = val },
	"Logical Sector Size":              func(pd *physicalDrive, val string) { pd.LogicalSectorSize = val },
	"Physical Sector Size":             func(pd *physicalDrive, val string) { pd.PhysicalSectorSize = val },
}

// parsePhysicalDrive parses the physical drive information.
func (lc *lsiController) parsePhysicalDrive(pd *pdList) (*physicalDrive, error) {
	res := &physicalDrive{
		DeviceId:           strconv.Itoa(pd.DID),
		State:              pd.State,
		Capacity:           pd.Size,
		ProtocolType:       pd.Intf,
		ModelName:          pd.Model,
		PhysicalSectorSize: pd.SeSz,
		DG:                 parseDG(pd.DG),
	}

	if err := lc.parsePhysicalDriveLocation(pd.EIDSlt, res); err != nil {
		return res, err
	}

	var multiErr utils.MultiError

	if err := parsePhysicalDriveDetails(res); err != nil {
		multiErr.Add(err)
	}

	res.MappingFile = GetBlockByWWN(res.WWN)
	if len(res.MappingFile) > 0 {
		if err := res.collectSMARTData(SMARTConfig{Option: "jbod", BlockDevice: res.MappingFile}); err != nil {
			multiErr.Add(err)
		}
	} else {
		if err := res.collectSMARTData(SMARTConfig{Option: "megaraid", ControllerID: lc.ID, DeviceID: res.DeviceId}); err != nil {
			multiErr.Add(err)
		}
	}

	return res, multiErr.Unwrap()
}

// parseDG parses the DG value.
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

// parsePhysicalDriveLocation parses the physical drive location.
func (lc *lsiController) parsePhysicalDriveLocation(eidSlt string, pdInfo *physicalDrive) error {
	parts := strings.Split(eidSlt, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid EID:SLT format: %s", eidSlt)
	}

	pdInfo.EnclosureId = parts[0]
	pdInfo.SlotId = parts[1]
	pdInfo.Location = fmt.Sprintf("/c%s/e%s/s%s", lc.ID, pdInfo.EnclosureId, pdInfo.SlotId)

	return nil
}

// parsePhysicalDriveDetails parses the physical drive details.
func parsePhysicalDriveDetails(pd *physicalDrive) error {
	output, err := executeStorcli([]string{pd.Location, "show", "all"})
	if err != nil {
		return err
	}

	scanner := scannerPool.Get().(*bufio.Scanner)
	defer func() {
		scannerPool.Put(scanner)
	}()

	*scanner = *bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, "=")
		if !find {
			continue
		}
		if fn, ok := physicalDriveFieldMap[key]; ok {
			fn(pd, value)
		}
	}

	return scanner.Err()
}

var virtualDriveFieldMap = map[string]func(vd *logicalDrive, val string){
	"Strip Size":                func(vd *logicalDrive, val string) { vd.StripSize = val },
	"Number of Blocks":          func(vd *logicalDrive, val string) { vd.NumberOfBlocks = val },
	"Number of Drives Per Span": func(vd *logicalDrive, val string) { vd.NumberOfDrivesPerSpan = val },
	"OS Drive Name":             func(vd *logicalDrive, val string) { vd.MappingFile = val },
	"Creation Date":             func(vd *logicalDrive, val string) { vd.CreateTime = val },
	"SCSI NAA Id":               func(vd *logicalDrive, val string) { vd.ScsiNaaId = val },
}

func (lc *lsiController) parseVirtualDrive(vd *vdList) error {
	ld := &logicalDrive{
		Type:     vd.Level,
		State:    vd.State,
		Capacity: vd.Size,
		Consist:  vd.Consist,
		Access:   vd.Access,
		Cache:    vd.Cache,
	}
	if err := lc.parseVirtualDriveLocation(vd.DGVD, ld); err != nil {
		return err
	}

	output, err := executeStorcli([]string{ld.Location, "show", "all"})
	if err != nil {
		return err
	}

	scanner := scannerPool.Get().(*bufio.Scanner)
	defer func() {
		scannerPool.Put(scanner)
	}()

	*scanner = *bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if lsiPDRegex.MatchString(line) {
			pds := strings.Fields(line)
			pdLocation := fmt.Sprintf("/c%s/e%s", lc.ID, strings.ReplaceAll(pds[0], ":", "/s"))
			if pd, ok := pdc.Get(pdLocation); ok {
				pd.MappingFile = ld.MappingFile
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
			continue
		}

		key, value, find := utils.Cut(line, "=")
		if !find {
			continue
		}

		if fn, exists := virtualDriveFieldMap[key]; exists {
			fn(ld, value)
		}
	}

	lc.LogicalDrives = append(lc.LogicalDrives, ld)

	return scanner.Err()
}

// parseVirtualDriveLocation parses the virtual drive location.
func (lc *lsiController) parseVirtualDriveLocation(dgvd string, ld *logicalDrive) error {
	parts := strings.Split(dgvd, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid DG/VD format: %s", dgvd)
	}

	ld.DG = parts[0]
	ld.VD = parts[1]
	ld.Location = fmt.Sprintf("/c%s/v%s", lc.ID, ld.VD)

	return nil
}

func (lc *lsiController) processEnclosureList(enclList []*enclosureList, multiErr *utils.MultiError) {
	for _, encl := range enclList {
		if err := lc.parseBackplane(encl); err != nil {
			multiErr.Add(fmt.Errorf("parse enclosure %d:%w", encl.EID, err))
		}
	}
}

var backplaneFieldMap = map[string]func(*backplane, string){
	"Connector Name":          func(bpl *backplane, val string) { bpl.ConnectorName = val },
	"Enclosure Type":          func(bpl *backplane, val string) { bpl.EnclosureType = val },
	"Enclosure Serial Number": func(bpl *backplane, val string) { bpl.EnclosureSerialNumber = val },
	"Device Type":             func(bpl *backplane, val string) { bpl.DeviceType = val },
	"Vendor Identification":   func(bpl *backplane, val string) { bpl.Vendor = val },
	"Product Identification":  func(bpl *backplane, val string) { bpl.ProductIdentification = val },
	"Product Revision Level":  func(bpl *backplane, val string) { bpl.ProductRevisionLevel = val },
}

func (lc *lsiController) parseBackplane(encl *enclosureList) error {
	bpl := &backplane{
		ID:                 strconv.Itoa(encl.EID),
		State:              encl.State,
		Slots:              strconv.Itoa(encl.Slots),
		Location:           fmt.Sprintf("/c%s/e%d", lc.ID, encl.EID),
		PhysicalDriveCount: strconv.Itoa(encl.PD),
	}

	output, err := executeStorcli([]string{bpl.Location, "show", "all"})
	if err != nil {
		return err
	}

	scanner := scannerPool.Get().(*bufio.Scanner)
	defer func() {
		scannerPool.Put(scanner)
	}()

	scanner = bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, "=")
		if !find {
			continue
		}

		if fn, exists := backplaneFieldMap[key]; exists {
			fn(bpl, value)
		}
	}

	lc.Backplanes = append(lc.Backplanes, bpl)

	return scanner.Err()
}

func (lc *lsiController) processBatteries(batteries []*cacheVaultInfo) {
	if len(batteries) == 0 {
		return
	}

	lc.Battery = make([]*battery, len(batteries))

	for _, cv := range batteries {
		cachevault := &battery{
			Model:         cv.Model,
			State:         cv.State,
			Temperature:   cv.Temp,
			RetentionTime: cv.RetentionTime,
			Mode:          cv.Mode,
			MfgDate:       cv.MfgDate,
		}
		lc.Battery = append(lc.Battery, cachevault)
	}
}
