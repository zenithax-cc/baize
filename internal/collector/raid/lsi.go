package raid

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
)

type lsiController controller

var lsiPDRegex = regexp.MustCompile(`^(\d+):(\d+)`)

func fromStorcli(ctrNum int, c *controller) error {
	lsiCtr := (*lsiController)(c)
	if err := lsiCtr.checkController(ctrNum); err != nil {
		return err
	}

	err := lsiCtr.getController()
	*c = (controller)(*lsiCtr)

	return err
}

func (lc *lsiController) checkController(ctrNum int) error {
	pcieAddr := strings.TrimPrefix(lc.PCIe.PCIeAddr, "00")
	for i := 0; i < ctrNum; i++ {
		output, err := utils.Run.Command("bash", "-c", fmt.Sprintf("%s /c%d show | grep %s", storcli, i, pcieAddr))
		if err == nil && len(output) > 0 {
			lc.ID = strconv.Itoa(i)
			return nil
		}
	}
	return fmt.Errorf("not found LSI controller %s", lc.PCIe.PCIeAddr)
}

func (lc *lsiController) getController() error {
	output, err := utils.Run.Command(storcli, "/c"+lc.ID, "show", "all", "J")
	if err != nil {
		return fmt.Errorf("storcli failed: %w", err)
	}

	var lsiCtr StorcliRes
	if err := json.Unmarshal(output, &lsiCtr); err != nil {
		return fmt.Errorf("unmarshal lsi controller json error: %w", err)
	}

	if len(lsiCtr.Controllers) != 1 {
		return fmt.Errorf("expected one controller,but got %d", len(lsiCtr.Controllers))
	}

	return lc.parseController(lsiCtr.Controllers[0].ResponseData)
}

