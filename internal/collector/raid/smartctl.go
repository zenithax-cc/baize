package raid

import (
	"encoding/json"
	"fmt"
	"strings"

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

	suffixCmd        = " -a -j | grep -v ^$"
	writeCacheSuffix = " -g wcache | grep -i cache"
	readCacheSuffix  = " -g rcache | grep -i cache"

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
		"megaraid": {format: "%s /dev/bus/%s -d megaraid,%s %s", argCount: 4, useCtrlID: true},
		"cciss":    {format: "%s %s -d cciss,%s %s", argCount: 4, useCtrlID: true},
		"aacraid":  {format: "%s %s -d aacraid,%s %s", argCount: 4, useCtrlID: true},
		"nvme":     {format: "%s %s -d nvme %s", argCount: 3},
		"jbod":     {format: "%s %s %s", argCount: 3},
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

func buildCommands(cmdTpl cmdTemplate, cfg SMARTConfig) (string, string) {
	args := make([]string, 0, cmdTpl.argCount)
	switch {
	case cmdTpl.useCtrlID:
		args = []string{smartctlPath, cfg.ControllerID, cfg.DeviceID}
	case cmdTpl.useBlockDev:
		args = []string{smartctlPath, cfg.BlockDevice, cfg.DeviceID}
	default:
		args = []string{smartctlPath, cfg.DeviceID}
	}

	smartctlCmd := fmt.Sprintf(cmdTpl.format, append(args, suffixCmd))
	cacheCmd := fmt.Sprintf(cmdTpl.format, append(args, ""))

	return smartctlCmd, cacheCmd
}

func (pd *physicalDrive) collectSMARTData(cfg SMARTConfig) error {
	cmdTpl, ok := cmdTemplates[cfg.Option]
	if !ok {
		return fmt.Errorf("not supported SMART type: %s", cfg.Option)
	}

	smartctlCmd, cacheCmd := buildCommands(cmdTpl, cfg)

	output := execute.ShellCommand(smartctlCmd)
	if output.Err != nil {
		return fmt.Errorf("running smartctl failed: %w", output.Err)
	}

	errs := make([]error, 0, 2)

	if err := pd.parseSMARTData(output.Stdout); err != nil {
		errs = append(errs, err)
	}

	if err := pd.getWriteAndReadCache(cacheCmd); err != nil {
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
