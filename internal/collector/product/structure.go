package product

type BaseInfo struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
}

type OS struct {
	KernelName    string `json:"kernel_name,omitempty"`
	KernelRelease string `json:"kernel_release,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
	HostName      string `json:"host_name,omitempty"`
	PrettyName    string `json:"pretty_name,omitempty"`
	Releases      string `json:"releases,omitempty"`
	DistrVersion  string `json:"distr_version,omitempty"`
	MinorVersion  string `json:"minor_version,omitempty"`
	IDLike        string `json:"id_like,omitempty"`
	CodeName      string `json:"code_name,omitempty"`
	Distr         string `json:"distr,omitempty"`
}

type BIOS struct {
	BaseInfo
	Vendor           string `json:"vendor,omitempty"`
	ReleaseDate      string `json:"release_date,omitempty"`
	ROMSize          string `json:"rom_size,omitempty"`
	BIOSRevision     string `json:"bios_revision,omitempty"`
	FirmwareRevision string `json:"firmware_revision,omitempty"`
}

type System struct {
	BaseInfo
	ProductName string `json:"product_name,omitempty"`
	UUID        string `json:"uuid,omitempty"`
	WakeupType  string `json:"wake-up_type,omitempty"`
	Family      string `json:"family,omitempty"`
}

type BaseBoard struct {
	BaseInfo
	ProductName string `json:"product_name,omitempty"`
	Type        string `json:"type,omitempty"`
}

type Chassis struct {
	BaseInfo
	Type             string `json:"type,omitempty"`
	SN               string `json:"sn,omitempty"`
	AssetTag         string `json:"asset_tag,omitempty"`
	BootupState      string `json:"bootup_state,omitempty"`
	PowerSupplyState string `json:"power_supply_state,omitempty"`
	ThermalState     string `json:"thermal_state,omitempty"`
	SecurityStatus   string `json:"security_status,omitempty"`
	Height           string `json:"height,omitempty"`
	NumberOfPower    string `json:"number_of_power_cards,omitempty"`
	SKU              string `json:"sku_number,omitempty"`
}

type Product struct {
	OS        `json:"os,omitempty"`
	BIOS      `json:"bios,omitempty"`
	System    `json:"system,omitempty"`
	BaseBoard `json:"base_board,omitempty"`
	Chassis   `json:"chassis,omitempty"`
}
