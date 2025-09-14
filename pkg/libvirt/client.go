package libvirt

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/digitalocean/go-libvirt"
)

type Client struct {
	conn *libvirt.Libvirt
}

// DomainInfo 包含域的详细信息
type DomainInfo struct {
	Name        string             `json:"name"`
	UUID        string             `json:"uuid"`
	State       string             `json:"state"`
	MaxMemory   uint64             `json:"max_memory"` // KB
	Memory      uint64             `json:"memory"`     // KB
	VCPUs       uint16             `json:"vcpus"`
	CPUTime     uint64             `json:"cpu_time"` // nanoseconds
	OSType      string             `json:"os_type"`
	Autostart   bool               `json:"autostart"`
	Persistent  bool               `json:"persistent"`
	NetworkInfo []NetworkInterface `json:"network_interfaces"`
	StartTime   *time.Time         `json:"start_time,omitempty"`
	OSVersion   string             `json:"os_version,omitempty"`
}

// NetworkInterface 网络接口信息
type NetworkInterface struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source"`
	MAC    string `json:"mac"`
	Model  string `json:"model"`
}

// XML 结构体用于解析域的 XML 配置
type DomainXML struct {
	XMLName xml.Name `xml:"domain"`
	Name    string   `xml:"name"`
	OS      struct {
		Type string `xml:"type"`
	} `xml:"os"`
	Devices struct {
		Interfaces []struct {
			Type   string `xml:"type,attr"`
			Source struct {
				Network string `xml:"network,attr"`
				Bridge  string `xml:"bridge,attr"`
			} `xml:"source"`
			MAC struct {
				Address string `xml:"address,attr"`
			} `xml:"mac"`
			Model struct {
				Type string `xml:"type,attr"`
			} `xml:"model"`
			Target struct {
				Dev string `xml:"dev,attr"`
			} `xml:"target"`
		} `xml:"interface"`
	} `xml:"devices"`
}

func New() (*Client, error) {
	uri, _ := url.Parse(string(libvirt.QEMUSystem))
	l, err := libvirt.ConnectToURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return &Client{conn: l}, nil
}

// formatLibvirtVersion converts libvirt version number to human readable format
// libvirt version is encoded as: major * 1000000 + minor * 1000 + micro
// For example: 8003000 = 8.3.0
func formatLibvirtVersion(version uint64) string {
	major := version / 1000000
	minor := (version % 1000000) / 1000
	micro := version % 1000
	return fmt.Sprintf("%d.%d.%d", major, minor, micro)
}

func (c *Client) GetVMSummaries() ([]libvirt.Domain, error) {
	v, err := c.conn.ConnectGetLibVersion()
	if err != nil {
		log.Fatalf("failed to retrieve libvirt version: %v", err)
	}
	fmt.Printf("Version: %s (raw: %d)\n", formatLibvirtVersion(v), v)

	// 获取所有类型的域信息，不限制数量
	flags := libvirt.ConnectListDomainsActive |
		libvirt.ConnectListDomainsInactive |
		libvirt.ConnectListDomainsPersistent |
		libvirt.ConnectListDomainsTransient |
		libvirt.ConnectListDomainsRunning |
		libvirt.ConnectListDomainsPaused |
		libvirt.ConnectListDomainsShutoff |
		libvirt.ConnectListDomainsOther |
		libvirt.ConnectListDomainsManagedsave |
		libvirt.ConnectListDomainsNoManagedsave |
		libvirt.ConnectListDomainsAutostart |
		libvirt.ConnectListDomainsNoAutostart |
		libvirt.ConnectListDomainsHasSnapshot |
		libvirt.ConnectListDomainsNoSnapshot |
		libvirt.ConnectListDomainsHasCheckpoint |
		libvirt.ConnectListDomainsNoCheckpoint

	// NeedResults 设置为足够大的数字以获取所有域
	domains, _, err := c.conn.ConnectListAllDomains(1000, flags)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve domains: %v", err)
	}

	fmt.Println("UUID\tName")
	fmt.Printf("--------------------------------------------------------\n")
	for _, d := range domains {
		fmt.Printf("%x\t%s\n", d.UUID, d.Name)
	}

	return domains, nil
}

// GetDomainInfo 获取指定域的详细信息
func (c *Client) GetDomainInfo(domainUUID libvirt.UUID) (*DomainInfo, error) {
	// 通过名称查找域
	domain, err := c.conn.DomainLookupByUUID(domainUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup domain '%s': %v", domainUUID, err)
	}

	info := &DomainInfo{
		Name: domain.Name,
		UUID: fmt.Sprintf("%x", domain.UUID),
	}

	// 获取基本信息
	state, maxMem, memory, vcpus, cpuTime, err := c.conn.DomainGetInfo(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain info: %v", err)
	}

	info.State = c.formatDomainState(state)
	info.MaxMemory = maxMem
	info.Memory = memory
	info.VCPUs = vcpus
	info.CPUTime = cpuTime

	// 获取 OS 类型
	osType, err := c.conn.DomainGetOsType(domain)
	if err == nil {
		info.OSType = osType
	}

	// 获取自动启动状态
	autostart, err := c.conn.DomainGetAutostart(domain)
	if err == nil {
		info.Autostart = autostart != 0
	}

	// 检查是否为持久化域
	info.Persistent = domain.ID != -1 || c.isDomainPersistent(domain)

	// 获取网络接口信息
	networkInfo, err := c.getDomainNetworkInfo(domain)
	if err == nil {
		info.NetworkInfo = networkInfo
	}

	// 尝试获取启动时间（仅对运行中的域有效）
	if state == uint8(libvirt.DomainRunning) {
		startTime := c.getDomainStartTime(domain)
		if startTime != nil {
			info.StartTime = startTime
		}
	}

	return info, nil
}

