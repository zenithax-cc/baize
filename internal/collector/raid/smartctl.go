package raid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
)

type BaseSmartInfo struct {
	ModelName       string `json:"model_name"`
	SN              string `json:"serial_number"`
	RotationRate    int    `json:"rotation_rate"`
	FirmwareVersion string `json:"firmware_version"`
	Device          struct {
		Protocol string `json:"protocol"`
	} `json:"device"`
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`
	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	FormFactor struct {
		Name string `json:"name"`
	} `json:"form_factor"`
	UserCapacity struct {
		Bytes float64 `json:"bytes"`
	} `json:"user_capacity"`
}

type SasSmartInfo struct {
	BaseSmartInfo
	Revision        string `json:"revision"`
	GrownDefectList int    `json:"scsi_grown_defect_list"`
	ErrorCounterLog struct {
		Read struct {
			TotalUncorrectedErrors int `json:"total_uncorrected_errors"`
		} `json:"read"`
		Write struct {
			TotalUncorrectedErrors int `json:"total_uncorrected_errors"`
		} `json:"write"`
		Verify struct {
			TotalUncorrectedErrors int `json:"total_uncorrected_errors"`
		} `json:"verify"`
	} `json:"scsi_error_counter_log"`
}

type AtaSmartInfo struct {
	BaseSmartInfo
	AtaSmartAttributes struct {
		Table []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Value      int    `json:"value"`
			Worst      int    `json:"worst"`
			Thresh     int    `json:"thresh"`
			WhenFailed string `json:"when_failed"`
			Flags      struct {
				Value         int    `json:"value"`
				String        string `json:"string"`
				PreFailure    bool   `json:"pre_failure"`
				UpdatedOnline bool   `json:"updated_online"`
				ErrorRate     bool   `json:"error_rate"`
				EventCount    bool   `json:"event_count"`
				AutoKeep      bool   `json:"auto_keep"`
			} `json:"flags"`
			Raw struct {
				Value  int    `json:"value"`
				String string `json:"string"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
}

type NVMeSmartInfo struct {
	BaseSmartInfo
	Capacity        float64 `json:"nvme_total_capacity"`
	NvmeSmartHealth struct {
		CriticalWarning         int `json:"critical_warning"`
		AvailableSpare          int `json:"available_spare"`
		AvailableSpareThreshold int `json:"available_spare_threshold"`
		PercentageUsed          int `json:"percentage_used"`
		DataUnitRead            int `json:"data_unit_read"`
		DataUnitWritten         int `json:"data_unit_written"`
		HostReads               int `json:"host_reads"`
		HostWrites              int `json:"host_writes"`
		ControllerBusyTime      int `json:"controller_busy_time"`
		UnsafeShutdowns         int `json:"unsafe_shutdowns"`
		MediaErrors             int `json:"media_errors"`
		NumErrLogEntries        int `json:"num_err_log_entries"`
		WarningTempTime         int `json:"warning_temperature_time"`
		CriticalCompTime        int `json:"critical_compliance_time"`
	} `json:"nvme_smart_health_information_log"`
}

const (
	sasProtocol  = "SAS"
	scsiProtocol = "SCSI"
	ataProtocol  = "ATA"
	sataProtocol = "SATA"
	nvmeProtocol = "NVMe"

	ssd = "Solid State Device"
)

var (
	blockDevice = "sda"
	blkOnce     sync.Once
	smartctlMap = map[string]struct {
		cmd          string
		needDeviceID bool
	}{
		"lsi":     {`%s /dev/bus/%s -d megaraid,%s -a -j | grep -v ^$`, true},
		"hpe":     {`%s /dev/%s -d cciss,%s -a -j | grep -v ^$`, true},
		"adaptec": {`%s /dev/%s -d aacraid,%s -a -j | grep -v ^$`, true},
		"nvme":    {`%s %s -d nvme -a -j | grep -v ^$`, false},
		"vroc":    {`%s %s -a -j | grep -v ^$`, false},
	}
)

func (pd *physicalDrive) getSmartctlData(vendor, ctrNum, did string) error {
	smart, ok := smartctlMap[vendor]
	if !ok {
		return fmt.Errorf("not supported SMART type: %s", vendor)
	}

	blk := pd.MappingFile
	if blk == "" {
		blk = findBlockDevice()
	}

	smartctlCmd := fmt.Sprintf(smart.cmd, smartctl, blk)
	if smart.needDeviceID {
		if pd.DeviceId == "" {
			return fmt.Errorf("device id is empty")
		}
		if ctrNum == "" {
			smartctlCmd = fmt.Sprintf(smart.cmd, smartctl, blk, did)
		} else {
			smartctlCmd = fmt.Sprintf(smart.cmd, smartctl, ctrNum, did)
		}
	}

	output, err := utils.Run.Command("bash", "-c", smartctlCmd)
	if err != nil {
		return fmt.Errorf("get smartctl data failed: %w", err)
	}

	if err := pd.parseSmartctlData(output); err != nil {
		return fmt.Errorf("parse smartctl data failed: %w", err)
	}

	return nil
}

