package smartctl

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

type protocol string

const (
	smartctlPath = "/usr/sbin/smartctl"

	ProtocolATA  protocol = "ATA"
	ProtocolSATA protocol = "SATA"
	ProtocolSAS  protocol = "SAS"
	ProtocolSCSI protocol = "SCSI"
	ProtocolNVMe protocol = "NVMe"

	ssdMediaType = "Solid State Device"
	hddMediaType = "Hard Disk Device"

	suffixCmd        = " -a -j | grep -v ^$"
	writeCacheSuffix = " -g wcache | grep cahce"
	readCacheSuffix  = " -g rcache | grep cahce"
)

var preMap = map[string]string{
	"megaraid": "%s /dev/bus/%s -d megaraid,%s %s",
	"cciss":    "%s %s -d cciss,%s %s",
	"aacraid":  "%s %s -d aacraid,%s %s",
	"nvme":     "%s %s -d nvme %s",
	"jbod":     "%s %s %s",
}

type SMARTConfig struct {
	Option       string
	ControllerID string
	BlockDevice  string
	DeviceID     string
}

func Collect(cfg SMARTConfig) (*PhysicalDrive, error) {
	cmd, ok := preMap[cfg.Option]
	if !ok {
		return nil, fmt.Errorf("not supported SMART type: %s", cfg.Option)
	}

	var smartctlCmd string
	var cacheCmd string

	switch cfg.Option {
	case "cciss":
		smartctlCmd = fmt.Sprintf(cmd, smartctlPath, cfg.BlockDevice, cfg.DeviceID, suffixCmd)
		cacheCmd = fmt.Sprintf(cmd, smartctlPath, cfg.BlockDevice, cfg.DeviceID)
	case "aacraid":
		smartctlCmd = fmt.Sprintf(cmd, smartctlPath, cfg.BlockDevice, cfg.DeviceID, suffixCmd)
		cacheCmd = fmt.Sprintf(cmd, smartctlPath, cfg.BlockDevice, cfg.DeviceID)
	case "megaraid":
		smartctlCmd = fmt.Sprintf(cmd, smartctlPath, cfg.ControllerID, cfg.DeviceID, suffixCmd)
		cacheCmd = fmt.Sprintf(cmd, smartctlPath, cfg.ControllerID, cfg.DeviceID)
	case "nvme", "jbod":
		smartctlCmd = fmt.Sprintf(cmd, smartctlPath, cfg.DeviceID, suffixCmd)
		cacheCmd = fmt.Sprintf(cmd, smartctlPath, cfg.DeviceID)
	default:
		return nil, fmt.Errorf("not supported SMART type: %s", cfg.Option)
	}

	output, err := utils.Run.Command("bash", "-c", smartctlCmd)
	if err != nil {
		return nil, fmt.Errorf("running smartctl failed: %w", err)
	}

	var pd PhysicalDrive
	var MultiErr utils.MultiError

	if err := pd.parseSMARTData(output); err != nil {
		MultiErr.Add(err)
	}

	if err := pd.getWriteAndReadCache(cacheCmd); err != nil {
		MultiErr.Add(err)
	}

	return &pd, MultiErr.Unwrap()
}

func (pd *PhysicalDrive) parseSMARTData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("SMART data is empty")
	}

	var device Device
	if err := json.Unmarshal(data, &device); err != nil {
		return fmt.Errorf("unmarshal Device error: %w", err)
	}

	switch device.Protocol {
	case string(ProtocolATA), string(ProtocolSATA):
		return pd.parseSMARTDataSATA(data)
	case string(ProtocolSAS), string(ProtocolSCSI):
		return pd.parseSMARTDataSAS(data)
	case string(ProtocolNVMe):
		return pd.parseSMARTDataNVMe(data)
	default:
		return fmt.Errorf("not supported device protocol: %s", device.Protocol)
	}
}

func (pd *PhysicalDrive) parseSMARTDataSATA(data []byte) error {
	var ataInfo AtaSmartInfo
	if err := json.Unmarshal(data, &ataInfo); err != nil {
		return fmt.Errorf("unmarshal AtaSmartInfo error: %w", err)
	}

	ataInfo.parseBaseInfo(pd)

	pd.ProtocolType = "SATA"
	pd.ProtocolVersion = ataInfo.SataVersion.String
	pd.SMARTAttributes = ataInfo.AtaSmartAttributes.Table

	return nil
}

func (pd *PhysicalDrive) parseSMARTDataSAS(data []byte) error {
	var sasInfo SasSmartInfo
	if err := json.Unmarshal(data, &sasInfo); err != nil {
		return fmt.Errorf("unmarshal SasSmartInfo error: %w", err)
	}
	sasInfo.parseBaseInfo(pd)

	pd.ProtocolType = "SAS"
	pd.ProtocolVersion = sasInfo.SCSIVersion
	pd.FirmwareVersion = sasInfo.Revision

	pd.SMARTAttributes = map[string]string{
		"grown_defect_list": strconv.Itoa(sasInfo.SCSIGrownDefectList),
		"read_uce_errors":   strconv.Itoa(sasInfo.SCSIErrorCounterLog.Read.TotalUncorrectedErrors),
		"write_uce_errors":  strconv.Itoa(sasInfo.SCSIErrorCounterLog.Write.TotalUncorrectedErrors),
		"verify_uce_errors": strconv.Itoa(sasInfo.SCSIErrorCounterLog.Verify.TotalUncorrectedErrors),
	}

	return nil
}