// formatDomainState 将域状态数字转换为可读字符串
func (c *Client) formatDomainState(state uint8) string {
	switch libvirt.DomainState(state) {
	case libvirt.DomainNostate:
		return "No State"
	case libvirt.DomainRunning:
		return "Running"
	case libvirt.DomainBlocked:
		return "Blocked"
	case libvirt.DomainPaused:
		return "Paused"
	case libvirt.DomainShutdown:
		return "Shutting Down"
	case libvirt.DomainShutoff:
		return "Shut Off"
	case libvirt.DomainCrashed:
		return "Crashed"
	case libvirt.DomainPmsuspended:
		return "PM Suspended"
	default:
		return fmt.Sprintf("Unknown (%d)", state)
	}
}

// isDomainPersistent 检查域是否为持久化域
func (c *Client) isDomainPersistent(domain libvirt.Domain) bool {
	// 尝试获取 XML 配置，如果成功则说明是持久化的
	_, err := c.conn.DomainGetXMLDesc(domain, 0)
	return err == nil
}

// getDomainNetworkInfo 获取域的网络接口信息
func (c *Client) getDomainNetworkInfo(domain libvirt.Domain) ([]NetworkInterface, error) {
	// 获取域的 XML 描述
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML: %v", err)
	}

	// 解析 XML
	var domainXML DomainXML
	err = xml.Unmarshal([]byte(xmlDesc), &domainXML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %v", err)
	}

	var interfaces []NetworkInterface
	for _, iface := range domainXML.Devices.Interfaces {
		netIface := NetworkInterface{
			Name:  iface.Target.Dev,
			Type:  iface.Type,
			MAC:   iface.MAC.Address,
			Model: iface.Model.Type,
		}

		// 设置网络源
		if iface.Source.Network != "" {
			netIface.Source = iface.Source.Network
		} else if iface.Source.Bridge != "" {
			netIface.Source = iface.Source.Bridge
		}

		interfaces = append(interfaces, netIface)
	}

	return interfaces, nil
}

// getDomainStartTime 尝试获取域的启动时间
func (c *Client) getDomainStartTime(domain libvirt.Domain) *time.Time {
	// 这是一个简化的实现，实际的启动时间获取可能需要更复杂的逻辑
	// 可以通过查看/proc/stat 或其他系统信息来获取更准确的启动时间

	// 获取当前 CPU 时间（纳秒）
	_, _, _, _, cpuTime, err := c.conn.DomainGetInfo(domain)
	if err != nil {
		return nil
	}

	// 简化计算：假设域从当前时间减去 CPU 时间开始运行
	// 注意：这只是一个粗略的估计
	if cpuTime > 0 {
		startTime := time.Now().Add(-time.Duration(cpuTime))
		return &startTime
	}

	return nil
}

// PrintDomainInfo 格式化打印域信息
func (c *Client) PrintDomainInfo(info *DomainInfo) {
	fmt.Printf("Domain Information:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Name:         %s\n", info.Name)
	fmt.Printf("UUID:         %s\n", info.UUID)
	fmt.Printf("State:        %s\n", info.State)
	fmt.Printf("OS Type:      %s\n", info.OSType)
	fmt.Printf("Max Memory:   %d KB (%.2f GB)\n", info.MaxMemory, float64(info.MaxMemory)/1024/1024)
	fmt.Printf("Memory:       %d KB (%.2f GB)\n", info.Memory, float64(info.Memory)/1024/1024)
	fmt.Printf("VCPUs:        %d\n", info.VCPUs)
	fmt.Printf("CPU Time:     %d ns (%s)\n", info.CPUTime, time.Duration(info.CPUTime))
	fmt.Printf("Autostart:    %t\n", info.Autostart)
	fmt.Printf("Persistent:   %t\n", info.Persistent)

	if info.StartTime != nil {
		fmt.Printf("Start Time:   %s\n", info.StartTime.Format("2006-01-02 15:04:05"))
	}

	if len(info.NetworkInfo) > 0 {
		fmt.Printf("\nNetwork Interfaces:\n")
		fmt.Printf("-------------------\n")
		for i, iface := range info.NetworkInfo {
			fmt.Printf("Interface %d:\n", i+1)
			fmt.Printf("  Name:   %s\n", iface.Name)
			fmt.Printf("  Type:   %s\n", iface.Type)
			fmt.Printf("  Source: %s\n", iface.Source)
			fmt.Printf("  MAC:    %s\n", iface.MAC)
			fmt.Printf("  Model:  %s\n", iface.Model)
			fmt.Printf("\n")
		}
	}
}
