// Package ipmi provides data structures for IPMI (Intelligent Platform
// Management Interface) information collected via ipmitool.
package ipmi

// IPMI is the top-level container for all IPMI-related hardware data,
// including BMC info, sensors, power supplies, and the system event log.
type IPMI struct {
	// BMC holds baseboard management controller identification information.
	BMC BMC `json:"bmc,omitempty" name:"BMC"`
	// PowerSupplies holds per-PSU status and power readings.
	PowerSupplies []*PowerSupply `json:"power_supplies,omitempty" name:"Power Supply" output:"detail"`
	// Sensors holds per-sensor readings grouped by category.
	Sensors *Sensors `json:"sensors,omitempty" name:"Sensors"`
	// SEL holds the most-recent critical/error entries from the System Event Log.
	SEL []*SELEntry `json:"sel,omitempty" name:"System Event Log" output:"detail"`
	// Diagnose is a human-readable overall health verdict.
	Diagnose string `json:"diagnose,omitempty" name:"Diagnose" output:"both" color:"Diagnose"`
	// DiagnoseDetail provides additional context when Diagnose is not "OK".
	DiagnoseDetail string `json:"diagnose_detail,omitempty" name:"Diagnose Detail" output:"both" color:"Diagnose"`
}

// BMC holds identification and firmware information for the Baseboard
// Management Controller.
type BMC struct {
	// DeviceID is the IPMI device identifier reported by the BMC.
	DeviceID string `json:"device_id,omitempty" name:"Device ID" output:"detail"`
	// DeviceRevision is the hardware revision of the BMC chip.
	DeviceRevision string `json:"device_revision,omitempty" name:"Device Revision" output:"detail"`
	// FirmwareRevision is the firmware version running on the BMC.
	FirmwareRevision string `json:"firmware_revision,omitempty" name:"Firmware Revision" output:"both"`
	// IPMIVersion is the IPMI specification version supported by the BMC.
	IPMIVersion string `json:"ipmi_version,omitempty" name:"IPMI Version" output:"both"`
	// ManufacturerID is the IANA manufacturer ID of the BMC vendor.
	ManufacturerID string `json:"manufacturer_id,omitempty" name:"Manufacturer ID" output:"detail"`
	// ProductID is the vendor-specific product identifier.
	ProductID string `json:"product_id,omitempty" name:"Product ID" output:"detail"`
	// ManagementIP is the IP address of the BMC management interface (LAN channel 1).
	ManagementIP string `json:"management_ip,omitempty" name:"Management IP" output:"both" color:"DefaultGreen"`
	// MACAddress is the MAC address of the BMC management interface.
	MACAddress string `json:"mac_address,omitempty" name:"MAC Address" output:"both"`
	// Subnet is the subnet mask of the BMC management interface.
	Subnet string `json:"subnet,omitempty" name:"Subnet" output:"detail"`
	// Gateway is the default gateway of the BMC management interface.
	Gateway string `json:"gateway,omitempty" name:"Gateway" output:"detail"`
}

// Sensors groups sensor readings by hardware subsystem for clarity.
type Sensors struct {
	// Temperature holds all temperature sensor readings (in °C).
	Temperature []*Sensor `json:"temperature,omitempty" name:"Temperature" output:"detail"`
	// Voltage holds all voltage sensor readings (in V).
	Voltage []*Sensor `json:"voltage,omitempty" name:"Voltage" output:"detail"`
	// Fan holds all fan speed sensor readings (in RPM).
	Fan []*Sensor `json:"fan,omitempty" name:"Fan" output:"detail"`
	// Current holds all current sensor readings (in A).
	Current []*Sensor `json:"current,omitempty" name:"Current" output:"detail"`
	// Other holds any sensors not fitting the above categories.
	Other []*Sensor `json:"other,omitempty" name:"Other" output:"detail"`
}

// Sensor represents a single IPMI sensor reading.
type Sensor struct {
	// Name is the sensor name as reported by ipmitool.
	Name string `json:"name,omitempty" name:"Sensor"`
	// Value is the current sensor reading with its unit (e.g., "42.000 degrees C").
	Value string `json:"value,omitempty" name:"Value" output:"detail"`
	// Status is the sensor threshold status (e.g., "ok", "cr", "nc").
	Status string `json:"status,omitempty" name:"Status" output:"detail" color:"Diagnose"`
	// LowerCritical is the lower critical threshold value.
	LowerCritical string `json:"lower_critical,omitempty"`
	// UpperCritical is the upper critical threshold value.
	UpperCritical string `json:"upper_critical,omitempty"`
}

// PowerSupply represents a single power supply unit (PSU) discovered via IPMI.
type PowerSupply struct {
	// Name is the PSU identifier (e.g., "PS1", "PSU2").
	Name string `json:"name,omitempty" name:"PSU"`
	// Status is the current PSU operational status (e.g., "Presence Detected").
	Status string `json:"status,omitempty" name:"Status" output:"detail"`
	// InputVoltage is the AC input voltage reading.
	InputVoltage string `json:"input_voltage,omitempty" name:"Input Voltage" output:"detail"`
	// OutputWatts is the current DC output power in watts.
	OutputWatts string `json:"output_watts,omitempty" name:"Output Watts" output:"detail"`
	// MaxWatts is the rated maximum output power in watts.
	MaxWatts string `json:"max_watts,omitempty" name:"Max Watts" output:"detail"`
}

// SELEntry represents a single entry from the IPMI System Event Log.
type SELEntry struct {
	// ID is the hexadecimal SEL record ID.
	ID string `json:"id,omitempty" name:"ID"`
	// Timestamp is the event timestamp as reported by the BMC.
	Timestamp string `json:"timestamp,omitempty" name:"Timestamp" output:"detail"`
	// Sensor is the sensor or component that generated the event.
	Sensor string `json:"sensor,omitempty" name:"Sensor" output:"detail"`
	// Event is the human-readable event description.
	Event string `json:"event,omitempty" name:"Event" output:"detail"`
	// Direction indicates whether the event is "Asserted" or "Deasserted".
	Direction string `json:"direction,omitempty" name:"Direction" output:"detail"`
	// Severity classifies the event: "Critical", "Warning", or "Info".
	Severity string `json:"severity,omitempty" name:"Severity" output:"detail" color:"Diagnose"`
}
