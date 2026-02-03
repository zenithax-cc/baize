package raid

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/pkg/execute"
	"github.com/zenithax-cc/baize/pkg/utils"
)

type pcotocol string

const (
	ProtocolATA  pcotocol = "ATA"
	ProtocolSATA pcotocol = "SATA"
	ProtocolSAS  pcotocol = "SAS"
	ProtocolSCSI pcotocol = "SCSI"
	ProtocolNVMe pcotocol = "NVMe"

	ssdMediaType = "Solid State Device"
	hddMediaType = "Hard Disk Device"

	suffixCmd   = " -a -j | grep -v ^$"
	cacheSuffix = " -g all| grep -v ^$"

	devMapConfigPath = "/usr/local/beidou/config/devmap.json"
	smartctlPath     = "/usr/sbin/smartctl"
)

type cmdTemplate struct {
	format      string
	argCount    int
	useCtrlID   bool
	useBlockDev bool
}

var (
	cmdTemplates = map[string]cmdTemplate{
		"megaraid": {format: "%s /dev/bus/%s -d megaraid,%s ", argCount: 4, useCtrlID: true},
		"cciss":    {format: "%s %s -d cciss,%s ", argCount: 4, useBlockDev: true},
		"aacraid":  {format: "%s %s -d aacraid,%s ", argCount: 4, useCtrlID: true},
		"nvme":     {format: "%s %s -d nvme ", argCount: 3},
		"jbod":     {format: "%s %s ", argCount: 3},
	}

	modelNameReplacer = strings.NewReplacer(
		"IBM-ESXS", " ",
		"HP", " ",
		"LENOVO-X", " ",
		"ATA", " ",
		"-", " ",
		"_", " ",
		"SAMSUNG", " ",
		"INTEL", " ",
		"SEAGATE", " ",
		"TOSHIBA", " ",
		"HGST", " ",
		"Micron", " ",
		"KIOXIA", " ",
	)
)

type SMARTConfig struct {
	Option       string
	ControllerID string
	BlockDevice  string
	DeviceID     string
}

func buildCommands(cmdTpl cmdTemplate, cfg SMARTConfig) string {
	var prefixCmd string
	switch {
	case cmdTpl.useCtrlID:
		prefixCmd = fmt.Sprintf(cmdTpl.format, smartctlPath, cfg.ControllerID, cfg.DeviceID)
	case cmdTpl.useBlockDev:
		prefixCmd = fmt.Sprintf(cmdTpl.format, smartctlPath, cfg.BlockDevice, cfg.DeviceID)
	default:
		prefixCmd = fmt.Sprintf(cmdTpl.format, smartctlPath, cfg.DeviceID, suffixCmd)
	}

	return prefixCmd
}

func (pd *physicalDrive) collectSMARTData(cfg SMARTConfig) error {
	cmdTpl, ok := cmdTemplates[cfg.Option]
	if !ok {
		return fmt.Errorf("not supported SMART type: %s", cfg.Option)
	}

	prefixCmd := buildCommands(cmdTpl, cfg)

	println(prefixCmd)

	output := execute.ShellCommand(prefixCmd + suffixCmd)
	if output.Err != nil {
		return fmt.Errorf("running smartctl failed: %w", output.Err)
	}

	errs := make([]error, 0, 2)

	if err := pd.parseSMARTData(output.Stdout); err != nil {
		errs = append(errs, err)
	}

	if err := pd.getWriteAndReadCache(prefixCmd); err != nil {
		errs = append(errs, err)
	}

	return utils.CombineErrors(errs)
}

var protocolParsers = map[string]func(*physicalDrive, []byte) error{
	string(ProtocolATA):  (*physicalDrive).parseSMARTDataSATA,
	string(ProtocolSATA): (*physicalDrive).parseSMARTDataSATA,
	string(ProtocolSAS):  (*physicalDrive).parseSMARTDataSAS,
	string(ProtocolSCSI): (*physicalDrive).parseSMARTDataSAS,
	string(ProtocolNVMe): (*physicalDrive).parseSMARTDataNVMe,
}

func (pd *physicalDrive) parseSMARTData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("SMART data is empty")
	}

	var device struct {
		Device struct {
			Protocol string `json:"protocol"`
		} `json:"device"`
	}

	if err := json.Unmarshal(data, &device); err != nil {
		return fmt.Errorf("unmarshal Device error: %w", err)
	}

	parser, ok := protocolParsers[device.Device.Protocol]
	if !ok {
		return fmt.Errorf("not supported device protocol: %s", device.Device.Protocol)
	}

	return parser(pd, data)
}

