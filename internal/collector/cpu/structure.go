package cpu

// from lscpu
type SummaryCPU struct {
	Architecture   string   `json:"architecture,omitempty"`
	CPUOpMode      string   `json:"cpu_op_mode,omitempty"`
	AddressSizes   string   `json:"address_sizes,omitempty"`
	ByteOrder      string   `json:"byte_order,omitempty"`
	CPUs           string   `json:"cpus,omitempty"`
	OnlineCPUs     string   `json:"online_cpus,omitempty"`
	VendorID       string   `json:"vendor_id,omitempty"`
	ModelName      string   `json:"model_name,omitempty"`
	CPUFamily      string   `json:"cpu_family,omitempty"`
	CPUModel       string   `json:"cpu_model,omitempty"`
	ThreadsPerCore string   `json:"threads_per_core,omitempty"`
	CoresPerSocket string   `json:"cores_per_socket,omitempty"`
	Sockets        string   `json:"sockets,omitempty"`
	Stepping       string   `json:"stepping,omitempty"`
	BogoMIPS       string   `json:"bogomips,omitempty"`
	Virtualization string   `json:"virtualization,omitempty"`
	HyperThreading string   `json:"hyper_threading,omitempty"`
	L1dCache       string   `json:"l1d_cache,omitempty"`
	L1iCache       string   `json:"l1i_cache,omitempty"`
	L2Cache        string   `json:"l2_cache,omitempty"`
	L3Cache        string   `json:"l3_cache,omitempty"`
	Flags          []string `json:"flags,omitempty"`
}

// from dmidecode
type SMBIOSCPUEntry struct {
	SocketDesignation string         `json:"socket_designation,omitempty"`
	ProcessorType     string         `json:"processor_type,omitempty"`
	Family            string         `json:"family,omitempty"`
	Manufacturer      string         `json:"manufacturer,omitempty"`
	Version           string         `json:"version,omitempty"`
	ExternalClock     string         `json:"external_clock,omitempty"`
	CurrentSpeed      string         `json:"current_speed,omitempty"`
	Status            string         `json:"status,omitempty"`
	Voltage           string         `json:"voltage,omitempty"`
	CoreCount         string         `json:"core_count,omitempty"`
	CoreEnabled       string         `json:"core_enabled,omitempty"`
	ThreadCount       string         `json:"threads_count,omitempty"`
	Characteristics   []string       `json:"characteristics,omitempty"`
	ThreadEntries     []*ThreadEntry `json:"thread_entries,omitempty"`
}

type SMBIOSCPU struct {
	MaxFreqMHz         string            `json:"max_freq_mhz,omitempty"`
	MinFreqMHz         string            `json:"min_freq_mhz,omitempty"`
	BasedFreqMHz       string            `json:"based_freq_mhz,omitempty"`
	TemperatureCelsius string            `json:"temperature_celsius,omitempty"`
	PowerState         string            `json:"power_state,omitempty"`
	Watt               string            `json:"watt,omitempty"`
	CPUEntries         []*SMBIOSCPUEntry `json:"cpu_entries,omitempty"`
}

// from turbostat
type ThreadEntry struct {
	ProcessorID   string `json:"processor_id,omitempty"`
	CoreID        string `json:"core_id,omitempty"`
	PhysicalID    string `json:"physical_id,omitempty"`
	CoreFrequency string `json:"core_frequency,omitempty"`
	Temperature   string `json:"temperature,omitempty"`
}

type CPU struct {
	SummaryCPU
	SMBIOSCPU
	Diagnose       string `json:"diagnose,omitempty"`
	DiagnoseDetail string `json:"diagnose_detail,omitempty"`
}
