package memory

type BaseMemoryInfo struct {
	Size         string `json:"size,omitempty"`
	DeviceType   string `json:"device_type,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
}

type MemoryInfo struct {
	MemTotal     string `json:"memory_total,omitempty"`
	MemAvailable string `json:"memory_available,omitempty"`
	MemUsed      string `json:"memory_used,omitempty"`
	SwapTotal    string `json:"swap_total,omitempty"`
	SwapUsed     string `json:"swap_used,omitempty"`
	Buffer       string `json:"buffer,omitempty"`
	Cached       string `json:"cached,omitempty"`
	Slab         string `json:"slab,omitempty"`
	SReclaimable string `json:"s_reclaimable,omitempty"`
	SUnreclaim   string `json:"s_unreclaim,omitempty"`
	KReclaimable string `json:"k_reclaimable,omitempty"`
	KernelStack  string `json:"kernel_stack,omitempty"`
	PageTables   string `json:"page_tables,omitempty"`
	Dirty        string `json:"dirty,omitempty"`
	Writeback    string `json:"writeback,omitempty"`
	HPagesTotal  string `json:"huge_page_total,omitempty"`
	HPageSize    string `json:"huge_page_size,omitempty"`
	HugeTlb      string `json:"huge_tlb,omitempty"`

	PhysicalMemoryTotal string `json:"physical_memory_total,omitempty"`
	MaximumSlots        string `json:"maximum_slot,omitempty"`
	UsedSlots           string `json:"used_slots,omitempty"`
	Diagnose            string `json:"diagnose,omitempty"`
	DiagnoseDetail      string `json:"diagnose_detail,omitempty"`
}

type SmbiosMemory struct {
	BaseMemoryInfo
	TotalWidth        string `json:"total_width,omitempty"`
	DataWidth         string `json:"data_width,omitempty"`
	FormFactor        string `json:"form_factor,omitempty"`
	DeviceLocator     string `json:"device_locator,omitempty"`
	BankLocator       string `json:"bank_locator,omitempty"`
	Type              string `json:"type,omitempty"`
	TypeDetail        string `json:"type_detail,omitempty"`
	Speed             string `json:"speed,omitempty"`
	PartNumber        string `json:"part_number,omitempty"`
	Rank              string `json:"rank,omitempty"`
	ConfiguredSpeed   string `json:"configured_speed,omitempty"`
	ConfiguredVoltage string `json:"configured_voltage,omitempty"`
	Technology        string `json:"technology,omitempty"`
}

type EdacMemory struct {
	BaseMemoryInfo
	CorrectableErrors   string `json:"correctable_errors,omitempty"`
	UncorrectableErrors string `json:"uncorrectable_errors,omitempty"`
	EdacMode            string `json:"edac_mode,omitempty"`
	MemoryLocation      string `json:"memory_location,omitempty"`
	MemoryType          string `json:"memory_type,omitempty"`
	SocketID            string `json:"socket_id,omitempty"`
	MemoryControllerID  string `json:"memory_controller_id,omitempty"`
	ChannelID           string `json:"channel_id,omitempty"`
	DIMMID              string `json:"dimm_id,omitempty"`
}

type Memory struct {
	MemoryInfo
	PhysicalMemoryEntries []*SmbiosMemory `json:"physical_memory_entries,omitempty"`
	EdacMemoryEntries     []*EdacMemory   `json:"edac_memory_entries,omitempty"`
}
