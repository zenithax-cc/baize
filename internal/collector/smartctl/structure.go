package smartctl

type PhysicalDrive struct {
	Vendor             string `json:"vendor,omitempty"`
	Product            string `json:"product,omitempty"`
	ModelName          string `json:"model_name,omitempty"`
	SN                 string `json:"sn,omitempty"`
	WWN                string `json:"wwn,omitempty"`
	FirmwareVersion    string `json:"firmware_version,omitempty"`
	ProtocolType       string `json:"protocol_type,omitempty"`
	ProtocolVersion    string `json:"protocol_version,omitempty"`
	Capacity           string `json:"capacity,omitempty"`
	LogicalSectorSize  int    `json:"logical_sector_size,omitempty"`
	PhysicalSectorSize int    `json:"physical_sector_size,omitempty"`
	RotationRate       string `json:"rotation_rate,omitempty"`
	FormFactor         string `json:"form_factor,omitempty"`
	PowerOnTime        string `json:"power_on_time,omitempty"`
	Temperature        string `json:"temperature,omitempty"`
	WriteCache         string `json:"write_cache,omitempty"`
	ReadCache          string `json:"read_cache,omitempty"`
	SMARTStatus        bool   `json:"smart_status,omitempty"`
	SMARTAttributes    any    `json:"smart_attributes,omitempty"`
}
