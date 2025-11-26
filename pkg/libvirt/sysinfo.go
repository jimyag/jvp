package libvirt

import (
	"encoding/xml"
	"strings"
)

// SysinfoXML 是 libvirt sysinfo 的 XML 结构
type SysinfoXML struct {
	XMLName    xml.Name          `xml:"sysinfo"`
	Type       string            `xml:"type,attr"`
	BIOS       SysinfoBIOS       `xml:"bios"`
	System     SysinfoSystem     `xml:"system"`
	BaseBoard  SysinfoBaseBoard  `xml:"baseBoard"`
	Chassis    SysinfoChassis    `xml:"chassis"`
	Processor  SysinfoProcessor  `xml:"processor"`
	Memory     []SysinfoMemory   `xml:"memory_device"`
	OEMStrings SysinfoOEMStrings `xml:"oemStrings"`
}

// SysinfoBIOS BIOS 信息
type SysinfoBIOS struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoSystem 系统信息
type SysinfoSystem struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoBaseBoard 主板信息
type SysinfoBaseBoard struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoChassis 机箱信息
type SysinfoChassis struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoProcessor 处理器信息
type SysinfoProcessor struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoMemory 内存设备信息
type SysinfoMemory struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoOEMStrings OEM 字符串
type SysinfoOEMStrings struct {
	Entries []SysinfoEntry `xml:"entry"`
}

// SysinfoEntry sysinfo 条目
type SysinfoEntry struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// ParseSysinfo 解析 sysinfo XML
func ParseSysinfo(xmlData string) (*SysinfoXML, error) {
	var sysinfo SysinfoXML
	err := xml.Unmarshal([]byte(xmlData), &sysinfo)
	if err != nil {
		return nil, err
	}
	return &sysinfo, nil
}

// GetProcessorVersion 获取 CPU 型号名称
func (s *SysinfoXML) GetProcessorVersion() string {
	for _, entry := range s.Processor.Entries {
		if entry.Name == "version" {
			return strings.TrimSpace(entry.Value)
		}
	}
	return ""
}

// GetProcessorMaxSpeed 获取 CPU 最大频率
func (s *SysinfoXML) GetProcessorMaxSpeed() string {
	for _, entry := range s.Processor.Entries {
		if entry.Name == "max_speed" {
			return strings.TrimSpace(entry.Value)
		}
	}
	return ""
}

// GetProcessorManufacturer 获取 CPU 制造商
func (s *SysinfoXML) GetProcessorManufacturer() string {
	for _, entry := range s.Processor.Entries {
		if entry.Name == "manufacturer" {
			return strings.TrimSpace(entry.Value)
		}
	}
	return ""
}
