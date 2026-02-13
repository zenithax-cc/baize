package cpu

// CPU collects CPU information
type CPU struct {
	ModelName          string            `json:"model_name,omitempty" name:"Model" output:"both" color:"defaultGreen"`
	VendorID           string            `json:"vendor_id,omitempty" name:"Vendor" output:"both"`
	Architecture       string            `json:"architecture,omitempty" name:"Architecture" output:"both"`
	Sockets            string            `json:"sockets,omitempty" name:"Socket(s)" output:"both"`
	CoresPerSocket     string            `json:"cores_per_socket,omitempty" name:"Cores Per Soket" output:"both"`
	ThreadsPerCore     string            `json:"threads_per_core,omitempty" name:"Threads Per Core" output:"both"`
	HyperThreading     string            `json:"hyper_threading,omitempty" name:"Hyper Threading" output:"both"`
	CPUOpMode          string            `json:"cpu_op_mode,omitempty"`
	AddressSizes       string            `json:"address_sizes,omitempty"`
	ByteOrder          string            `json:"byte_order,omitempty"`
	CPUs               string            `json:"cpus,omitempty"`
	OnlineCPUs         string            `json:"online_cpus,omitempty"`
	CPUFamily          string            `json:"cpu_family,omitempty"`
	CPUModel           string            `json:"cpu_model,omitempty"`
	Stepping           string            `json:"stepping,omitempty"`
	BogoMIPS           string            `json:"bogomips,omitempty"`
	Virtualization     string            `json:"virtualization,omitempty"`
	L1dCache           string            `json:"l1d_cache,omitempty"`
	L1iCache           string            `json:"l1i_cache,omitempty"`
	L2Cache            string            `json:"l2_cache,omitempty"`
	L3Cache            string            `json:"l3_cache,omitempty"`
	PowerState         string            `json:"power_state,omitempty" name:"Power State" output:"both" color:"powerGreen"`
	BasedFreqMHz       string            `json:"based_freq_mhz,omitempty" name:"Frequency" output:"both"`
	MaxFreqMHz         string            `json:"max_freq_mhz,omitempty" name:"Core Frequency Max" output:"both"`
	MinFreqMHz         string            `json:"min_freq_mhz,omitempty" name:"Core Frequency Min" output:"both"`
	TemperatureCelsius string            `json:"temperature_celsius,omitempty" name:"Temperature" output:"both"`
	Watt               string            `json:"watt,omitempty" name:"Watt" output:"both"`
	Diagnose           string            `json:"diagnose,omitempty" name:"Diagnose" color:"Diagnose" output:"both"`
	DiagnoseDetail     string            `json:"diagnose_detail,omitempty" name:"Diagnose Detail" output:"both" color:"Diagnose"`
	Flags              []string          `json:"flags,omitempty"`
	CPUEntries         []*SMBIOSCPUEntry `json:"cpu_entries,omitempty" name:"cpu entriy" output:"detail"`
	threads            []*ThreadEntry
}

// from dmidecode
type SMBIOSCPUEntry struct {
	SocketDesignation string         `json:"socket_designation,omitempty" name:"Socket Designation"`
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

// from turbostat
type ThreadEntry struct {
	ProcessorID   string `json:"processor_id,omitempty"`
	CoreID        string `json:"core_id,omitempty"`
	PhysicalID    string `json:"physical_id,omitempty"`
	CoreFrequency string `json:"core_frequency,omitempty"`
	Temperature   string `json:"temperature,omitempty"`
}
