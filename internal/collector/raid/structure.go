package raid

import "github.com/zenithax-cc/baize/internal/collector/pci"

type Controllers struct {
	Controller []*controller `json:"controller,omitempty"`
	NVMe       []*nvme       `json:"nvme,omitempty"`
}

type controller struct {
	ID             string `json:"controller_id,omitempty"`   // 控制器ID
	ProductName    string `json:"product_name,omitempty"`    // 产品名称
	CacheSize      string `json:"cache_size,omitempty"`      // 缓存大小
	SerialNumber   string `json:"serial_number,omitempty"`   // 序列号
	SasAddress     string `json:"sas_address,omitempty"`     // SAS地址
	ControllerTime string `json:"controller_time,omitempty"` // 控制器当前时间

	Firmware     string `json:"firmware_version,omitempty"` // 固件版本
	BiosVersion  string `json:"bios_version,omitempty"`     // BIOS版本
	FwVersion    string `json:"fw_version,omitempty"`       // FW版本
	ChipRevision string `json:"chip_revision,omitempty"`    // 修订固件版本

	CurrentPersonality string `json:"current_personality,omitempty"` // 当前工作模式
	ControllerStatus   string `json:"controller_status,omitempty"`   // 控制器当前状态

	NumberOfRaid string `json:"number_of_raid,omitempty"` // 逻辑硬盘数量
	FailedRaid   string `json:"failed_raid,omitempty"`    // 失败的逻辑盘数
	DegradedRaid string `json:"degraded_raid,omitempty"`  // 降级的逻辑盘数
	NumberOfDisk string `json:"number_of_disk,omitempty"` // 物理硬盘总数
	FailedDisk   string `json:"failed_disk,omitempty"`    // 失败硬盘数
	CriticalDisk string `json:"critical_disk,omitempty"`  // 出现致命错误硬盘数

	MemoryCorrectableErrors   string `json:"memory_correctable_errors,omitempty"`   // 缓存可纠正错误
	MemoryUncorrectableErrors string `json:"memory_uncorrectable_errors,omitempty"` // 缓存不可纠正错误

	FrontEndPortCount string `json:"front_end_port_count,omitempty"` // 前背板接口数量
	BackendPortCount  string `json:"backend_port_count,omitempty"`   // 后背板接口数量
	NumberOfBackplane string `json:"number_of_backplane,omitempty"`  // 硬盘背板数量
	HostInterface     string `json:"host_interface,omitempty"`       // RAID卡接口
	DeviceInterface   string `json:"device_interface,omitempty"`     // 硬盘接口

	NVRAMSize string `json:"nvram_size,omitempty"` // NVRAM大小
	FlashSize string `json:"flash_size,omitempty"` // Flash大小

	SupportedDrives     string `json:"supported_drives,omitempty"`      // 支持硬盘类型
	RaidLevelSupported  string `json:"raid_level_supported,omitempty"`  // 支持RAID类型
	SurpportedJBOD      string `json:"supports_jbod,omitempty"`         // 支持JBOD
	EnableJBOD          string `json:"enable_jbod,omitempty"`           // JBOD使能状态
	ForeignConfigImport string `json:"foreign_config_import,omitempty"` // 外部配置导入

	Diagnose       string   `json:"diagnose,omitempty"`        // RAID卡健康诊断
	DiagnoseDetail string   `json:"diagnose_detail,omitempty"` // RAID卡诊断详情
	PCIe           *pci.PCI `json:"pcie_info,omitempty"`       // PCIe信息

	Backplanes     []*backplane     `json:"backplanes,omitempty"`      // 背板列表
	Battery        []*battery       `json:"battery,omitempty"`         // 电池信息
	LogicalDrives  []*logicalDrive  `json:"logical_drives,omitempty"`  // 逻辑硬盘列表
	PhysicalDrives []*physicalDrive `json:"physical_drives,omitempty"` // 物理硬盘列表
}

type backplane struct {
	Location              string `json:"location,omitempty"`                // 背板位置
	ID                    string `json:"id,omitempty"`                      // 背板ID
	State                 string `json:"state,omitempty"`                   // 背板状态
	Slots                 string `json:"slots,omitempty"`                   // 背板插槽编号
	PhysicalDriveCount    string `json:"physical_drive_count,omitempty"`    // 背板硬盘总数
	ConnectorName         string `json:"connector_name,omitempty"`          // 背板接口名
	EnclosureType         string `json:"enclosure_type,omitempty"`          // 背板类型
	EnclosureSerialNumber string `json:"enclosure_serial_number,omitempty"` // 背板SN
	DeviceType            string `json:"device_type,omitempty"`             // 背板设备类型
	Vendor                string `json:"vendor,omitempty"`                  // 背板厂商
	ProductIdentification string `json:"product_identification,omitempty"`  // 背板产品标识
	ProductRevisionLevel  string `json:"product_revision_level,omitempty"`  // 产品修订级别
}

type battery struct {
	Model         string `json:"model,omitempty"`          // 型号
	State         string `json:"state,omitempty"`          // 状态
	Temperature   string `json:"temperature,omitempty"`    // 温度
	RetentionTime string `json:"retention_time,omitempty"` // 保留时间
	Mode          string `json:"mode,omitempty"`           // 工作模式
	MfgDate       string `json:"mfg_date,omitempty"`       // 制造日期
}

