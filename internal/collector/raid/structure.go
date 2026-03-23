// Package raid provides data structures for RAID controllers, enclosures,
// batteries, logical drives, physical drives, and NVMe devices.
package raid

import (
	"github.com/zenithax-cc/baize/internal/collector/pci"
)

// Controllers is the top-level container for all discovered storage controllers
// (vendor RAID cards) and directly-attached NVMe drives.
type Controllers struct {
	Controller []*controller `json:"controller,omitempty" name:"Controller"`
	NVMe       []*nvme       `json:"nvme,omitempty" name:"NVMe"`
}

// controller holds all information for a single RAID controller card,
// including firmware versions, drive/RAID statistics, and sub-component lists.
type controller struct {
	ID             string `json:"controller_id,omitempty" name:"Controller ID"` // Controller identifier
	ProductName    string `json:"product_name,omitempty" name:"Product"`        // Product model name
	CacheSize      string `json:"cache_size,omitempty" name:"Cache Size"`       // Onboard cache size
	SerialNumber   string `json:"serial_number,omitempty"`                      // Controller serial number
	SasAddress     string `json:"sas_address,omitempty"`                        // SAS address of the controller
	ControllerTime string `json:"controller_time,omitempty"`                    // Current controller date/time

	Firmware     string `json:"firmware_version,omitempty"` // Firmware version
	BiosVersion  string `json:"bios_version,omitempty"`     // BIOS version
	FwVersion    string `json:"fw_version,omitempty"`       // Firmware package build version
	ChipRevision string `json:"chip_revision,omitempty"`    // Chip revision number

	CurrentPersonality string `json:"current_personality,omitempty"` // Current operating mode (RAID/HBA/JBOD)
	ControllerStatus   string `json:"controller_status,omitempty"`   // Current controller health status

	NumberOfRaid string `json:"number_of_raid,omitempty" name:"Number Of Raid"` // Total number of logical drives
	FailedRaid   string `json:"failed_raid,omitempty"`                          // Number of failed logical drives
	DegradedRaid string `json:"degraded_raid,omitempty"`                        // Number of degraded logical drives
	NumberOfDisk string `json:"number_of_disk,omitempty"`                       // Total number of physical drives
	FailedDisk   string `json:"failed_disk,omitempty"`                          // Number of failed physical drives
	CriticalDisk string `json:"critical_disk,omitempty"`                        // Number of drives with critical errors

	MemoryCorrectableErrors   string `json:"memory_correctable_errors,omitempty"`   // Controller cache correctable error count
	MemoryUncorrectableErrors string `json:"memory_uncorrectable_errors,omitempty"` // Controller cache uncorrectable error count

	FrontEndPortCount string `json:"front_end_port_count,omitempty"` // Number of host-side (front-end) ports
	BackendPortCount  string `json:"backend_port_count,omitempty"`   // Number of device-side (back-end) ports
	NumberOfBackplane string `json:"number_of_backplane,omitempty"`  // Number of connected backplanes/enclosures
	HostInterface     string `json:"host_interface,omitempty"`       // Host-side interface type (e.g., PCIe)
	DeviceInterface   string `json:"device_interface,omitempty"`     // Device-side interface type (e.g., SAS, SATA)

	NVRAMSize string `json:"nvram_size,omitempty"` // NVRAM (non-volatile RAM) size
	FlashSize string `json:"flash_size,omitempty"` // Onboard flash memory size

	SupportedDrives     string `json:"supported_drives,omitempty"`      // Supported drive interface types
	RaidLevelSupported  string `json:"raid_level_supported,omitempty"`  // Supported RAID levels
	SurpportedJBOD      string `json:"supports_jbod,omitempty"`         // Whether JBOD mode is supported
	EnableJBOD          string `json:"enable_jbod,omitempty"`           // Whether JBOD mode is currently enabled
	ForeignConfigImport string `json:"foreign_config_import,omitempty"` // Whether foreign configuration import is supported

	Diagnose       string   `json:"diagnose,omitempty"`        // Overall health diagnosis result
	DiagnoseDetail string   `json:"diagnose_detail,omitempty"` // Detailed diagnosis message
	PCIe           *pci.PCI `json:"pcie_info,omitempty"`       // Associated PCIe device information

	Backplanes     []*enclosure     `json:"backplanes,omitempty" name:"Enclosure"`           // Connected enclosures/backplanes
	Battery        []*battery       `json:"battery,omitempty" name:"Battery"`                // Battery/cache vault units
	LogicalDrives  []*logicalDrive  `json:"logical_drives,omitempty" name:"Logical Drive"`   // Configured logical drives (virtual disks)
	PhysicalDrives []*physicalDrive `json:"physical_drives,omitempty" name:"Physical Drive"` // Physical drives attached to controller
}

