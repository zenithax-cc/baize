package raid

type BasicInfo struct {
	ModelName         string      `json:"model_name"`
	SerialNumber      string      `json:"serial_number"`
	FirmwareVersion   string      `json:"firmware_version"`
	LogicalBlockSize  int         `json:"logical_block_size"`
	PhysicalBlockSize int         `json:"physical_block_size"`
	RotationRate      int         `json:"rotation_rate"`
	Device            Device      `json:"device"`
	WWN               WWN         `json:"wwn"`
	UserCapacity      Capacity    `json:"user_capacity"`
	FormFactor        FormFactor  `json:"form_factor"`
	PowerOnTime       PowerOnTime `json:"power_on_time"`
	Temperature       Temperature `json:"temperature"`
	SmartStatus       SmartStatus `json:"smart_status"`
}

type Device struct {
	Name     string `json:"name"`
	InfoName string `json:"info_name"`
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
}

type PowerOnTime struct {
	Hours int `json:"hours"`
}

type Temperature struct {
	Current int `json:"current"`
}

type SmartStatus struct {
	Passed bool `json:"passed"`
}

type WWN struct {
	NAA int `json:"naa"`
	OUI int `json:"oui"`
	ID  int `json:"id"`
}

type Capacity struct {
	Blocks int `json:"blocks"`
	Bytes  int `json:"bytes"`
}

type FormFactor struct {
	Name string `json:"name"`
}

type AtaSmartInfo struct {
	BasicInfo
	SataVersion        SataVersion        `json:"sata_version"`
	AtaSmartAttributes AtaSmartAttributes `json:"ata_smart_attributes"`
}

type SataVersion struct {
	Value  int    `json:"value"`
	String string `json:"string"`
}

type InterfaceSpeed struct {
	Max     SpeedInfo `json:"max"`
	Current SpeedInfo `json:"current"`
}

type SpeedInfo struct {
	Value          int    `json:"sata_value"`
	String         string `json:"string"`
	UnitsPerSecond int    `json:"units_per_second"`
	BitsPerUnit    int    `json:"bits_per_unit"`
}

type AtaSmartAttributes struct {
	Revision string              `json:"revision"`
	Table    []AtaSmartAttribute `json:"table"`
}

type AtaSmartAttribute struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Worst      int    `json:"worst"`
	Thresh     int    `json:"thresh"`
	WhenFailed string `json:"when_failed"`
	Flags      Flags  `json:"flags"`
	Raw        Raw    `json:"raw"`
}

type Flags struct {
	Value         int    `json:"value"`
	String        string `json:"string"`
	Prefailure    bool   `json:"prefailure"`
	UpdatedOnline bool   `json:"updated_online"`
	Performance   bool   `json:"performance"`
	ErrorRate     bool   `json:"error_rate"`
	EventCount    bool   `json:"event_count"`
	AutoKeep      bool   `json:"auto_keep"`
}

type Raw struct {
	Value  int    `json:"value"`
	String string `json:"string"`
}

type SasSmartInfo struct {
	BasicInfo
	Revision            string          `json:"revision"` // 固件版本
	SCSIVersion         string          `json:"scsi_version"`
	SCSIGrownDefectList int             `json:"scsi_grown_defect_list"`
	SCSIErrorCounterLog ErrorCounterLog `json:"scsi_error_counter_log"`
}

type ErrorCounterLog struct {
	Read   ErrorInfo `json:"read"`
	Write  ErrorInfo `json:"write"`
	Verify ErrorInfo `json:"verify"`
}

type ErrorInfo struct {
	ErrorsCorrectedByECCfast         int    `json:"errors_corrected_by_eccfast"`
	ErrorsCorrectedByECCdelayed      int    `json:"errors_corrected_by_eccdelayed"`
	ErrorsCorrectedByRereadsRewrites int    `json:"errors_corrected_by_rereads_rewrites"`
	TotalErrorsCorrected             int    `json:"total_errors_corrected"`
	CorrectionAlgorithmInvocations   int    `json:"correction_algorithm_invocations"`
	GigabytesProcessed               string `json:"gigabytes_processed"`
	TotalUncorrectedErrors           int    `json:"total_uncorrected_errors"`
}

type NVMeSmartInfo struct {
	BasicInfo
	NVMeCapacity            int64           `json:"nvme_total_capacity"`
	NVMeUnallocatedCapacity int64           `json:"nvme_unallocated_capacity"`
	NumberOfNamespaces      int             `json:"nvme_number_of_namespaces"`
	NVMeVersion             NVMeVersion     `json:"nvme_version"`
	NVMeSmartHealthInfo     NVMeSmartHealth `json:"nvme_smart_health_information_log"`
}

type NVMeVersion struct {
	String string `json:"string"`
	Value  int    `json:"value"`
}

type NVMeSmartHealth struct {
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
}