type logicalDrive struct {
	Location              string           `json:"location,omitempty"`                  // 逻辑硬盘位置
	VD                    string           `json:"vd,omitempty"`                        // 逻辑硬盘ID
	DG                    string           `json:"dg,omitempty"`                        // 逻辑硬盘组标识
	Type                  string           `json:"raid_level,omitempty"`                // RAID级别
	SpanDepth             string           `json:"span_depth,omitempty"`                // 逻辑硬盘深度
	Capacity              string           `json:"capacity,omitempty"`                  // 逻辑硬盘容量
	State                 string           `json:"state,omitempty"`                     // 逻辑硬盘状态
	Access                string           `json:"access,omitempty"`                    // 逻辑硬盘读写状态
	Consist               string           `json:"consistent,omitempty"`                // 逻辑硬盘一致性状态
	Cache                 string           `json:"current_cache_policy,omitempty"`      // 逻辑硬盘缓存策略
	StripSize             string           `json:"strip_size,omitempty"`                // 逻辑硬盘块大小
	NumberOfBlocks        string           `json:"number_of_blocks,omitempty"`          // 逻辑硬盘块数量
	NumberOfDrivesPerSpan string           `json:"number_of_drives_per_span,omitempty"` // 逻辑硬盘每层硬盘数量
	NumberOfDrives        string           `json:"number_of_drives,omitempty"`          // 逻辑硬盘物理硬盘数量
	MappingFile           string           `json:"mapping_file,omitempty"`              // 逻辑硬盘对应系统块设备
	CreateTime            string           `json:"create_time,omitempty"`               // 逻辑硬盘创建时间
	ScsiNaaId             string           `json:"scsi_naa_id,omitempty"`               // 逻辑硬盘SCSI编号
	PhysicalDrives        []*physicalDrive `json:"physical_drives,omitempty"`           // 逻辑盘包含的物理硬盘
}

type physicalDrive struct {
	// 位置和标识信息
	Location    string `json:"location,omitempty"`     // 物理硬盘位置
	EnclosureId string `json:"enclosure_id,omitempty"` // 物理硬盘背板编号
	SlotId      string `json:"slot_id,omitempty"`      // 物理硬盘插槽编号
	DeviceId    string `json:"device_id,omitempty"`    // 物理硬盘设备编号
	DG          string `json:"drive_group,omitempty"`  // 硬盘组

	// 厂商和产品信息
	Vendor    string `json:"vendor,omitempty"`        // 物理硬盘厂商
	Product   string `json:"product,omitempty"`       // 物理硬盘产品名称
	OemVendor string `json:"oem_vendor,omitempty"`    // 物理硬盘OEM厂商
	Model     string `json:"model,omitempty"`         // 物理硬盘Model
	SN        string `json:"serial_number,omitempty"` // 物理硬盘SN
	FruCru    string `json:"fru_cru,omitempty"`       // 物理硬盘FRU信息
	WWN       string `json:"wwn,omitempty"`           // 物理硬盘WWN

	// 容量和规格
	Capacity           string `json:"capacity,omitempty"`             // 物理硬盘容量
	MediumType         string `json:"medium_type,omitempty"`          // 物理硬盘类型
	RotationRate       string `json:"rotation_rate,omitempty"`        // 物理硬盘转速
	FormFactor         string `json:"form_factor,omitempty"`          // 物理硬盘尺寸
	LogicalSectorSize  string `json:"logical_sector_size,omitempty"`  // 物理硬盘逻辑扇区大小
	PhysicalSectorSize string `json:"physical_sector_size,omitempty"` // 物理硬盘物理扇区大小

	// 接口和速度
	Interface   string `json:"interface,omitempty"`    // 物理硬盘接口
	DeviceSpeed string `json:"device_speed,omitempty"` // 物理硬盘设备速度
	LinkSpeed   string `json:"link_speed,omitempty"`   // 物理硬盘链路速度

	// 状态信息
	State                 string `json:"state,omitempty"`                    // 物理硬盘状态
	RebuildInfo           string `json:"rebuild_info,omitempty"`             // 物理硬盘重建信息
	PowerOnTime           string `json:"power_on_time,omitempty"`            // 物理硬盘通电时间
	Temperature           string `json:"temperature,omitempty"`              // 物理硬盘温度
	MediaWearoutIndicator string `json:"media_wearout_indicator,omitempty"`  // SSD磨损值
	AvailableReservdSpace string `json:"available_reserved_space,omitempty"` // 可用的预留闪存数量

	// 缓存配置
	WriteCache string `json:"write_cache,omitempty"` // 物理硬盘写缓存
	ReadCache  string `json:"read_cache,omitempty"`  // 物理硬盘读缓存

	// 错误和健康状态
	ShieldCounter          string `json:"shield_counter,omitempty"`           // 物理硬盘保护计数器
	OtherErrorCount        string `json:"other_error_count,omitempty"`        // 物理硬盘其他错误数
	MediaErrorCount        string `json:"media_error_count,omitempty"`        // 物理硬盘物理媒介错误数
	PredictiveFailureCount string `json:"predictive_failure_count,omitempty"` // 预测失效计数
	SmartStatus            string `json:"smart_status,omitempty"`             // 物理硬盘SMART状态
	SmartAlert             string `json:"smart_alert,omitempty"`              // 物理硬盘SMART警告

	// 其他信息
	MappingFile    string `json:"mapping_file,omitempty"`    // 物理硬盘映射系统块设备名称
	Type           string `json:"type,omitempty"`            // 类型
	Firmware       string `json:"firmware,omitempty"`        // 物理硬盘固件
	Diagnose       string `json:"diagnose,omitempty"`        // 物理硬盘健康分析接口
	DiagnoseDetail string `json:"diagnose_detail,omitempty"` // 物理硬盘健康分析详情

	// SMART属性 - 使用更具体的结构体类型
	SmartAttribute any `json:"smart_attributes,omitempty"` // Smart属性
}

type nvme struct {
	physicalDrive
	Namespaces []string `json:"namespaces,omitempty"` // 命名空间
	PCIe       *pci.PCI `json:"pcie,omitempty"`       // PCIe信息
}
