// Package cpu provides functionality for collecting CPU hardware information.
package cpu

// CPU holds comprehensive information about a physical CPU socket,
// collected from lscpu, SMBIOS (dmidecode), turbostat, and kernel hwmon.
type CPU struct {
	// ModelName is the human-readable CPU model string (e.g., "Intel(R) Xeon(R) ...").
	ModelName string `json:"model_name,omitempty" name:"Model" output:"both" color:"defaultGreen"`
	// VendorID is the normalized CPU vendor name (e.g., "Intel", "AMD", "ARM").
	VendorID     string `json:"vendor_id,omitempty" name:"Vendor" output:"both"`
	Architecture string `json:"architecture,omitempty" name:"Architecture" output:"both"`
	// Sockets is the total number of physical CPU sockets detected by lscpu.
	Sockets        string `json:"sockets,omitempty" name:"Socket(s)" output:"both"`
	CoresPerSocket string `json:"cores_per_socket,omitempty" name:"Cores Per Socket" output:"both"`
	ThreadsPerCore string `json:"threads_per_core,omitempty" name:"Threads Per Core" output:"both"`
	// HyperThreading indicates the HT/SMT support and enable state.
	HyperThreading string `json:"hyper_threading,omitempty" name:"Hyper Threading" output:"both"`
	CPUOpMode      string `json:"cpu_op_mode,omitempty"`
	AddressSizes   string `json:"address_sizes,omitempty"`
	ByteOrder      string `json:"byte_order,omitempty"`
	// CPUs is the total logical CPU count as reported by lscpu.
	CPUs       string `json:"cpus,omitempty"`
	OnlineCPUs string `json:"online_cpus,omitempty"`
	CPUFamily  string `json:"cpu_family,omitempty"`
	CPUModel   string `json:"cpu_model,omitempty"`
	Stepping   string `json:"stepping,omitempty"`
	BogoMIPS   string `json:"bogomips,omitempty"`
	// Virtualization indicates supported virtualization extensions (e.g., VT-x, AMD-V).
	Virtualization string `json:"virtualization,omitempty"`
	// Cache sizes as reported by lscpu.
	L1dCache string `json:"l1d_cache,omitempty"`
	L1iCache string `json:"l1i_cache,omitempty"`
	L2Cache  string `json:"l2_cache,omitempty"`
	L3Cache  string `json:"l3_cache,omitempty"`
	// PowerState reflects the active CPU frequency scaling governor mode.
	PowerState   string `json:"power_state,omitempty" name:"Power State" output:"both" color:"powerGreen"`
	BasedFreqMHz string `json:"based_freq_mhz,omitempty" name:"Frequency" output:"both"`
	MaxFreqMHz   string `json:"max_freq_mhz,omitempty" name:"Core Frequency Max" output:"both"`
	MinFreqMHz   string `json:"min_freq_mhz,omitempty" name:"Core Frequency Min" output:"both"`
	// TemperatureCelsius is the package-level temperature in Celsius.
	TemperatureCelsius string `json:"temperature_celsius,omitempty" name:"Temperature" output:"both"`
	// Watt is the CPU package power consumption reported by turbostat.
	Watt           string `json:"watt,omitempty" name:"Watt" output:"both"`
	Diagnose       string `json:"diagnose,omitempty" name:"Diagnose" color:"Diagnose" output:"both"`
	DiagnoseDetail string `json:"diagnose_detail,omitempty" name:"Diagnose Detail" output:"both" color:"Diagnose"`
	// Flags lists the CPU feature flags as reported by lscpu.
	Flags []string `json:"flags,omitempty"`
	// CPUEntries contains per-socket detailed data sourced from SMBIOS type-4 tables.
	CPUEntries []*SMBIOSCPUEntry `json:"cpu_entries,omitempty" name:"CPU Entry" output:"detail"`
	// threads holds per-logical-thread turbostat data; not exported in JSON.
	threads []*ThreadEntry
}

// SMBIOSCPUEntry represents per-socket CPU information decoded from
// SMBIOS (dmidecode) Type 4 - Processor Information tables.
type SMBIOSCPUEntry struct {
	// SocketDesignation is the motherboard-printed socket label (e.g., "CPU1", "P0").
	SocketDesignation string   `json:"socket_designation,omitempty" name:"Socket Designation"`
	ProcessorType     string   `json:"processor_type,omitempty"`
	Family            string   `json:"family,omitempty"`
	Manufacturer      string   `json:"manufacturer,omitempty"`
	Version           string   `json:"version,omitempty"`
	ExternalClock     string   `json:"external_clock,omitempty"`
	CurrentSpeed      string   `json:"current_speed,omitempty"`
	Status            string   `json:"status,omitempty"`
	Voltage           string   `json:"voltage,omitempty"`
	CoreCount         string   `json:"core_count,omitempty"`
	CoreEnabled       string   `json:"core_enabled,omitempty"`
	ThreadCount       string   `json:"threads_count,omitempty"`
	Characteristics   []string `json:"characteristics,omitempty"`
	// ThreadEntries holds per-logical-thread data associated with this socket.
	ThreadEntries []*ThreadEntry `json:"thread_entries,omitempty"`
}

// ThreadEntry stores per-logical-CPU thread data collected from turbostat output.
type ThreadEntry struct {
	// ProcessorID is the logical processor index as reported by the OS.
	ProcessorID string `json:"processor_id,omitempty"`
	// CoreID is the physical core identifier within the socket.
	CoreID string `json:"core_id,omitempty"`
	// PhysicalID is the socket (package) identifier.
	PhysicalID    string `json:"physical_id,omitempty"`
	CoreFrequency string `json:"core_frequency,omitempty"`
	Temperature   string `json:"temperature,omitempty"`
}
