package smbios

type TableType uint8

const (
	BIOS TableType = iota
	System
	BaseBoard
	Chassis
	Processor
	Controller
	Module
	Cache
	PortConnector
	SystemSlots
	OnBoardDevices
	OEMStrings
	SystemConfigurationOptions
	BIOSLanguage
	GroupAssociations
	SystemEventLog
	PhysicalMemoryArray
	MemoryDevice
	Bit32MemoryError
	MemoryArrayMappedAddress
	MemoryDeviceMappedAddress
	BuiltInPointingDevice
	PortableBattery
	SystemReset
	HardwareSecurity
	SystemPowerControls
	VoltageProbe
	CoolingDevice
	TemperatureProbe
	ElectricalCurrentProbe
	OutOfBandRemoteAccess
	BootIntegrityServices
	SystemBoot
	Bit64MemoryError
	ManagementDevice
	ManagementDeviceComponent
	ManagementDeviceThresholdData
	MemoryChannel
	IPMIDevice
	PowerSupply
	AdditionalInformation
	OnBoardDevicesExtendedInformation
	ManagementControllerHostInterface
	TPMDevice            /*43*/
	Inactive   TableType = 126
	EndOfTable TableType = 127
)
