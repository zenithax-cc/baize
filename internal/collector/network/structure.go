package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/zenithax-cc/baize/internal/collector/pci"
)

// Constants and Predefined Values

// Interface status constants
const (
	StatusUp      = "up"
	StatusDown    = "down"
	StatusUnknown = "unknown"
)

// Duplex mode constants
const (
	DuplexFull    = "full"
	DuplexHalf    = "half"
	DuplexUnknown = "unknown"
)

// Bond mode constants
const (
	BondModeBalanceRR    = "balance-rr"
	BondModeActiveBackup = "active-backup"
	BondModeBalanceXOR   = "balance-xor"
	BondModeBroadcast    = "broadcast"
	BondMode8023AD       = "802.3ad"
	BondModeBalanceTLB   = "balance-tlb"
	BondModeBalanceALB   = "balance-alb"
)

// LACP rate constants
const (
	LACPRateSlow = "slow"
	LACPRateFast = "fast"
)

// Validation patterns
var (
	macAddressPattern = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	vlanIDPattern     = regexp.MustCompile(`^\d{1,4}$`)
)

// Predefined errors
var (
	ErrInvalidMACAddress  = errors.New("invalid MAC address format")
	ErrInvalidSpeed       = errors.New("invalid speed value")
	ErrInvalidMTU         = errors.New("invalid MTU value")
	ErrInterfaceNotFound  = errors.New("interface not found")
	ErrInvalidBondMode    = errors.New("invalid bond mode")
	ErrNoSlaveInterfaces  = errors.New("bond has no slave interfaces")
	ErrDuplicateInterface = errors.New("duplicate interface name")
	ErrInvalidVLAN        = errors.New("invalid VLAN ID")
	ErrNegativeValue      = errors.New("value cannot be negative")
	ErrExceedsMaxValue    = errors.New("value exceeds maximum allowed")
)

// Core Types

// Network represents complete network configuration including physical,
// virtual, and bonded interfaces. It provides indexed access for O(1) lookups.
type Network struct {
	NetInterfaces  []NetInterface  `json:"net_interfaces,omitempty"`
	PhyInterfaces  []PhyInterface  `json:"phy_interfaces,omitempty"`
	BondInterfaces []BondInterface `json:"bond_interfaces,omitempty"`

	// Indexes for O(1) lookup - not serialized to JSON
	netInterfaceIdx  map[string]*NetInterface  `json:"-"`
	phyInterfaceIdx  map[string]*PhyInterface  `json:"-"`
	bondInterfaceIdx map[string]*BondInterface `json:"-"`
}

// NetInterface represents a network interface from /sys/class/net.
// Includes both physical and virtual interfaces.
type NetInterface struct {
	DeviceName      string `json:"device_name,omitempty"`
	MACAddress      string `json:"mac_address,omitempty"`
	Driver          string `json:"driver,omitempty"`
	DriverVersion   string `json:"driver_version,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`
	Status          string `json:"status,omitempty"`
	Speed           uint64 `json:"speed,omitempty"` // Speed in Mbps (numeric for calculations)
	Duplex          string `json:"duplex,omitempty"`
	MTU             uint32 `json:"mtu,omitempty"` // MTU as numeric value
	Port            string `json:"port,omitempty"`
	LinkDetected    bool   `json:"link_detected,omitempty"` // Boolean for clarity
}

// PhyInterface represents physical interface details including
// hardware configuration and upstream switch information.
type PhyInterface struct {
	DeviceName string     `json:"device_name,omitempty"` // Added for indexing
	RingBuffer RingBuffer `json:"ring_buffer,omitempty"`
	Channel    Channel    `json:"channel,omitempty"`
	LLDP       LLDP       `json:"lldp,omitempty"`
	PCI        pci.PCI    `json:"pci,omitempty"`
}

// RingBuffer represents NIC ring buffer configuration.
// Using uint32 for numeric values enables calculations and comparisons.
type RingBuffer struct {
	CurrentRX uint32 `json:"current_rx,omitempty"`
	CurrentTX uint32 `json:"current_tx,omitempty"`
	MaxRX     uint32 `json:"max_rx,omitempty"`
	MaxTX     uint32 `json:"max_tx,omitempty"`
}