// enclosure represents a disk backplane or JBOD enclosure managed by the controller.
type enclosure struct {
	Location              string `json:"location,omitempty" name:"Location"` // Physical location description
	ID                    string `json:"id,omitempty" name:"ID"`             // Enclosure identifier (EID)
	State                 string `json:"state,omitempty" name:"State"`       // Enclosure health state
	Slots                 string `json:"slots,omitempty"`                    // Total number of drive slots
	PhysicalDriveCount    string `json:"physical_drive_count,omitempty"`     // Number of drives currently inserted
	ConnectorName         string `json:"connector_name,omitempty"`           // Connector/port name
	EnclosureType         string `json:"enclosure_type,omitempty"`           // Enclosure type (e.g., SES, SGPIO)
	EnclosureSerialNumber string `json:"enclosure_serial_number,omitempty"`  // Enclosure serial number
	DeviceType            string `json:"device_type,omitempty"`              // Device type string
	Vendor                string `json:"vendor,omitempty"`                   // Enclosure vendor
	ProductIdentification string `json:"product_identification,omitempty"`   // Product identification string
	ProductRevisionLevel  string `json:"product_revision_level,omitempty"`   // Product firmware revision level
}

// battery represents a RAID controller battery backup unit (BBU) or CacheVault module.
type battery struct {
	Model         string `json:"model,omitempty" name:"Model"`             // Battery model
	State         string `json:"state,omitempty" name:"State"`             // Battery health state
	Temperature   string `json:"temperature,omitempty" name:"Temperature"` // Battery temperature
	RetentionTime string `json:"retention_time,omitempty"`                 // Data retention time (capacitor/cache)
	Mode          string `json:"mode,omitempty"`                           // Operating mode
	MfgDate       string `json:"mfg_date,omitempty"`                       // Manufacturing date
}

// logicalDrive represents a virtual disk (VD) configured on a RAID controller,
// backed by one or more physical drives.
type logicalDrive struct {
	Location              string           `json:"location,omitempty" name:"Location"`              // Human-readable location (e.g., "Cx/Dy")
	VD                    string           `json:"vd,omitempty"`                                    // Virtual drive index
	DG                    string           `json:"dg,omitempty"`                                    // Drive group identifier
	Type                  string           `json:"raid_level,omitempty"`                            // RAID level (e.g., "RAID 5")
	SpanDepth             string           `json:"span_depth,omitempty"`                            // Number of spans in the drive group
	Capacity              string           `json:"capacity,omitempty"`                              // Total logical drive capacity
	State                 string           `json:"state,omitempty"`                                 // Current state (Optl, Dgrd, Pdgd, etc.)
	Access                string           `json:"access,omitempty"`                                // Read/write access state
	Consist               string           `json:"consistent,omitempty"`                            // Data consistency state
	Cache                 string           `json:"current_cache_policy,omitempty"`                  // Current cache policy (WT/WB/NORA/RA)
	StripSize             string           `json:"strip_size,omitempty"`                            // Strip size per physical drive
	NumberOfBlocks        string           `json:"number_of_blocks,omitempty"`                      // Total number of logical blocks
	NumberOfDrivesPerSpan string           `json:"number_of_drives_per_span,omitempty"`             // Number of drives per span
	NumberOfDrives        string           `json:"number_of_drives,omitempty"`                      // Total number of physical drives
	MappingFile           string           `json:"mapping_file,omitempty"`                          // OS block device path (e.g., /dev/sda)
	CreateTime            string           `json:"create_time,omitempty"`                           // Creation timestamp
	ScsiNaaId             string           `json:"scsi_naa_id,omitempty"`                           // SCSI NAA identifier
	PhysicalDrives        []*physicalDrive `json:"physical_drives,omitempty" name:"Physical Drive"` // Physical drives composing this VD
	pds                   []string         // Internal list of physical drive identifiers for association
}

