package memory

type Memory struct {
	PhysicalMemorySize    string               `json:"physical_memory__size,omitempty" name:"Physical Memory" output:"both" color:"defaultGreen"`
	Maxslots              string               `json:"max_slots,omitempty" name:"Slot Max" output:"both"`
	UsedSlots             string               `json:"used_slots,omitempty" name:"Slot Used" output:"both"`
	MemTotal              string               `json:"memory_total,omitempty" name:"System Memory" output:"both"`
	MemFree               string               `json:"memory_free,omitempty" name:"Memory Free" output:"both"`
	MemAvailable          string               `json:"memory_available,omitempty" name:"Memory Avaliable" output:"both"`
	SwapCached            string               `json:"swap_cached,omitempty"`
	SwapTotal             string               `json:"swap_total,omitempty" name:"Swap" output:"both"`
	SwapFree              string               `json:"swap_free,omitempty"`
	Buffer                string               `json:"buffer,omitempty" name:"Buffer" output:"both"`
	Cached                string               `json:"cached,omitempty" name:"Cached" output:"both"`
	Slab                  string               `json:"slab,omitempty"`
	SReclaimable          string               `json:"s_reclaimable,omitempty"`
	SUnreclaim            string               `json:"s_unreclaim,omitempty"`
	KReclaimable          string               `json:"k_reclaimable,omitempty"`
	KernelStack           string               `json:"kernel_stack,omitempty"`
	PageTables            string               `json:"page_tables,omitempty"`
	Dirty                 string               `json:"dirty,omitempty"`
	Writeback             string               `json:"writeback,omitempty"`
	HPagesTotal           string               `json:"huge_page_total,omitempty"`
	HPageSize             string               `json:"huge_page_size,omitempty"`
	HugeTlb               string               `json:"huge_tlb,omitempty"`
	Diagnose              string               `json:"diagnose,omitempty" name:"Diagnose" output:"both"`
	DiagnoseDetail        string               `json:"diagnose_detail,omitempty" name:"Diagnose Detail" output:"both"`
	EdacSlots             string               `json:"slots,omitempty"`
	EdacMemorySize        string               `json:"edac_memory_size,omitempty"`
	PhysicalMemoryEntries []*SmbiosMemoryEntry `json:"physical_memory_entries,omitempty"`
	EdacMemoryEntries     []*EdacMemoryEntry   `json:"edac_memory_entries,omitempty"`
}

type SmbiosMemoryEntry struct {
	Size              string `json:"size,omitempty"`
	DeviceType        string `json:"device_type,omitempty"`
	SerialNumber      string `json:"serial_number,omitempty"`
	Manufacturer      string `json:"manufacturer,omitempty"`
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

type EdacMemoryEntry struct {
	Size                string `json:"size,omitempty"`
	DeviceType          string `json:"device_type,omitempty"`
	SerialNumber        string `json:"serial_number,omitempty"`
	Manufacturer        string `json:"manufacturer,omitempty"`
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