// Channel represents NIC channel/queue configuration.
type Channel struct {
	MaxRX           uint32 `json:"max_rx,omitempty"`
	MaxTX           uint32 `json:"max_tx,omitempty"`
	MaxCombined     uint32 `json:"max_combined,omitempty"`
	MaxOther        uint32 `json:"max_other,omitempty"`
	CurrentRX       uint32 `json:"current_rx,omitempty"`
	CurrentTX       uint32 `json:"current_tx,omitempty"`
	CurrentCombined uint32 `json:"current_combined,omitempty"`
	CurrentOther    uint32 `json:"current_other,omitempty"`
}

// LLDP represents Link Layer Discovery Protocol information
// from upstream ToR (Top of Rack) switch.
type LLDP struct {
	Interface    string `json:"interface,omitempty"`
	ChassisID    string `json:"chassis_id,omitempty"`
	SystemName   string `json:"system_name,omitempty"`
	SystemDesc   string `json:"system_desc,omitempty"`
	PortID       string `json:"port_id,omitempty"`
	PortDesc     string `json:"port_desc,omitempty"`
	ManagementIP string `json:"management_ip,omitempty"`
	VLAN         uint16 `json:"vlan,omitempty"` // VLAN ID: 1-4094
	PPVID        uint16 `json:"ppvid,omitempty"`
}

// BondInterface represents a Linux bonding interface configuration.
type BondInterface struct {
	BondName           string           `json:"bond_name,omitempty"`
	BondMode           string           `json:"bond_mode,omitempty"`
	TransmitHashPolicy string           `json:"transmit_hash_policy,omitempty"` // Fixed: lowercase 't'
	MIIStatus          string           `json:"mii_status,omitempty"`
	MIIPollingInterval uint32           `json:"mii_polling_interval,omitempty"` // Milliseconds
	LACPRate           string           `json:"lacp_rate,omitempty"`
	MACAddress         string           `json:"mac_address,omitempty"`
	AggregatorID       uint16           `json:"aggregator_id,omitempty"`
	NumberOfPorts      uint8            `json:"number_of_ports,omitempty"`
	Diagnose           DiagnoseStatus   `json:"diagnose,omitempty"`
	DiagnoseDetail     string           `json:"diagnose_detail,omitempty"`
	SlaveInterfaces    []SlaveInterface `json:"slave_interfaces,omitempty"`

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

// MarshalJSON implements custom JSON marshaling for DiagnoseStatus
func (d DiagnoseStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements custom JSON unmarshaling for DiagnoseStatus
func (d *DiagnoseStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	case "ok":
		*d = DiagnoseOK
	case "warning":
		*d = DiagnoseWarning
	case "error":
		*d = DiagnoseError
	default:
		*d = DiagnoseUnknown
	}
	return nil
}

// SlaveInterface represents a bond slave (member) interface.
type SlaveInterface struct {
	SlaveName     string `json:"slave_name,omitempty"`
	MIIStatus     string `json:"mii_status,omitempty"`
	Duplex        string `json:"duplex,omitempty"`
	Speed         uint64 `json:"speed,omitempty"` // Mbps
	LinkFailCount uint32 `json:"link_fail_count,omitempty"`
	MACAddress    string `json:"mac_address,omitempty"`
	SlaveQueueID  uint16 `json:"slave_queue_id,omitempty"`
	AggregatorID  uint16 `json:"aggregator_id,omitempty"`
}

// Constructor and Initialization

// NewNetwork creates a new Network instance with initialized indexes.
func NewNetwork() *Network {
	return &Network{
		NetInterfaces:    make([]NetInterface, 0),
		PhyInterfaces:    make([]PhyInterface, 0),
		BondInterfaces:   make([]BondInterface, 0),
		netInterfaceIdx:  make(map[string]*NetInterface),
		phyInterfaceIdx:  make(map[string]*PhyInterface),
		bondInterfaceIdx: make(map[string]*BondInterface),
	}
}

// NewNetworkWithCapacity creates a Network with pre-allocated capacity.
// Use when the approximate number of interfaces is known.
func NewNetworkWithCapacity(netCap, phyCap, bondCap int) *Network {
	return &Network{
		NetInterfaces:    make([]NetInterface, 0, netCap),
		PhyInterfaces:    make([]PhyInterface, 0, phyCap),
		BondInterfaces:   make([]BondInterface, 0, bondCap),
		netInterfaceIdx:  make(map[string]*NetInterface, netCap),
		phyInterfaceIdx:  make(map[string]*PhyInterface, phyCap),
		bondInterfaceIdx: make(map[string]*BondInterface, bondCap),
	}
}

// Network Methods - Add Operations

// AddNetInterface adds a network interface with validation.
// Returns error if interface name already exists or validation fails.
func (n *Network) AddNetInterface(iface NetInterface) error {
	if err := iface.Validate(); err != nil {
		return fmt.Errorf("validating interface %s: %w", iface.DeviceName, err)
	}

	// Initialize index if nil (for deserialized instances)
	if n.netInterfaceIdx == nil {
		n.netInterfaceIdx = make(map[string]*NetInterface)
	}

	if _, exists := n.netInterfaceIdx[iface.DeviceName]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateInterface, iface.DeviceName)
	}

	n.NetInterfaces = append(n.NetInterfaces, iface)
	// Store pointer to the element in slice
	n.netInterfaceIdx[iface.DeviceName] = &n.NetInterfaces[len(n.NetInterfaces)-1]
	return nil
}

