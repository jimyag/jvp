package libvirt

import (
	"encoding/xml"
	"fmt"
)

// NodeDeviceXML 节点设备 XML 结构
type NodeDeviceXML struct {
	XMLName     xml.Name               `xml:"device"`
	Name        string                 `xml:"name"`
	Path        string                 `xml:"path"`
	Parent      string                 `xml:"parent"`
	Capability  NodeDeviceCapability   `xml:"capability"`
}

// NodeDeviceCapability 设备能力
type NodeDeviceCapability struct {
	Type        string                    `xml:"type,attr"`
	// 通用字段（PCI/USB 共用）
	Domain      string                    `xml:"domain"`
	Bus         string                    `xml:"bus"`      // PCI 的 bus 或 USB 的 bus number
	Slot        string                    `xml:"slot"`
	Function    string                    `xml:"function"`
	Device      string                    `xml:"device"`   // USB 的 device number
	Product     NodeDeviceProduct         `xml:"product"`
	Vendor      NodeDeviceVendor          `xml:"vendor"`
	IOMMUGroup  *NodeDeviceIOMMUGroup     `xml:"iommuGroup"`
	// 存储设备字段
	Block       string                    `xml:"block"`
	DriveType   string                    `xml:"drive_type"`
	Model       string                    `xml:"model"`
	Serial      string                    `xml:"serial"`
	Size        string                    `xml:"size"`
	// 网络接口字段
	Interface   string                    `xml:"interface"`
	Address     string                    `xml:"address"`
	Link        NodeDeviceLink            `xml:"link"`
	// 嵌套的 capability（用于识别子类型）
	SubCapability *NodeDeviceCapability   `xml:"capability"`
}

// NodeDeviceProduct 产品信息
type NodeDeviceProduct struct {
	ID   string `xml:"id,attr"`
	Name string `xml:",chardata"`
}

// NodeDeviceVendor 厂商信息
type NodeDeviceVendor struct {
	ID   string `xml:"id,attr"`
	Name string `xml:",chardata"`
}

// NodeDeviceIOMMUGroup IOMMU 组信息
type NodeDeviceIOMMUGroup struct {
	Number string `xml:"number,attr"`
}

// NodeDeviceLink 网络链接状态
type NodeDeviceLink struct {
	State string `xml:"state,attr"`
	Speed string `xml:"speed,attr"`
}

// ParseNodeDeviceXML 解析节点设备 XML
func ParseNodeDeviceXML(xmlData string) (*NodeDeviceXML, error) {
	var device NodeDeviceXML
	if err := xml.Unmarshal([]byte(xmlData), &device); err != nil {
		return nil, err
	}
	return &device, nil
}

// IsPCIDevice 判断是否为 PCI 设备
func (d *NodeDeviceXML) IsPCIDevice() bool {
	return d.Capability.Type == "pci"
}

// IsUSBDevice 判断是否为 USB 设备
func (d *NodeDeviceXML) IsUSBDevice() bool {
	return d.Capability.Type == "usb_device"
}

// IsNetworkInterface 判断是否为网络接口
func (d *NodeDeviceXML) IsNetworkInterface() bool {
	return d.Capability.Type == "net"
}

// IsStorageDevice 判断是否为存储设备（磁盘）
func (d *NodeDeviceXML) IsStorageDevice() bool {
	if d.Capability.Type == "storage" {
		// 只返回真正的磁盘设备，排除 CD-ROM 等
		return d.Capability.DriveType == "disk"
	}
	return false
}

// GetPCIAddress 获取 PCI 地址
func (d *NodeDeviceXML) GetPCIAddress() string {
	if !d.IsPCIDevice() {
		return ""
	}
	// 移除前导的 0x
	domain := d.Capability.Domain
	bus := d.Capability.Bus
	slot := d.Capability.Slot
	function := d.Capability.Function

	return domain + ":" + bus + ":" + slot + "." + function
}

// GetIOMMUGroup 获取 IOMMU 组号
func (d *NodeDeviceXML) GetIOMMUGroup() int {
	if d.Capability.IOMMUGroup != nil {
		var group int
		_, _ = fmt.Sscanf(d.Capability.IOMMUGroup.Number, "%d", &group)
		return group
	}
	return -1
}