// physicalDrive represents a single physical disk drive (HDD, SSD, or SAS drive)
// attached to a RAID controller or directly to the system.
type physicalDrive struct {
	// Location and identification
	Location    string `json:"location,omitempty" name:"Location"` // Physical location (e.g., enclosure:slot)
	EnclosureId string `json:"enclosure_id,omitempty"`             // Enclosure identifier
	SlotId      string `json:"slot_id,omitempty"`                  // Slot number within the enclosure
	DeviceId    string `json:"device_id,omitempty"`                // Controller-assigned device ID
	DG          string `json:"drive_group,omitempty"`              // Drive group this drive belongs to
	DeviceSpeed string `json:"device_speed,omitempty"`             // Negotiated device speed
	LinkSpeed   string `json:"link_speed,omitempty"`               // Physical link speed

	// Drive state and rebuild information
	State                 string `json:"state,omitempty"`                    // Current drive state (Onln, Offln, Rbld, etc.)
	RebuildInfo           string `json:"rebuild_info,omitempty"`             // Rebuild progress information
	MediaWearoutIndicator string `json:"media_wearout_indicator,omitempty"`  // SSD wear-level indicator (%)
	AvailableReservdSpace string `json:"available_reserved_space,omitempty"` // Available reserved flash space (SSD)

	// Error counters and health status
	ShieldCounter          string `json:"shield_counter,omitempty"`           // Shield diagnostics counter
	OtherErrorCount        string `json:"other_error_count,omitempty"`        // Non-media error count
	MediaErrorCount        string `json:"media_error_count,omitempty"`        // Media (physical) error count
	PredictiveFailureCount string `json:"predictive_failure_count,omitempty"` // Predictive failure event count
	SmartAlert             string `json:"smart_alert,omitempty"`              // SMART alert status

	// Mapping and diagnosis
	MappingFile    string `json:"mapping_file,omitempty"`    // OS block device mapping (e.g., /dev/sdb)
	Diagnose       string `json:"diagnose,omitempty"`        // Drive health diagnosis result
	DiagnoseDetail string `json:"diagnose_detail,omitempty"` // Detailed diagnosis message

	// Drive identity and characteristics (populated from SMART data)
	Vendor             string `json:"vendor,omitempty"`
	Product            string `json:"product,omitempty"`
	ModelName          string `json:"model_name,omitempty"`
	SN                 string `json:"sn,omitempty"`
	WWN                string `json:"wwn,omitempty"`
	FirmwareVersion    string `json:"firmware_version,omitempty"`
	MediaType          string `json:"media_type,omitempty"`
	ProtocolType       string `json:"protocol_type,omitempty"`
	ProtocolVersion    string `json:"protocol_version,omitempty"`
	Capacity           string `json:"capacity,omitempty"`
	LogicalSectorSize  string `json:"logical_sector_size,omitempty"`
	PhysicalSectorSize string `json:"physical_sector_size,omitempty"`
	RotationRate       string `json:"rotation_rate,omitempty"`
	FormFactor         string `json:"form_factor,omitempty"`
	PowerOnTime        string `json:"power_on_time,omitempty"`
	Temperature        string `json:"temperature,omitempty"`
	WriteCache         string `json:"write_cache,omitempty"`
	ReadCache          string `json:"read_cache,omitempty"`
	SMARTStatus        bool   `json:"smart_status,omitempty"`
	SMARTAttributes    any    `json:"smart_attributes,omitempty"`
}

// nvme extends physicalDrive with NVMe-specific fields such as namespaces and PCIe info.
type nvme struct {
	physicalDrive
	Namespaces []string `json:"namespaces,omitempty"` // List of NVMe namespace device paths
	PCIe       *pci.PCI `json:"pcie,omitempty"`       // PCIe device information for this NVMe
}