// AddPhyInterface adds a physical interface with validation.
func (n *Network) AddPhyInterface(iface PhyInterface) error {
	if err := iface.Validate(); err != nil {
		return fmt.Errorf("validating physical interface %s: %w", iface.DeviceName, err)
	}

	if n.phyInterfaceIdx == nil {
		n.phyInterfaceIdx = make(map[string]*PhyInterface)
	}

	if _, exists := n.phyInterfaceIdx[iface.DeviceName]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateInterface, iface.DeviceName)
	}

	n.PhyInterfaces = append(n.PhyInterfaces, iface)
	n.phyInterfaceIdx[iface.DeviceName] = &n.PhyInterfaces[len(n.PhyInterfaces)-1]
	return nil
}

// AddBondInterface adds a bond interface with validation.
func (n *Network) AddBondInterface(iface BondInterface) error {
	if err := iface.Validate(); err != nil {
		return fmt.Errorf("validating bond interface %s: %w", iface.BondName, err)
	}

	if n.bondInterfaceIdx == nil {
		n.bondInterfaceIdx = make(map[string]*BondInterface)
	}

	if _, exists := n.bondInterfaceIdx[iface.BondName]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateInterface, iface.BondName)
	}

	// Build slave index
	iface.buildSlaveIndex()

	n.BondInterfaces = append(n.BondInterfaces, iface)
	n.bondInterfaceIdx[iface.BondName] = &n.BondInterfaces[len(n.BondInterfaces)-1]
	return nil
}

// Network Methods - Lookup Operations (O(1))

// GetNetInterface returns a network interface by name in O(1) time.
func (n *Network) GetNetInterface(name string) (*NetInterface, error) {
	if n.netInterfaceIdx == nil {
		return nil, ErrInterfaceNotFound
	}
	if iface, ok := n.netInterfaceIdx[name]; ok {
		return iface, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrInterfaceNotFound, name)
}

// GetPhyInterface returns a physical interface byname in O(1) time.
func (n *Network) GetPhyInterface(name string) (*PhyInterface, error) {
	if n.phyInterfaceIdx == nil {
		return nil, ErrInterfaceNotFound
	}
	if iface, ok := n.phyInterfaceIdx[name]; ok {
		return iface, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrInterfaceNotFound, name)
}

// GetBondInterface returns a bond interface by name in O(1) time.
func (n *Network) GetBondInterface(name string) (*BondInterface, error) {
	if n.bondInterfaceIdx == nil {
		return nil, ErrInterfaceNotFound
	}
	if iface, ok := n.bondInterfaceIdx[name]; ok {
		return iface, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrInterfaceNotFound, name)
}

// RebuildIndexes rebuilds all indexes after deserialization.
func (n *Network) RebuildIndexes() {
	n.netInterfaceIdx = make(map[string]*NetInterface, len(n.NetInterfaces))
	for i := range n.NetInterfaces {
		n.netInterfaceIdx[n.NetInterfaces[i].DeviceName] = &n.NetInterfaces[i]
	}

	n.phyInterfaceIdx = make(map[string]*PhyInterface, len(n.PhyInterfaces))
	for i := range n.PhyInterfaces {
		n.phyInterfaceIdx[n.PhyInterfaces[i].DeviceName] = &n.PhyInterfaces[i]
	}

	n.bondInterfaceIdx = make(map[string]*BondInterface, len(n.BondInterfaces))
	for i := range n.BondInterfaces {
		n.BondInterfaces[i].buildSlaveIndex()
		n.bondInterfaceIdx[n.BondInterfaces[i].BondName] = &n.BondInterfaces[i]
	}
}

