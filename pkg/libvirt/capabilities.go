package libvirt

import (
	"encoding/xml"
)

// CapabilitiesXML 是 libvirt capabilities 的 XML 结构
type CapabilitiesXML struct {
	XMLName xml.Name        `xml:"capabilities"`
	Host    CapabilitiesHost `xml:"host"`
}

// CapabilitiesHost 主机信息
type CapabilitiesHost struct {
	UUID     string              `xml:"uuid"`
	CPU      CapabilitiesCPU     `xml:"cpu"`
	IOMMU    CapabilitiesIOMMU   `xml:"iommu"`
	Topology CapabilitiesTopology `xml:"topology"`
	Cache    CapabilitiesCache   `xml:"cache"`
}

// CapabilitiesCPU CPU 信息
type CapabilitiesCPU struct {
	Arch       string                       `xml:"arch"`
	Model      string                       `xml:"model"`
	Vendor     string                       `xml:"vendor"`
	Microcode  CapabilitiesMicrocode        `xml:"microcode"`
	Signature  CapabilitiesSignature        `xml:"signature"`
	Counter    CapabilitiesCounter          `xml:"counter"`
	Topology   CapabilitiesCPUTopology      `xml:"topology"`
	MaxPhysAddr CapabilitiesMaxPhysAddr     `xml:"maxphysaddr"`
	Features   []CapabilitiesFeature        `xml:"feature"`
	Pages      []CapabilitiesPage           `xml:"pages"`
}

// CapabilitiesMicrocode 微代码版本
type CapabilitiesMicrocode struct {
	Version string `xml:"version,attr"`
}

// CapabilitiesSignature CPU 签名
type CapabilitiesSignature struct {
	Family   string `xml:"family,attr"`
	Model    string `xml:"model,attr"`
	Stepping string `xml:"stepping,attr"`
}

// CapabilitiesCounter 计数器信息
type CapabilitiesCounter struct {
	Name      string `xml:"name,attr"`
	Frequency string `xml:"frequency,attr"`
	Scaling   string `xml:"scaling,attr"`
}

// CapabilitiesCPUTopology CPU 拓扑
type CapabilitiesCPUTopology struct {
	Sockets string `xml:"sockets,attr"`
	Dies    string `xml:"dies,attr"`
	Cores   string `xml:"cores,attr"`
	Threads string `xml:"threads,attr"`
}

// CapabilitiesMaxPhysAddr 最大物理地址
type CapabilitiesMaxPhysAddr struct {
	Mode string `xml:"mode,attr"`
	Bits string `xml:"bits,attr"`
}

// CapabilitiesFeature CPU 特性
type CapabilitiesFeature struct {
	Name string `xml:"name,attr"`
}

// CapabilitiesPage 内存页信息
type CapabilitiesPage struct {
	Unit string `xml:"unit,attr"`
	Size string `xml:"size,attr"`
}

// CapabilitiesIOMMU IOMMU 信息
type CapabilitiesIOMMU struct {
	Support string `xml:"support,attr"`
}

// CapabilitiesTopology NUMA 拓扑
type CapabilitiesTopology struct {
	Cells CapabilitiesCells `xml:"cells"`
}

// CapabilitiesCells NUMA cells
type CapabilitiesCells struct {
	Num   string             `xml:"num,attr"`
	Cells []CapabilitiesCell `xml:"cell"`
}

// CapabilitiesCell NUMA cell
type CapabilitiesCell struct {
	ID        string                   `xml:"id,attr"`
	Memory    CapabilitiesCellMemory   `xml:"memory"`
	Pages     []CapabilitiesPage       `xml:"pages"`
	Distances CapabilitiesDistances    `xml:"distances"`
	CPUs      CapabilitiesCellCPUs     `xml:"cpus"`
}

// CapabilitiesCellMemory Cell 内存
type CapabilitiesCellMemory struct {
	Unit  string `xml:"unit,attr"`
	Value string `xml:",chardata"`
}

// CapabilitiesDistances NUMA 距离
type CapabilitiesDistances struct {
	Siblings []CapabilitiesSibling `xml:"sibling"`
}

// CapabilitiesSibling NUMA sibling
type CapabilitiesSibling struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

// CapabilitiesCellCPUs Cell CPUs
type CapabilitiesCellCPUs struct {
	Num  string               `xml:"num,attr"`
	CPUs []CapabilitiesCellCPU `xml:"cpu"`
}

// CapabilitiesCellCPU Cell CPU
type CapabilitiesCellCPU struct {
	ID       string `xml:"id,attr"`
	SocketID string `xml:"socket_id,attr"`
	DieID    string `xml:"die_id,attr"`
	CoreID   string `xml:"core_id,attr"`
	Siblings string `xml:"siblings,attr"`
}

// CapabilitiesCache Cache 信息
type CapabilitiesCache struct {
	Banks []CapabilitiesCacheBank `xml:"bank"`
}

// CapabilitiesCacheBank Cache bank
type CapabilitiesCacheBank struct {
	ID    string `xml:"id,attr"`
	Level string `xml:"level,attr"`
	Type  string `xml:"type,attr"`
	Size  string `xml:"size,attr"`
	Unit  string `xml:"unit,attr"`
	CPUs  string `xml:"cpus,attr"`
}

// ParseCapabilities 解析 capabilities XML
func ParseCapabilities(xmlData string) (*CapabilitiesXML, error) {
	var caps CapabilitiesXML
	err := xml.Unmarshal([]byte(xmlData), &caps)
	if err != nil {
		return nil, err
	}
	return &caps, nil
}