func (pd *PhysicalDrive) parseSMARTDataNVMe(data []byte) error {
	var nvmeInfo NVMeSmartInfo
	if err := json.Unmarshal(data, &nvmeInfo); err != nil {
		return fmt.Errorf("unmarshal NVMeSmartInfo error: %w", err)
	}

	nvmeInfo.parseBaseInfo(pd)

	pd.ProtocolType = "NVMe"
	pd.ProtocolVersion = nvmeInfo.NVMeVersion.String
	pd.SMARTAttributes = nvmeInfo.NVMeSmartHealthInfo
	pd.FormFactor = "2.5 inchs"

	cap, _ := utils.ConvertUnit(float64(nvmeInfo.NVMeCapacity), "B", false)
	pd.Capacity = cap

	return nil
}

func (bi *BasicInfo) parseBaseInfo(pd *PhysicalDrive) {
	pd.ModelName = bi.ModelName
	pd.SN = bi.SerialNumber
	pd.SMARTStatus = bi.SmartStatus.Passed
	pd.Temperature = fmt.Sprintf("%d℃", bi.Temperature.Current)
	pd.PowerOnTime = strconv.Itoa(bi.PowerOnTime.Hours)
	pd.FirmwareVersion = bi.FirmwareVersion
	pd.FormFactor = bi.FormFactor.Name
	pd.LogicalSectorSize = bi.LogicalBlockSize
	pd.PhysicalSectorSize = bi.PhysicalBlockSize

	pd.RotationRate = strconv.Itoa(bi.RotationRate) + " RPM"
	if bi.RotationRate == 0 {
		pd.RotationRate = ssdMediaType
	}

	if bi.UserCapacity.Bytes != 0 {
		cap, _ := utils.ConvertUnit(float64(bi.UserCapacity.Bytes), "B", false)
		pd.Capacity = cap
	}

	if bi.WWN.NAA != 0 && bi.WWN.OUI != 0 && bi.WWN.ID != 0 {
		pd.WWN = fmt.Sprintf("%d%d%d", bi.WWN.NAA, bi.WWN.OUI, bi.WWN.ID)
	}

	pd.Vendor, pd.Product = parseModelName(bi.ModelName)
}

type disk struct {
	Manufacturer []map[string]string `json:"manufacturer"`
	Product      []map[string]string `json:"product"`
}

type dev struct {
	Disk disk `json:"disk"`
}

func parseModelName(modelName string) (string, string) {
	retVendor, retProduct := "Unkown", "Unkown"
	strReplace := []string{"IBM-ESXS", "HP", "LENOVO-X", "ATA",
		"-", "_", "SAMSUNG", "INTEL", "SEAGATE", "TOSHIBA", "HGST",
		"Micron", "KIOXIA"}
	for _, i := range strReplace {
		modelName = strings.ReplaceAll(strings.TrimSpace(modelName), i, " ")
	}

	js, err := os.Open("/usr/local/beidou/config/devmap.json")
	if err != nil {
		return retVendor, retProduct
	}
	defer js.Close()

	dev := dev{}
	if err := json.NewDecoder(js).Decode(&dev); err != nil {
		return retVendor, retProduct
	}

	sl := strings.Split(modelName, " ")
	for _, value := range dev.Disk.Manufacturer {
		reg := regexp.MustCompile(value["regular"])
		if reg.MatchString(sl[len(sl)-1]) {
			retVendor = value["stdName"]
			break
		}
	}

	for _, value := range dev.Disk.Product {

		if !strings.HasPrefix(value["stdName"], retVendor) {
			continue
		}

		reg := regexp.MustCompile(value["regular"])
		if reg.MatchString(sl[len(sl)-1]) {
			retProduct = value["stdName"]
			break
		}
	}

	return retVendor, retProduct
}

func (pd *PhysicalDrive) getWriteAndReadCache(cacheCmd string) error {
	var MultiErr utils.MultiError
	woutput, err := utils.Run.Command("bash", "-c", fmt.Sprintf(cacheCmd, writeCacheSuffix))
	MultiErr.Add(err)
	if len(woutput) > 0 && err == nil {
		pd.WriteCache = parseCacheData(woutput)
	}

	routput, err := utils.Run.Command("bash", "-c", fmt.Sprintf(cacheCmd, readCacheSuffix))
	MultiErr.Add(err)
	if len(routput) > 0 && err == nil {
		pd.ReadCache = parseCacheData(routput)
	}

	return MultiErr.Unwrap()
}

func parseCacheData(data []byte) string {
	cacheState := "Unknown"
	parts := strings.SplitN(string(data), ":", 2)
	if len(parts) < 2 {
		return cacheState
	}

	cacheState = strings.TrimSpace(parts[1])

	return cacheState
}