// Validation Methods

// Validate validates NetInterface fields.
func (ni *NetInterface) Validate() error {
	if ni.DeviceName == "" {
		return errors.New("device name is required")
	}
	if ni.MACAddress != "" && !macAddressPattern.MatchString(ni.MACAddress) {
		return ErrInvalidMACAddress
	}
	return nil
}

// Validate validates PhyInterface fields.
func (pi *PhyInterface) Validate() error {
	if pi.DeviceName == "" {
		return errors.New("device name is required")
	}
	return pi.RingBuffer.Validate()
}

// Validate validates RingBuffer configuration.
func (rb *RingBuffer) Validate() error {
	if rb.CurrentRX > rb.MaxRX && rb.MaxRX > 0 {
		return fmt.Errorf("current RX (%d) exceeds max RX (%d)", rb.CurrentRX, rb.MaxRX)
	}
	if rb.CurrentTX > rb.MaxTX && rb.MaxTX > 0 {
		return fmt.Errorf("current TX (%d) exceeds max TX (%d)", rb.CurrentTX, rb.MaxTX)
	}
	return nil
}

// Validate validates BondInterface configuration.
func (bi *BondInterface) Validate() error {
	if bi.BondName == "" {
		return errors.New("bond name is required")
	}
	if bi.MACAddress != "" && !macAddressPattern.MatchString(bi.MACAddress) {
		return ErrInvalidMACAddress
	}
	for i := range bi.SlaveInterfaces {
		if err := bi.SlaveInterfaces[i].Validate(); err != nil {
			return fmt.Errorf("slave %s: %w", bi.SlaveInterfaces[i].SlaveName, err)
		}
	}
	return nil
}

// Validate validates SlaveInterface fields.
func (si *SlaveInterface) Validate() error {
	if si.SlaveName == "" {
		return errors.New("slave name is required")
	}
	if si.MACAddress != "" && !macAddressPattern.MatchString(si.MACAddress) {
		return ErrInvalidMACAddress
	}
	return nil
}

// Validate validates LLDP information.
func (l *LLDP) Validate() error {
	if l.ManagementIP != "" && net.ParseIP(l.ManagementIP) == nil {
		return errors.New("invalid management IP address")
	}
	if l.VLAN > 4094 {
		return ErrInvalidVLAN
	}
	return nil
}

// Helper Methods

// buildSlaveIndex builds the slave interface index for O(1) lookup.
func (bi *BondInterface) buildSlaveIndex() {
	bi.slaveIdx = make(map[string]*SlaveInterface, len(bi.SlaveInterfaces))
	for i := range bi.SlaveInterfaces {
		bi.slaveIdx[bi.SlaveInterfaces[i].SlaveName] = &bi.SlaveInterfaces[i]
	}
}

// GetSlave returns a slave interface by name in O(1) time.
func (bi *BondInterface) GetSlave(name string) (*SlaveInterface, error) {
	if bi.slaveIdx == nil {
		bi.buildSlaveIndex()
	}
	if slave, ok := bi.slaveIdx[name]; ok {
		return slave, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrInterfaceNotFound, name)
}

// IsUp returns true if the interface status is up.
func (ni *NetInterface) IsUp() bool {
	return strings.ToLower(ni.Status) == StatusUp
}

// SpeedString returns human-readable speed string.
func (ni *NetInterface) SpeedString() string {
	if ni.Speed >= 1000 {
		return strconv.FormatUint(ni.Speed/1000, 10) + " Gbps"
	}
	return strconv.FormatUint(ni.Speed, 10) + " Mbps"
}

// TotalBandwidth calculates total bandwidth of all active slaves.
func (bi *BondInterface) TotalBandwidth() uint64 {
	var total uint64
	for _, slave := range bi.SlaveInterfaces {
		if strings.ToLower(slave.MIIStatus) == StatusUp {
			total += slave.Speed
		}
	}
	return total
}

// ActiveSlaveCount returns number of active slave interfaces.
func (bi *BondInterface) ActiveSlaveCount() int {
	count := 0
	for _, slave := range bi.SlaveInterfaces {
		if strings.ToLower(slave.MIIStatus) == StatusUp {
			count++
		}
	}
	return count
}