func (lc *lsiController) parseController(data *ResponseData) error {
	var errs []error
	lc.populateController(data)

	if len(data.PDList) > 0 {
		var wg sync.WaitGroup
		pdChan := make(chan *physicalDrive, len(data.PDList))
		errChan := make(chan error, len(data.PDList))

		for _, pd := range data.PDList {
			wg.Add(1)
			go func(phyDrive *pdList) {
				defer wg.Done()
				drive, err := lc.parsePhysicalDrive(phyDrive)
				if err != nil {
					errChan <- err
					return
				}
				pdChan <- drive
			}(pd)
		}

		wg.Wait()
		close(pdChan)
		close(errChan)

		for err := range errChan {
			errs = append(errs, err)
		}
		for drive := range pdChan {
			lc.PhysicalDrives = append(lc.PhysicalDrives, drive)
			pdMap[drive.Location] = drive
		}
	}

	if len(data.VDList) > 0 {
		for _, vd := range data.VDList {
			if err := lc.parseVirtualDrive(vd); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(data.EnclosureList) > 0 {
		for _, enclosure := range data.EnclosureList {
			if err := lc.parseBackplane(enclosure); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(data.CachevaultInfo) > 0 {
		for _, cv := range data.CachevaultInfo {
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
	return utils.CombineErrors(errs)
}

func (lc *lsiController) populateController(data *ResponseData) {
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

func (lc *lsiController) parsePhysicalDrive(pd *pdList) (*physicalDrive, error) {
	res := &physicalDrive{
		DeviceId:           strconv.Itoa(pd.DID),
		State:              pd.State,
		Capacity:           pd.Size,
		MediumType:         pd.Med,
		Interface:          pd.Intf,
		Model:              pd.Model,
		PhysicalSectorSize: pd.SeSz,
		Type:               pd.Type,
	}

	res.DG = parseDG(pd.DG)

	eidSlt := strings.Split(pd.EIDSlt, ":")
	if len(eidSlt) != 2 {
		return res, fmt.Errorf("invalid EIDSlt format: %s", pd.EIDSlt)
	}
	res.EnclosureId = eidSlt[0]
	res.SlotId = eidSlt[1]
	res.Location = fmt.Sprintf("/c%s/e%s/s%s", lc.ID, res.EnclosureId, res.SlotId)

	var errs []error
	if err := res.getSmartctlData("lsi", lc.ID, res.DeviceId); err != nil {
		errs = append(errs, err)
	}

	fieldMap := map[string]*string{
		"Shield Couter":                    &res.ShieldCounter,
		"Media Error Count":                &res.MediaErrorCount,
		"Other Error Count":                &res.OtherErrorCount,
		"Predictive Failure Count":         &res.PredictiveFailureCount,
		"Drive Temperature":                &res.Temperature,
		"S.M.A.R.T alert flagged by drive": &res.SmartAlert,
		"SN":                               &res.SN,
		"Manufacturer Id":                  &res.OemVendor,
		"FRU/CRU":                          &res.FruCru,
		"WWN":                              &res.WWN,
		"Firmware Revision":                &res.Firmware,
		"Device Speed":                     &res.DeviceSpeed,
		"Link Speed":                       &res.LinkSpeed,
		"Write Cache":                      &res.WriteCache,
		"Logical Sector Size":              &res.LogicalSectorSize,
		"Physical Sector Size":             &res.PhysicalSectorSize,
	}
	output, err := utils.Run.Command(storcli, res.Location, "show", "all")
	if err != nil {
		return res, fmt.Errorf("storcli %s failed: %w", res.Location, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, "=")
		if !find {
			continue
		}

		if ptr, exists := fieldMap[key]; exists {
			*ptr = value
		}
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error scanning physical drive: %w", err))
	}

	return res, utils.CombineErrors(errs)
}

func parseDG(dg any) string {
	switch v := dg.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	}
	return "Unknown"
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
	dgVD := strings.Split(vd.DGVD, "/")
	if len(dgVD) != 2 {
		return fmt.Errorf("invalid DGVD format: %s", vd.DGVD)
	}
	ld.DG = dgVD[0]
	ld.VD = dgVD[1]
	ld.Location = fmt.Sprintf("/c%s/v%s", lc.ID, ld.VD)

	fieldMap := map[string]*string{
		"Strip Size":                &ld.StripSize,
		"Number of Blocks":          &ld.NumberOfBlocks,
		"Number of Drives Per Span": &ld.NumberOfDrivesPerSpan,
		"OS Drive Name":             &ld.MappingFile,
		"Creation Date":             &ld.CreateTime,
		"SCSI NAA Id":               &ld.ScsiNaaId,
	}

	output, err := utils.Run.Command(storcli, ld.Location, "show", "all")
	if err != nil {
		return fmt.Errorf("storcli %s failed: %w", ld.Location, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if lsiPDRegex.MatchString(line) {
			pds := strings.Fields(line)
			pdLocation := fmt.Sprintf("/c%s/e%s", lc.ID, strings.ReplaceAll(pds[0], ":", "/s"))
			pdMapMutex.Lock()
			if pd, ok := pdMap[pdLocation]; ok {
				ld.PhysicalDrives = append(ld.PhysicalDrives, pd)
			}
			pdMapMutex.Unlock()
			continue
		}

		key, value, find := utils.Cut(line, "=")
		if !find {
			continue
		}

		if ptr, exists := fieldMap[key]; exists {
			*ptr = value
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning virtual drive: %w", err)
	}

	lc.LogicalDrives = append(lc.LogicalDrives, ld)
	return nil
}

func (lc *lsiController) parseBackplane(encl *enclosureList) error {
	bpl := &backplane{
		ID:                 strconv.Itoa(encl.EID),
		State:              encl.State,
		Slots:              strconv.Itoa(encl.Slots),
		Location:           fmt.Sprintf("/c%s/e%d", lc.ID, encl.EID),
		PhysicalDriveCount: strconv.Itoa(encl.PD),
	}

	output, err := utils.Run.Command(storcli, bpl.Location, "show", "all")
	if err != nil {
		return fmt.Errorf("storcli %s failed: %w", bpl.Location, err)
	}

	fieldMap := map[string]*string{
		"Connector Name":          &bpl.ConnectorName,
		"Enclosure Type":          &bpl.EnclosureType,
		"Enclosure Serial Number": &bpl.EnclosureSerialNumber,
		"Device Type":             &bpl.DeviceType,
		"Vendor Identification":   &bpl.Vendor,
		"Product Identification":  &bpl.ProductIdentification,
		"Product Revision Level":  &bpl.ProductRevisionLevel,
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, find := utils.Cut(line, "=")
		if !find {
			continue
		}

		if ptr, exists := fieldMap[key]; exists {
			*ptr = value
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning backplane: %w", err)
	}

	lc.Backplanes = append(lc.Backplanes, bpl)
	return nil
}
