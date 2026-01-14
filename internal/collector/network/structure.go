package network

import (
	"github.com/zenithax-cc/baize/internal/collector/pci"
)

// Core Types

// Network represents complete network configuration including physical,
// virtual, and bonded interfaces. It provides indexed access for O(1) lookups.
type Network struct {
	NetInterfaces  []NetInterface  `json:"net_interfaces,omitzero"`
	PhyInterfaces  []PhyInterface  `json:"phy_interfaces,omitzero"`
	BondInterfaces []BondInterface `json:"bond_interfaces,omitzero"`

	// Indexes for O(1) lookup - not serialized to JSON
	netInterfaceIdx  map[string]*NetInterface  `json:"-"`
	phyInterfaceIdx  map[string]*PhyInterface  `json:"-"`
	bondInterfaceIdx map[string]*BondInterface `json:"-"`
}

// NetInterface represents a network interface from /sys/class/net.
// Includes both physical and virtual interfaces.
type NetInterface struct {
	DeviceName      string        `json:"device_name,omitzero"`
	MACAddress      string        `json:"mac_address,omitzero"`
	Driver          string        `json:"driver,omitzero"`
	DriverVersion   string        `json:"driver_version,omitzero"`
	FirmwareVersion string        `json:"firmware_version,omitzero"`
	Status          string        `json:"status,omitzero"`
	Speed           string        `json:"speed,omitzero"` // Speed in Mbps (numeric for calculations)
	Duplex          string        `json:"duplex,omitzero"`
	MTU             string        `json:"mtu,omitzero"` // MTU as numeric value
	Port            string        `json:"port,omitzero"`
	LinkDetected    string        `json:"link_detected,omitzero"` // Boolean for clarity
	IPv4            []IPv4Address `json:"ipv4,omitzero"`
}

type IPv4Address struct {
	Address   string `json:"address,omitzero"`
	Netmask   string `json:"netmask,omitzero"`
	Gateway   string `json:"gateway,omitzero"`
	PrefixLen string `json:"prefix_length,omitzero"`
}

// PhyInterface represents physical interface details including
// hardware configuration and upstream switch information.
type PhyInterface struct {
	DeviceName string     `json:"device_name,omitzero"` // Added for indexing
	RingBuffer RingBuffer `json:"ring_buffer,omitzero"`
	Channel    Channel    `json:"channel,omitzero"`
	LLDP       LLDP       `json:"lldp,omitzero"`
	PCI        pci.PCI    `json:"pci,omitzero"`
}

// RingBuffer represents NIC ring buffer configuration.
// Using uint32 for numeric values enables calculations and comparisons.
type RingBuffer struct {
	CurrentRX string `json:"current_rx,omitzero"`
	CurrentTX string `json:"current_tx,omitzero"`
	MaxRX     string `json:"max_rx,omitzero"`
	MaxTX     string `json:"max_tx,omitzero"`
}

// Channel represents NIC channel/queue configuration.
type Channel struct {
	MaxRX           string `json:"max_rx,omitzero"`
	MaxTX           string `json:"max_tx,omitzero"`
	MaxCombined     string `json:"max_combined,omitzero"`
	CurrentRX       string `json:"current_rx,omitzero"`
	CurrentTX       string `json:"current_tx,omitzero"`
	CurrentCombined string `json:"current_combined,omitzero"`
}

// LLDP represents Link Layer Discovery Protocol information
// from upstream ToR (Top of Rack) switch.
type LLDP struct {
	Interface       string `json:"interface,omitzero"`
	ToRAddress      string `json:"tor_mac,omitzero"`
	ToRName         string `json:"tor_name,omitzero"`
	ToRDesc         string `json:"tor_desc,omitzero"`
	PortName        string `json:"port_name,omitzero"`
	PortAggregation string `json:"port_aggregation,omitzero"`
	ManagementIP    string `json:"management_ip,omitzero"`
	VLAN            string `json:"vlan,omitzero"` // VLAN ID: 1-4094
	PPVID           string `json:"ppvid,omitzero"`
	PPVIDSupport    string `json:"ppvid_support,omitzero"`
	PPVIDEnabled    string `json:"ppvid_enabled,omitzero"`
}

// BondInterface represents a Linux bonding interface configuration.
type BondInterface struct {
	BondName           string           `json:"bond_name,omitzero"`
	BondMode           string           `json:"bond_mode,omitzero"`
	TransmitHashPolicy string           `json:"transmit_hash_policy,omitzero"` // Fixed: lowercase 't'
	MIIStatus          string           `json:"mii_status,omitzero"`
	MIIPollingInterval string           `json:"mii_polling_interval,omitzero"` // Milliseconds
	LACPRate           string           `json:"lacp_rate,omitzero"`
	MACAddress         string           `json:"mac_address,omitzero"`
	AggregatorID       string           `json:"aggregator_id,omitzero"`
	NumberOfPorts      string           `json:"number_of_ports,omitzero"`
	Diagnose           string           `json:"diagnose,omitzero"`
	DiagnoseDetail     string           `json:"diagnose_detail,omitzero"`
	SlaveInterfaces    []SlaveInterface `json:"slave_interfaces,omitzero"`

	// Index for O(1) slave lookup
	slaveIdx map[string]*SlaveInterface `json:"-"`
}

// DiagnoseStatus represents bond health diagnosis result
type DiagnoseStatus uint8

const (
	DiagnoseOK DiagnoseStatus = iota
	DiagnoseWarning
	DiagnoseError
	DiagnoseUnknown
)

// String returns human-readable diagnosis status
func (d DiagnoseStatus) String() string {
	switch d {
	case DiagnoseOK:
		return "ok"
	case DiagnoseWarning:
		return "warning"
	case DiagnoseError:
		return "error"
	default:
		return "unknown"
	}
}

// SlaveInterface represents a bond slave (member) interface.
type SlaveInterface struct {
	SlaveName        string `json:"slave_name,omitzero"`
	MIIStatus        string `json:"mii_status,omitzero"`
	State            string `json:"state,omitzero"`
	LinkFailureCount string `json:"link_failure_count,omitzero"`
	QueueID          string `json:"queue_id,omitzero"`
	AggregatorID     string `json:"aggregator_id,omitzero"`
}
