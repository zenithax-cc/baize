package network

import "github.com/zenithax-cc/diting/internal/collector/pci"

// NetWork 网络信息
type Network struct {
	NetInterfaces  []NetInterface  `json:"net_interfaces,omitzero"`
	PhyInterfaces  []PhyInterface  `json:"phy_interfaces,omitzero"`
	BondInterfaces []BondInterface `json:"bond_interfaces,omitzero"`
}

// NetInterface 网络接口信息，包括物理接口、虚拟接口等，从/sys/class/net目录获取
type NetInterface struct {
	DeviceName      string `json:"device_name,omitzero"`      // 设备名称
	MACAddress      string `json:"mac_address,omitzero"`      // MAC地址
	Driver          string `json:"driver,omitzero"`           // 驱动名称
	DriverVersion   string `json:"driver_version,omitzero"`   // 驱动版本
	FirmwareVersion string `json:"firmware_version,omitzero"` // 固件版本
	Status          string `json:"status,omitzero"`           // 状态
	Speed           string `json:"speed,omitzero"`            // 速率
	Duplex          string `json:"duplex,omitzero"`           // 双工模式
	MTU             string `json:"mtu,omitzero"`              // 最大传输单元
	Port            string `json:"port,omitzero"`             // 端口
	LinkDetected    string `json:"link_detected,omitzero"`    // 链路检测
}

// PhyInterface 表示物理接口信息，包括网卡、交换机等
type PhyInterface struct {
	RingBuffer RingBuffer `json:"ring_buffer,omitzero"` // 环形缓冲区
	Channel    Channel    `json:"channel,omitzero"`     // 通道
	LLDP       LLDP       `json:"lldp,omitzero"`        // LLDP信息
	PCI        pci.PCI    `json:"pci,omitzero"`         // PCI信息
}

// RingBuffer 表示环形缓冲区信息
type RingBuffer struct {
	CurrentRX string `json:"current_rx,omitzero"` // 当前接收环形缓冲区大小
	CurrentTX string `json:"current_tx,omitzero"` // 当前发送环形缓冲区大小
	MaxRX     string `json:"max_rx,omitzero"`     // 最大接收环形缓冲区大小
	MaxTX     string `json:"max_tx,omitzero"`     // 最大发送环形缓冲区大小
}

// Channel 表示通道信息
type Channel struct {
	MaxRX           string `json:"max_rx,omitzero"`           // 最大接收通道数
	MaxTX           string `json:"max_tx,omitzero"`           // 最大发送通道数
	MaxCombined     string `json:"max_combined,omitzero"`     // 最大组合通道数
	CurrentRX       string `json:"current_rx,omitzero"`       // 当前接收通道数
	CurrentTX       string `json:"current_tx,omitzero"`       // 当前发送通道数
	CurrentCombined string `json:"current_combined,omitzero"` // 当前组合通道数
}

// LLDP 表示LLDP信息，上联tor端口信息
type LLDP struct {
	Interface    string `json:"interface,omitzero"`          // 接口名称
	ChassisID    string `json:"chassis_id,omitzero"`         // 设备ID
	SystemName   string `json:"system_name,omitzero"`        // 系统名称
	SystemDesc   string `json:"system_description,omitzero"` // 系统描述
	PortID       string `json:"port_id,omitzero"`            // 端口ID
	ManagementIP string `json:"management_ip,omitzero"`      // 管理IP地址
	VLAN         string `json:"vlan,omitzero"`               // VLAN ID
	PPVID        string `json:"ppvid,omitzero"`              // PPVID
}

// BondInterface 表示Bond接口信息
type BondInterface struct {
	BondName           string           `json:"bond_name,omitzero"`            // Bond接口名称
	BondMode           string           `json:"bond_mode,omitzero"`            // Bond模式
	TransmitHashPolicy string           `json:"Transmit_hash_policy,omitzero"` // 传输哈希策略
	MIIStatus          string           `json:"mii_status,omitzero"`           // MII状态
	MIIPollingInterval string           `json:"mii_polling_interval,omitzero"` // MII轮询间隔
	LACPRate           string           `json:"lacp_rate,omitzero"`            // LACP速率
	MACAddress         string           `json:"mac_address,omitzero"`          // MAC地址
	AggregatorID       string           `json:"aggregator_id,omitzero"`        // 聚合ID
	NumberOfPorts      string           `json:"number_of_ports,omitzero"`      // 端口数
	Diagnose           string           `json:"diagnose,omitzero"`             // 诊断情况
	DiagnoseDetail     string           `json:"diagnose_detail,omitzero"`      // 诊断详细信息
	SlaveInterfaces    []SlaveInterface `json:"slave_interfaces,omitzero"`     // 从接口信息
}

// SlaveInterface 表示bond从接口信息
type SlaveInterface struct {
	SlaveName     string `json:"slave_name,omitzero"`      // 从接口名称
	MIIStatus     string `json:"mii_status,omitzero"`      // MII状态
	Duplex        string `json:"duplex,omitzero"`          // 双工模式
	Speed         string `json:"speed,omitzero"`           // 速率
	LinkFailCount string `json:"link_fail_count,omitzero"` // 链路失败次数
	MACAddress    string `json:"mac_address,omitzero"`     // MAC地址
	SlaveQueueID  string `json:"slave_queue_id,omitzero"`  // 从接口队列ID
	AggregatorID  string `json:"aggregator_id,omitzero"`   // 聚合ID
}
