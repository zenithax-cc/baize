package pci

// PCI 表示PCI设备信息
type PCI struct {
	PCIID       string    `json:"pci_id,omitzero"`            // PCI设备ID
	PCIAddr     string    `json:"pci_address,omitzero"`       // PCI设备地址
	Vendor      string    `json:"vendor,omitzero"`            // 厂商名称
	VendorID    string    `json:"vendor_id,omitzero"`         // 厂商ID
	Device      string    `json:"device,omitzero"`            // 设备名称
	DeviceID    string    `json:"device_id,omitzero"`         // 设备ID
	SubVendor   string    `json:"sub_vendor,omitzero"`        // 子厂商名称
	SubVendorID string    `json:"sub_vendor_id,omitzero"`     // 子厂商ID
	SubDevice   string    `json:"sub_device,omitzero"`        // 子设备名称
	SubDeviceID string    `json:"sub_device_id,omitzero"`     // 子设备ID
	Class       string    `json:"class,omitzero"`             // 设备类型
	ClassID     string    `json:"class_id,omitzero"`          // 设备类型ID
	SubClass    string    `json:"sub_class,omitzero"`         // 子设备类型
	SubClassID  string    `json:"sub_class_id,omitzero"`      //	子设备类型ID
	ProgIfID    string    `json:"prog_interface_id,omitzero"` // 编程接口ID
	Numa        string    `json:"numa,omitzero"`              // NUMA节点
	Revision    string    `json:"revision,omitzero"`          // 修订版本
	Driver      PCIDriver `json:"driver,omitzero"`            // 驱动信息
	Link        PCILink   `json:"link,omitzero"`              // 链接信息
}

// PCIDriver 表示PCI设备的驱动信息
type PCIDriver struct {
	DriverName string `json:"driver_name,omitzero"`    // 驱动名称
	DriverVer  string `json:"driver_version,omitzero"` // 驱动版本
	SrcVer     string `json:"src_version,omitzero"`    // 源版本
	FileName   string `json:"file_name,omitzero"`      // 文件名
}

// PCILink 表示PCI设备的链接信息
type PCILink struct {
	MaxSpeed  string `json:"max_link_speed,omitzero"`     // 最大链接速度
	MaxWidth  string `json:"max_link_width,omitzero"`     // 最大链接宽度
	CurrSpeed string `json:"current_link_speed,omitzero"` // 当前链接速度
	CurrWidth string `json:"current_link_width,omitzero"` // 当前链接宽度
}