type populater interface {
	populate(res *physicalDrive) error
}

func (pd *physicalDrive) parseSmartctlData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("SMART data is empty")
	}

	var baseInfo BaseSmartInfo
	if err := json.Unmarshal(data, &baseInfo); err != nil {
		return fmt.Errorf("unmarshal BaseSmartInfo error: %w", err)
	}

	var pop populater
	protocol := baseInfo.Device.Protocol
	switch protocol {
	case sasProtocol, scsiProtocol:
		var smart SasSmartInfo
		if err := json.Unmarshal(data, &smart); err != nil {
			return fmt.Errorf("unmarshal SasSmartInfo error: %w", err)
		}
		pop = &smart
	case nvmeProtocol:
		var smart NVMeSmartInfo
		if err := json.Unmarshal(data, &smart); err != nil {
			return fmt.Errorf("unmarshal NVMeSmartInfo error: %w", err)
		}
		pop = &smart
	case ataProtocol, sataProtocol:
		var smart AtaSmartInfo
		if err := json.Unmarshal(data, &smart); err != nil {
			return fmt.Errorf("unmarshal AtaSmartInfo error: %w", err)
		}
		pop = &smart
	default:
		return fmt.Errorf("unsupported device protocol: %s", protocol)
	}
	return pop.populate(pd)
}

func (b *BaseSmartInfo) populate(res *physicalDrive) error {
	res.Model = b.ModelName
	res.SN = b.SN
	res.SmartStatus = strconv.FormatBool(b.SmartStatus.Passed)
	res.Temperature = strconv.Itoa(b.Temperature.Current)
	res.PowerOnTime = strconv.Itoa(b.PowerOnTime.Hours)

	if b.FormFactor.Name != "" {
		res.FormFactor = b.FormFactor.Name
	}

	if b.RotationRate == 0 {
		res.RotationRate = ssd
	} else {
		res.RotationRate = strconv.Itoa(b.RotationRate) + " RPM"
	}

	if b.UserCapacity.Bytes != 0 {
		cap, err := utils.ConvertUnit(b.UserCapacity.Bytes, "B", false)
		if err != nil {
			return err
		}
		res.Capacity = cap
	}

	return nil
}

func (s *SasSmartInfo) populate(res *physicalDrive) error {
	if err := s.BaseSmartInfo.populate(res); err != nil {
		return err
	}
	res.Firmware = s.Revision
	res.Interface = "SAS"
	res.SmartAttribute = map[string]string{
		"grown_defect_list": strconv.Itoa(s.GrownDefectList),
		"read_uce_errors":   strconv.Itoa(s.ErrorCounterLog.Read.TotalUncorrectedErrors),
		"write_uce_errors":  strconv.Itoa(s.ErrorCounterLog.Write.TotalUncorrectedErrors),
		"verify_uce_errors": strconv.Itoa(s.ErrorCounterLog.Verify.TotalUncorrectedErrors),
	}
	return nil
}

func (n *NVMeSmartInfo) populate(res *physicalDrive) error {
	if err := n.BaseSmartInfo.populate(res); err != nil {
		return err
	}
	res.SmartAttribute = n.NvmeSmartHealth
	res.Interface = "NVMe"
	cap, err := utils.ConvertUnit(n.Capacity, "B", false)
	if err != nil {
		return err
	}
	res.Capacity = cap
	return nil
}

func (a *AtaSmartInfo) populate(res *physicalDrive) error {
	if err := a.BaseSmartInfo.populate(res); err != nil {
		return err
	}
	res.Interface = "SATA"
	res.SmartAttribute = a.AtaSmartAttributes.Table
	return nil
}

func findBlockDevice() string {
	blkOnce.Do(func() {
		output, err := utils.Run.Command("bash", "-c", "lsblk -d -o NAME | grep -v NAME")
		if err == nil && len(output) > 0 {
			if !bytes.ContainsAny(output, blockDevice) {
				blockDevice = string(output[0])
			}
		}
	})

	return blockDevice
}