func (pd *physicalDrive) parseSMARTDataSATA(data []byte) error {
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

func (pd *physicalDrive) parseSMARTDataSAS(data []byte) error {
	var sasInfo SasSmartInfo
	if err := json.Unmarshal(data, &sasInfo); err != nil {
		return fmt.Errorf("unmarshal SasSmartInfo error: %w", err)
	}

	sasInfo.parseBaseInfo(pd)
	pd.ProtocolType = "SAS"
	pd.ProtocolVersion = sasInfo.SCSIVersion
	pd.FirmwareVersion = sasInfo.Revision

	pd.SMARTAttributes = map[string]int{
		"grown_defect_list": sasInfo.SCSIGrownDefectList,
		"read_uce_errors":   sasInfo.SCSIErrorCounterLog.Read.TotalUncorrectedErrors,
		"write_uce_errors":  sasInfo.SCSIErrorCounterLog.Write.TotalUncorrectedErrors,
		"verify_uce_errors": sasInfo.SCSIErrorCounterLog.Verify.TotalUncorrectedErrors,
	}

	return nil
}

func (pd *physicalDrive) parseSMARTDataNVMe(data []byte) error {
	var nvmeInfo NVMeSmartInfo
	if err := json.Unmarshal(data, &nvmeInfo); err != nil {
		return fmt.Errorf("unmarshal NVMeSmartInfo error: %w", err)
	}

	nvmeInfo.parseBaseInfo(pd)
	pd.ProtocolType = "NVMe"
	pd.ProtocolVersion = nvmeInfo.NVMeVersion.String
	pd.SMARTAttributes = nvmeInfo.NVMeSmartHealthInfo
	pd.FormFactor = "2.5 inchs"

	pd.Capacity = utils.KGMT(float64(nvmeInfo.NVMeCapacity), false)

	return nil
}

func (bi *BasicInfo) parseBaseInfo(pd *physicalDrive) {
	pd.ModelName = bi.ModelName
	pd.SN = bi.SerialNumber
	pd.SMARTStatus = bi.SmartStatus.Passed
	pd.PowerOnTime = strconv.Itoa(bi.PowerOnTime.Hours)
	pd.FirmwareVersion = bi.FirmwareVersion
	pd.LogicalSectorSize = strconv.Itoa(bi.LogicalBlockSize)
	pd.PhysicalSectorSize = strconv.Itoa(bi.PhysicalBlockSize)

	pd.Temperature = fmt.Sprintf("%d ℃", bi.Temperature.Current)

	if bi.RotationRate == 0 {
		pd.RotationRate = ssdMediaType
	} else {
		pd.RotationRate = fmt.Sprintf("%d RPM", bi.RotationRate)
	}

	if bi.UserCapacity.Bytes != 0 {
		pd.Capacity = utils.KGMT(float64(bi.UserCapacity.Bytes), false)
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

var (
	devMapCache     *dev
	devMapCacheOnce sync.Once
	devMapCacheErr  error

	regexCache   = make(map[string]*regexp.Regexp)
	regexCacheMu sync.RWMutex
)

func loadDevMapConfig() (*dev, error) {
	devMapCacheOnce.Do(func() {
		js, err := os.Open(devMapConfigPath)
		if err != nil {
			devMapCacheErr = fmt.Errorf("open devmap config file failed: %w", err)
			return
		}
		defer js.Close()

		devMapCache = &dev{}
		if err := json.NewDecoder(js).Decode(devMapCache); err != nil {
			devMapCacheErr = fmt.Errorf("decode devmap config file failed: %w", err)
			devMapCache = nil
		}
	})

	return devMapCache, devMapCacheErr
}

func getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
	regexCacheMu.RLock()
	if regex, ok := regexCache[pattern]; ok {
		regexCacheMu.RUnlock()
		return regex, nil
	}
	regexCacheMu.RUnlock()

	regexCacheMu.Lock()
	defer regexCacheMu.Unlock()

	if reg, ok := regexCache[pattern]; ok {
		return reg, nil
	}
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCache[pattern] = reg
	return reg, nil
}

func parseModelName(m string) (string, string) {
	retVendor, retProduct := "Unkown", "Unkown"

	cleanedName := modelNameReplacer.Replace(strings.TrimSpace(m))

	devMap, err := loadDevMapConfig()
	if err != nil || devMap == nil {
		return retVendor, retProduct
	}

	parts := strings.Fields(cleanedName)
	if len(parts) == 0 {
		return retVendor, retProduct
	}

	lastField := parts[len(parts)-1]

	for _, value := range devMap.Disk.Manufacturer {
		reg, err := getOrCompileRegex(value["regular"])
		if err != nil {
			continue
		}
		if reg.MatchString(lastField) {
			retVendor = value["stdName"]
			break
		}
	}

	for _, value := range devMap.Disk.Product {
		if !strings.HasPrefix(value["stdName"], retVendor) {
			continue
		}
		reg, err := getOrCompileRegex(value["regular"])
		if err != nil {
			continue
		}
		if reg.MatchString(lastField) {
			retProduct = value["stdName"]
			break
		}
	}

	return retVendor, retProduct
}

func (pd *physicalDrive) getWriteAndReadCache(cacheCmd string) error {
	output := execute.ShellCommand(cacheCmd + cacheSuffix)
	if output.Err != nil {
		return output.Err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output.Stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "Writeback Cache is":
			pd.WriteCache = value
		case "Write Cache is":
			pd.WriteCache = value
		case "Read Cache is":
			pd.ReadCache = value
		case "Rd look-ahead is":
			pd.ReadCache = value
		}
	}

	return scanner.Err()
}
