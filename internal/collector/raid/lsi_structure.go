package raid

type showAllJSON struct {
	Controllers []*showAllControllersJSON `json:"controllers"`
}

type showAllControllersJSON struct {
	//	CommandStatus *CommandStatus `json:"Command Status"`
	ResponseData *showAllResponseData `json:"Response Data"`
}

// type CommandStatus struct {
// 	CLIVersion      string `json:"CLI Version"`
// 	OperatingSystem string `json:"Operating System"`
// 	Controller      int    `json:"Controller"`
// 	Status          string `json:"Status"`
// 	Description     string `json:"Description"`
// }

type showAllResponseData struct {
	Basics       *basics            `json:"Basics"`
	Version      *driverVersion     `json:"Version"`
	Bus          *bus               `json:"Bus"`
	Status       *status            `json:"Status"`
	Adapter      *adapterOperations `json:"Supported Adapter Operations"`
	HwCfg        *hwcfg             `json:"HwCfg"`
	Default      *defaults          `json:"Defaults"`
	Capabilities *capabilities      `json:"Capabilities"`
}

type basics struct {
	Model      string `json:"Model"`
	SN         string `json:"Serial Number"`
	CTD        string `json:"Current Controller Date/Time"`
	CSD        string `json:"Current System Date/Time"`
	SAS        string `json:"SAS Address"`
	PCI        string `json:"PCI Address"`
	MfgDate    string `json:"Manufacture Date"`
	ReworkDate string `json:"Rework Date"`
	Revision   string `json:"Revision No"`
}

type driverVersion struct {
	DriverName     string `json:"Driver Name"`
	DriverVer      string `json:"Driver Version"`
	FirmwareVer    string `json:"Firmware Version"`
	FirmwarePackge string `json:"Firmware Package Build"`
	BiosVersion    string `json:"Bios Version"`
}

type bus struct {
	HostInterface   string `json:"Host Interface"`
	DeviceInterface string `json:"Device Interface"`
}

type status struct {
	ControllerStatus string `json:"Controller Status"`
	MemoryCeErr      int    `json:"Memory Correctable Errors"`
	MemoryUeErr      int    `json:"Memory Uncorrectable Errors"`
}

type adapterOperations struct {
	RebuildRate         string `json:"Rebuild Rate"`
	CCRate              string `json:"CC Rate"`
	ReconstructRate     string `json:"Reconstruct Rate"`
	PatrolReadRate      string `json:"Patrol Read Rate"`
	BBU                 string `json:"BBU"`
	ForeignConfigImport string `json:"Foreign Config Import"`
	SupportJBOD         string `json:"Support JBOD"`
}

type hwcfg struct {
	ChipRevision        string `json:"ChipRevision"`
	BatteryFRU          string `json:"BatteryFRU"`
	FrontEndPortCount   int    `json:"Front End Port Count"`
	BackendPortCount    int    `json:"Backend Port Count"`
	BBU                 string `json:"BBU"`
	NVRAMSize           string `json:"NVRAM Size"`
	FlashSize           string `json:"Flash Size"`
	OnBoardMemorySize   string `json:"On Board Memory Size"`
	CacheVaultFlashSize string `json:"CacheVault Flash Size"`
	TPM                 string `json:"TPM"`
}

type defaults struct {
	StripSize       string `json:"Strip Size"`
	WritePolicy     string `json:"Write Policy"`
	ReadPolicy      string `json:"Read Policy"`
	CacheWhenBBUBad string `json:"Cache When BBU Bad"`
	SMARTMode       string `json:"SMART Mode"`
}

type capabilities struct {
	SupportedDrives    string `json:"Supported Drives"`
	RaidLevelSupported string `json:"RAID Level Supported"`
	EnableJBOD         string `json:"Enable JBOD"`
	MinStripSize       string `json:"Min Strip Size"`
	MaxStripSize       string `json:"Max Strip Size"`
}

type vdList struct {
	DGVD    string `json:"DG/VD"`
	Level   string `json:"TYPE"`
	State   string `json:"State"`
	Access  string `json:"Access"`
	Consist string `json:"Consist"`
	Cache   string `json:"Cache"`
	Cac     string `json:"Cac"`
	SCC     string `json:"sCC"`
	Size    string `json:"Size"`
}

type pdList struct {
	EIDSlt string `json:"EID:Slt"`
	DID    int    `json:"DID"`
	State  string `json:"State"`
	DG     any    `json:"DG"`
	Size   string `json:"Size"`
	Intf   string `json:"Intf"`
	Med    string `json:"Med"`
	SED    string `json:"SED"`
	PI     string `json:"PI"`
	SeSz   string `json:"SeSz"`
	Model  string `json:"Model"`
	Sp     string `json:"Sp"`
	Type   string `json:"Type"`
}

type enclosureList struct {
	EID    int    `json:"EID"`
	State  string `json:"State"`
	Slots  int    `json:"Slots"`
	PD     int    `json:"PD"`
	PS     int    `json:"PS"`
	Fans   int    `json:"Fans"`
	Port   string `json:"Port#"`
	PortID string `json:"Port ID"`
}

type cacheVaultInfo struct {
	Model         string `json:"Model"`
	State         string `json:"State"`
	RetentionTime string `json:"RetentionTime"`
	Temp          string `json:"Temp"`
	Mode          string `json:"Mode"`
	MfgDate       string `json:"MfgDate"`
}

type showJSON struct {
	Controllers []*showControllersJSON `json:"Controllers"`
}

type showControllersJSON struct {
	ResponseData *showResponseData `json:"Response Data"`
}

type showResponseData struct {
	VirtualDrives  int               `json:"Virtual Drives"`
	VDList         []*vdList         `json:"VD List"`
	PhysicalDrives int               `json:"Physical Drives"`
	PDList         []*pdList         `json:"PD List"`
	Enclosures     int               `json:"Enclosures"`
	EnclosureList  []*enclosureList  `json:"Enclosure List,"`
	CacheVaultInfo []*cacheVaultInfo `json:"Cachevault_Info"`
}
