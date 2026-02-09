package product

type OS struct {
	KernelName    string `json:"kernel_name,omitempty" name:"OS Type"`
	KernelRelease string `json:"kernel_release,omitempty" name:"Kernel Release"`
	KernelVersion string `json:"kernel_version,omitempty"`
	HostName      string `json:"host_name,omitempty"`
	PrettyName    string `json:"pretty_name,omitempty"`
	Releases      string `json:"releases,omitempty"`
	DistrVersion  string `json:"distr_version,omitempty"`
	MinorVersion  string `json:"minor_version,omitempty" name:"Distro Version"`
	IDLike        string `json:"id_like,omitempty"`
	CodeName      string `json:"code_name,omitempty"`
	Distr         string `json:"distr,omitempty" name:"Distro"`
}

type BIOS struct {
	Manufacturer     string `json:"manufacturer,omitempty"`
	Version          string `json:"version,omitempty"`
	SerialNumber     string `json:"serial_number,omitempty"`
	Vendor           string `json:"vendor,omitempty"`
	ReleaseDate      string `json:"release_date,omitempty"`
	ROMSize          string `json:"rom_size,omitempty"`
	BIOSRevision     string `json:"bios_revision,omitempty"`
	FirmwareRevision string `json:"firmware_revision,omitempty"`
}

type System struct {
	Manufacturer string `json:"manufacturer,omitempty" name:"Manufacturer" color:"DefaultGreen"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty" name:"SN" color:"DefaultGreen"`
	ProductName  string `json:"product_name,omitempty" name:"Product Name" color:"DefaultGreen"`
	UUID         string `json:"uuid,omitempty" name:"UUID"`
	WakeupType   string `json:"wake-up_type,omitempty"`
	Family       string `json:"family,omitempty"`
}

type BaseBoard struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	ProductName  string `json:"product_name,omitempty"`
	Type         string `json:"type,omitempty"`
}

type Chassis struct {
	Manufacturer     string `json:"manufacturer,omitempty"`
	Version          string `json:"version,omitempty"`
	SerialNumber     string `json:"serial_number,omitempty"`
	Type             string `json:"type,omitempty"`
	SN               string `json:"sn,omitempty"`
	AssetTag         string `json:"asset_tag,omitempty" name:"Asset Tag" color:"DefaultGreen"`
	BootupState      string `json:"bootup_state,omitempty"`
	PowerSupplyState string `json:"power_supply_state,omitempty"`
	ThermalState     string `json:"thermal_state,omitempty"`
	SecurityStatus   string `json:"security_status,omitempty"`
	Height           string `json:"height,omitempty"`
	NumberOfPower    string `json:"number_of_power_cards,omitempty"`
	SKU              string `json:"sku_number,omitempty"`
}

type Product struct {
	OS        `json:"os" name:"OS"`
	BIOS      `json:"bios" name:"BIOS"`
	System    `json:"system" name:"System"`
	BaseBoard `json:"base_board" name:"Baseboard"`
	Chassis   `json:"chassis" name:"Chassis"`
}
