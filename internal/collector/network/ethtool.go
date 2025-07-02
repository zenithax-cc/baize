package network

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/zenithax-cc/baize/common/utils"
)

const (
	flagChannel = "-l"
	flagRingBuf = "-g"
	flagDriver  = "-i"
	privFlags   = "--show-priv-flags"
	setPriv     = "--set-priv-flags"

	fieldRx           = "RX"
	fieldTx           = "TX"
	fieldCombined     = "Combined"
	fieldSpeed        = "Speed"
	fieldDuplex       = "Duplex"
	fieldType         = "Port"
	fieldLinkDetected = "Link detected"
	fieldDriver       = "driver"
	fieldVersion      = "version"
	fieldFirmware     = "firmware-version"
	fieldBusInfo      = "bus-info"
	fieldCurrMarker   = "Current"

	i40eDriverName    = "i40e"
	kernelReleaseFile = "/proc/sys/kernel/osrelease"
	kernelVersion316  = "3.16"
	disableFwLldpFlag = "disable-fw-lldp"
	lldpStopCmd       = "lldp stop"
	i40eCmdTemplate   = "/sys/kernel/debug/i40e/%s/command"
)

type ethSetting struct {
	Speed        string `json:"speed,omitempty"`
	Duplex       string `json:"duplex,omitempty"`
	Port         string `json:"port,omitempty"`
	LinkDetected string `json:"link_detected,omitempty"`
}

type ethChannel struct {
	MaxRxChan    string `json:"max_rx_channel,omitempty"`
	MaxTxChan    string `json:"max_tx_channel,omitempty"`
	MaxCombined  string `json:"max_combined_channel,omitempty"`
	CurrRxChan   string `json:"current_rx_channel,omitempty"`
	CurrTxChan   string `json:"current_tx_channel,omitempty"`
	CurrCombined string `json:"current_combined_channel,omitempty"`
}

type ethRingBuffer struct {
	MaxRxRing  string `json:"max_rx_ring,omitempty"`
	MaxTxRing  string `json:"max_tx_ring,omitempty"`
	CurrRxRing string `json:"current_rx_ring,omitempty"`
	CurrTxRing string `json:"current_tx_ring,omitempty"`
}

type ethDriver struct {
	DriverName  string `json:"driver_name,omitempty"`
	DriverVer   string `json:"driver_version,omitempty"`
	FirmwareVer string `json:"firmware_version,omitempty"`
	BusInfo     string `json:"bus_info,omitempty"`
}

// getEthSetting returns the ethernet setting of a port
func getEthSetting(port string) (*ethSetting, error) {
	b, err := utils.Run.Command("ethtool", port)
	if err != nil {
		return nil, err
	}

	res := &ethSetting{}
	fieldMap := map[string]*string{
		fieldSpeed:        &res.Speed,
		fieldDuplex:       &res.Duplex,
		fieldType:         &res.Port,
		fieldLinkDetected: &res.LinkDetected,
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if v, ok := fieldMap[key]; ok {
				*v = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("scanning ethtool setting failed: %w", err)
	}
	return res, nil
}

// parseEthOutput parses the output of ethtool command for channel and ring buffer
func parseEthOutput(b []byte, maxMap, currentMap map[string]*string) error {
	isPre := true
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, fieldCurrMarker) {
			isPre = false
			continue
		}

		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			if isPre {
				if v, ok := maxMap[key]; ok {
					*v = value
				}
			} else {
				if v, ok := currentMap[key]; ok {
					*v = value
				}
			}
		}
	}

	return scanner.Err()
}

// getEthSetting returns the ethernet channel of a port
func getEthChannel(p string) (*ethChannel, error) {
	b, err := utils.Run.Command("ethtool", flagChannel, p)
	if err != nil {
		return nil, err
	}

	res := &ethChannel{}
	fieldMax := map[string]*string{
		fieldRx:       &res.MaxRxChan,
		fieldTx:       &res.MaxTxChan,
		fieldCombined: &res.MaxCombined,
	}

	fieldCurr := map[string]*string{
		fieldRx:       &res.CurrRxChan,
		fieldTx:       &res.CurrTxChan,
		fieldCombined: &res.CurrCombined,
	}

	if err := parseEthOutput(b, fieldMax, fieldCurr); err != nil {
		return res, fmt.Errorf("parsing ethtool channel failed: %w", err)
	}

	return res, nil
}

// getEthRingBuffer returns the ethernet ring buffer of a port
func getEthRingBuffer(port string) (*ethRingBuffer, error) {
	b, err := utils.Run.Command("ethtool", flagRingBuf, port)
	if err != nil {
		return nil, err
	}

	res := &ethRingBuffer{}
	fieldMax := map[string]*string{
		fieldRx: &res.MaxRxRing,
		fieldTx: &res.MaxTxRing,
	}

	fieldCurr := map[string]*string{
		fieldRx: &res.CurrRxRing,
		fieldTx: &res.CurrTxRing,
	}

	if err := parseEthOutput(b, fieldMax, fieldCurr); err != nil {
		return res, fmt.Errorf("parsing ethtool ring buffer failed: %w", err)
	}

	return res, nil
}

// getEthDriver returns the ethernet driver of a port
func getEthDriver(p string) (*ethDriver, error) {

	b, err := utils.Run.Command("ethtool", flagDriver, p)
	if err != nil {
		return nil, err
	}

	res := &ethDriver{}
	fieldMap := map[string]*string{
		fieldDriver:   &res.DriverName,
		fieldVersion:  &res.DriverVer,
		fieldFirmware: &res.FirmwareVer,
		fieldBusInfo:  &res.BusInfo,
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			if v, ok := fieldMap[key]; ok {
				*v = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("scanning ethtool driver failed: %w", err)
	}

	return res, nil
}

func optimizeI40e(port string) error {
	fw, err := utils.Run.Command("bash", "-c", fmt.Sprintf("ethtool %s %s | awk -F: '/disable-fw-lldp/{print $2}'", privFlags, port))
	if err != nil {
		return fmt.Errorf("failed to get %s fw-lldp info: %v", port, err)
	}

	if strings.TrimSpace(string(fw)) == "off" {
		kernelRelease, err := utils.ReadOneLineFile(kernelReleaseFile)
		if strings.HasPrefix(kernelRelease, kernelVersion316) && err == nil {
			file := fmt.Sprintf(i40eCmdTemplate, port)
			utils.Run.Command("bash", "-c", fmt.Sprintf("echo %s > %s", lldpStopCmd, file))
		}
		utils.Run.Command("bash", "-c", fmt.Sprintf("ethtool %s %s %s on 2>/dev/null", setPriv, port, disableFwLldpFlag))
	}
	return nil
}
