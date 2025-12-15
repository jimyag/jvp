package libvirt

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/pkg/cloudinit"
	"github.com/rs/zerolog/log"
)

type Client struct {
	conn *libvirt.Libvirt
	uri  string // 保存原始连接 URI
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

// CreateVMConfig 创建虚拟机配置参数
type CreateVMConfig struct {
	Name              string              // 虚拟机名称（必填）
	Memory            uint64              // 内存大小（KB）（必填）
	VCPUs             uint16              // 虚拟 CPU 数量（必填）
	DiskPath          string              // 磁盘路径（必填）
	DiskSize          uint64              // 磁盘大小（GB）（可选，用于创建新磁盘）
	DiskBus           string              // 磁盘总线类型：virtio, sata, scsi, ide（默认：virtio）
	NetworkType       string              // 网络类型：network, bridge, direct（默认：bridge）
	NetworkSource     string              // 网络源：网络名称或网桥名称（默认：br0）
	OSType            string              // 操作系统类型：hvm, linux, exe（默认：hvm）
	Architecture      string              // CPU 架构：x86_64, aarch64, i686 等（默认：x86_64）
	MachineType       string              // 机器类型（可选，如：pc-q35-6.2）
	ISOPath           string              // ISO 路径（可选，用于操作系统安装）
	VNCSocket         string              // VNC Unix socket 路径（可选，默认：/var/lib/jvp/qemu/{name}.vnc）
	Autostart         bool                // 是否开机自动启动（默认：false）
	CloudInit         *cloudinit.Config   // cloud-init 配置（可选）
	CloudInitUserData *cloudinit.UserData // cloud-init 用户数据（可选）
	cloudInitISOPath  string              // cloud-init ISO 路径（内部使用）
}

func New() (*Client, error) {
	return NewWithURI("")
}

// NewWithURI 使用指定的 URI 创建 libvirt 客户端
// 如果 uri 为空，则使用默认的 qemu:///system
func NewWithURI(uri string) (*Client, error) {
	if uri == "" {
		uri = string(libvirt.QEMUSystem)
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI %s: %v", uri, err)
	}

	l, err := libvirt.ConnectToURI(parsedURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt: %v", err)
	}

	return &Client{conn: l, uri: uri}, nil
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

// GetHostname 获取 libvirt 主机的 hostname
func (c *Client) GetHostname() (string, error) {
	hostname, err := c.conn.ConnectGetHostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	return hostname, nil
}

// GetLibvirtVersion 获取 libvirt 版本
func (c *Client) GetLibvirtVersion() (string, error) {
	v, err := c.conn.ConnectGetLibVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get libvirt version: %w", err)
	}
	return formatLibvirtVersion(v), nil
}

// GetNodeInfo 获取物理节点的硬件信息
func (c *Client) GetNodeInfo() (*NodeInfo, error) {
	model, memory, cpus, mhz, nodes, sockets, cores, threads, err := c.conn.NodeGetInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	// 转换 model ([]int8) 为 string
	modelBytes := make([]byte, len(model))
	for i, b := range model {
		if b == 0 {
			modelBytes = modelBytes[:i]
			break
		}
		modelBytes[i] = byte(b)
	}

	return &NodeInfo{
		Model:   string(modelBytes),
		Memory:  memory,
		CPUs:    uint32(cpus),
		MHz:     uint32(mhz),
		Nodes:   uint32(nodes),
		Sockets: uint32(sockets),
		Cores:   uint32(cores),
		Threads: uint32(threads),
	}, nil
}

// GetCapabilities 获取 libvirt 主机能力（XML 格式）
func (c *Client) GetCapabilities() (string, error) {
	caps, err := c.conn.ConnectGetCapabilities()
	if err != nil {
		return "", fmt.Errorf("failed to get capabilities: %w", err)
	}
	return caps, nil
}

// GetSysinfo 获取主机系统信息（SMBIOS，包含真实的 CPU 型号）
func (c *Client) GetSysinfo() (string, error) {
	sysinfo, err := c.conn.ConnectGetSysinfo(0)
	if err != nil {
		return "", fmt.Errorf("failed to get sysinfo: %w", err)
	}
	return sysinfo, nil
}

// NodeInfo 物理节点硬件信息
type NodeInfo struct {
	Model   string // CPU 型号
	Memory  uint64 // 内存大小 (KB)
	CPUs    uint32 // CPU 总数
	MHz     uint32 // CPU 频率 (MHz)
	Nodes   uint32 // NUMA 节点数
	Sockets uint32 // Socket 数
	Cores   uint32 // 每个 socket 的核心数
	Threads uint32 // 每个核心的线程数
}

func (c *Client) GetVMSummaries() ([]libvirt.Domain, error) {
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
		return "NoState"
	case libvirt.DomainRunning:
		return "Running"
	case libvirt.DomainBlocked:
		return "Blocked"
	case libvirt.DomainPaused:
		return "Paused"
	case libvirt.DomainShutdown:
		return "ShuttingDown"
	case libvirt.DomainShutoff:
		return "ShutOff"
	case libvirt.DomainCrashed:
		return "Crashed"
	case libvirt.DomainPmsuspended:
		return "PMSuspended"
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
	// 优先使用 libvirt 提供的控制信息中的状态时间（纳秒时间戳）
	if _, _, stateTime, err := c.conn.DomainGetControlInfo(domain, 229); err == nil && stateTime > 0 {
		start := time.Unix(0, int64(stateTime))
		return &start
	} else if err != nil {
		log.Warn().Err(err).Str("domain", domain.Name).Msg("Failed to get domain control info")
	}

	// 其次尝试获取 guest 时间（非精确启动时间，但可作为近似值避免为空）
	if seconds, nseconds, err := c.conn.DomainGetTime(domain, 337); err == nil && seconds > 0 {
		start := time.Unix(seconds, int64(nseconds))
		return &start
	} else if err != nil {
		log.Warn().Err(err).Str("domain", domain.Name).Msg("Failed to get domain time")
	}

	// 最后退化为 CPU 时间推算（粗略）
	if _, _, _, _, cpuTime, err := c.conn.DomainGetInfo(domain); err == nil && cpuTime > 0 {
		start := time.Now().Add(-time.Duration(cpuTime))
		return &start
	} else if err != nil {
		log.Warn().Err(err).Str("domain", domain.Name).Msg("Failed to get domain info")
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

// ============================================================================
// 虚拟机创建相关方法
// ============================================================================

// CreateDomainFromXML 从 DomainXML 结构创建虚拟机域
// persistent: true=持久化域（推荐），false=临时域
// 返回创建的 Domain 对象
func (c *Client) CreateDomainFromXML(domainXML *DomainXML, persistent bool) (libvirt.Domain, error) {
	// 序列化 DomainXML 为 XML 字符串
	xmlBytes, err := xml.MarshalIndent(domainXML, "", "  ")
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("failed to marshal domain XML: %v", err)
	}

	xmlString := string(xmlBytes)

	if persistent {
		// 方案 A：定义持久化域
		domain, err := c.conn.DomainDefineXML(xmlString)
		if err != nil {
			return libvirt.Domain{}, fmt.Errorf("failed to define domain: %v", err)
		}
		return domain, nil
	} else {
		// 创建临时域（会自动启动）
		domain, err := c.conn.DomainCreateXML(xmlString, libvirt.DomainNone)
		if err != nil {
			return libvirt.Domain{}, fmt.Errorf("failed to create transient domain: %v", err)
		}
		return domain, nil
	}
}

// DefineDomain 定义持久化域（不启动）
// 使用 CreateVMConfig 配置参数来构建域
func (c *Client) DefineDomain(config *CreateVMConfig) (libvirt.Domain, error) {
	// 验证配置
	if err := c.validateVMConfig(config); err != nil {
		return libvirt.Domain{}, fmt.Errorf("invalid config: %v", err)
	}

	// 设置默认值
	c.setDefaultVMConfig(config)

	// 确保 VNC socket 目录存在，并设置正确的权限
	if config.VNCSocket != "" {
		vncDir := filepath.Dir(config.VNCSocket)
		if c.IsRemoteConnection() {
			// 远程连接：通过 SSH 创建目录
			if err := c.ExecuteRemoteCommand(fmt.Sprintf("mkdir -p '%s' && chmod 755 '%s'", vncDir, vncDir)); err != nil {
				return libvirt.Domain{}, fmt.Errorf("create VNC socket directory on remote: %w", err)
			}
			// 尝试设置目录所有者（可能失败，取决于远程系统配置）
			_ = c.ExecuteRemoteCommand(fmt.Sprintf("chown libvirt-qemu:kvm '%s' 2>/dev/null || chown qemu:qemu '%s' 2>/dev/null || true", vncDir, vncDir))
		} else {
			// 本地连接：直接创建目录
			if err := os.MkdirAll(vncDir, 0o755); err != nil {
				return libvirt.Domain{}, fmt.Errorf("create VNC socket directory: %w", err)
			}
			// 设置目录所有者为 libvirt-qemu:kvm，以便 QEMU 进程可以创建 socket
			if err := c.fixVNCDirOwnership(vncDir); err != nil {
				// 非致命错误，只记录警告
				log.Warn().Err(err).Str("dir", vncDir).Msg("Failed to fix VNC directory ownership")
			}
		}
	}

	// 如果配置了 cloud-init，生成 cloud-init ISO
	if config.CloudInit != nil {
		isoPath, err := c.generateCloudInitISO(config)
		if err != nil {
			return libvirt.Domain{}, fmt.Errorf("failed to generate cloud-init ISO: %v", err)
		}
		config.cloudInitISOPath = isoPath
	}
	if config.CloudInitUserData != nil {
		isoPath, err := c.generateCloudInitISO(config)
		if err != nil {
			return libvirt.Domain{}, fmt.Errorf("failed to generate cloud-init ISO: %v", err)
		}
		config.cloudInitISOPath = isoPath
	}

	// 构建 DomainXML
	domainXML, err := c.buildDomainXML(config)
	if err != nil {
		// 清理 cloud-init ISO（如果生成了）
		c.cleanupCloudInitISOOnError(config.cloudInitISOPath)
		return libvirt.Domain{}, fmt.Errorf("failed to build domain XML: %v", err)
	}

	// 定义持久化域
	domain, err := c.CreateDomainFromXML(domainXML, true)
	if err != nil {
		return libvirt.Domain{}, err
	}

	// 设置自动启动（如果需要）
	if config.Autostart {
		if err := c.conn.DomainSetAutostart(domain, 1); err != nil {
			// 自动启动设置失败不是致命错误，记录日志即可
			log.Warn().Err(err).Str("domain", config.Name).Msg("Failed to set autostart for domain")
		}
	}

	return domain, nil
}

// StartDomain 启动已定义的域
func (c *Client) StartDomain(domain libvirt.Domain) error {
	// 启动前确保 VNC socket 目录存在
	if err := c.ensureVNCSocketDir(domain); err != nil {
		log.Warn().Err(err).Str("domain", domain.Name).Msg("Failed to ensure VNC socket directory")
	}

	err := c.conn.DomainCreate(domain)
	if err != nil {
		return fmt.Errorf("failed to start domain %s: %v", domain.Name, err)
	}
	return nil
}

// ensureVNCSocketDir 确保 VNC socket 目录存在
func (c *Client) ensureVNCSocketDir(domain libvirt.Domain) error {
	// 获取 domain XML
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return fmt.Errorf("get domain XML: %w", err)
	}

	// 解析 VNC socket 路径
	// 查找 <graphics type='vnc' ... socket='/path/to/socket' />
	// 使用简单的字符串查找
	socketStart := strings.Index(xmlDesc, "socket='/var/lib/jvp/qemu/")
	if socketStart == -1 {
		// 没有使用 Unix socket 的 VNC
		return nil
	}

	// 提取 socket 路径
	socketStart += len("socket='")
	socketEnd := strings.Index(xmlDesc[socketStart:], "'")
	if socketEnd == -1 {
		return nil
	}
	socketPath := xmlDesc[socketStart : socketStart+socketEnd]

	// 获取目录
	vncDir := filepath.Dir(socketPath)

	// 创建目录
	if c.IsRemoteConnection() {
		if err := c.ExecuteRemoteCommand(fmt.Sprintf("mkdir -p '%s' && chmod 755 '%s'", vncDir, vncDir)); err != nil {
			return fmt.Errorf("create VNC socket directory on remote: %w", err)
		}
		_ = c.ExecuteRemoteCommand(fmt.Sprintf("chown libvirt-qemu:kvm '%s' 2>/dev/null || chown qemu:qemu '%s' 2>/dev/null || true", vncDir, vncDir))
	} else {
		if err := os.MkdirAll(vncDir, 0o755); err != nil {
			return fmt.Errorf("create VNC socket directory: %w", err)
		}
		if err := c.fixVNCDirOwnership(vncDir); err != nil {
			log.Warn().Err(err).Str("dir", vncDir).Msg("Failed to fix VNC directory ownership")
		}
	}

	return nil
}

// StopDomain 停止运行中的域（优雅关闭）
func (c *Client) StopDomain(domain libvirt.Domain) error {
	err := c.conn.DomainShutdown(domain)
	if err != nil {
		return fmt.Errorf("failed to stop domain %s: %v", domain.Name, err)
	}
	return nil
}

// RebootDomain 重启域
func (c *Client) RebootDomain(domain libvirt.Domain) error {
	err := c.conn.DomainReboot(domain, 0)
	if err != nil {
		return fmt.Errorf("failed to reboot domain %s: %v", domain.Name, err)
	}
	return nil
}

// GetDomainByName 通过名称查找域
func (c *Client) GetDomainByName(name string) (libvirt.Domain, error) {
	domain, err := c.conn.DomainLookupByName(name)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("failed to lookup domain by name %s: %v", name, err)
	}
	return domain, nil
}

// GetDomainState 获取域的状态
func (c *Client) GetDomainState(domain libvirt.Domain) (uint8, uint32, error) {
	state, reason, err := c.conn.DomainGetState(domain, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get domain state: %v", err)
	}
	return uint8(state), uint32(reason), nil
}

// DestroyDomain 强制销毁运行中的域
func (c *Client) DestroyDomain(domain libvirt.Domain) error {
	err := c.conn.DomainDestroy(domain)
	if err != nil {
		return fmt.Errorf("failed to destroy domain %s: %v", domain.Name, err)
	}
	return nil
}

// ModifyDomainMemory 修改域的内存大小
// memoryKB: 新的内存大小（KB）
// live: true=热修改（如果域正在运行），false=仅修改配置（需要重启生效）
func (c *Client) ModifyDomainMemory(domain libvirt.Domain, memoryKB uint64, live bool) error {
	// 获取持久化配置 XML（使用 DomainXMLInactive 标志）
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, libvirt.DomainXMLInactive)
	if err != nil {
		return fmt.Errorf("get domain XML: %w", err)
	}

	// 解析 XML
	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		return fmt.Errorf("unmarshal domain XML: %w", err)
	}

	// 修改内存
	domainXML.Memory.Value = memoryKB
	domainXML.CurrentMemory.Value = memoryKB

	// 重新序列化 XML
	xmlBytes, err := xml.MarshalIndent(&domainXML, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal domain XML: %w", err)
	}

	// 更新持久化配置
	_, err = c.conn.DomainDefineXML(string(xmlBytes))
	if err != nil {
		return fmt.Errorf("define domain with new memory: %w", err)
	}

	// 如果请求热修改且域正在运行，尝试立即生效
	if live {
		state, _, err := c.conn.DomainGetState(domain, 0)
		if err == nil && libvirt.DomainState(state) == libvirt.DomainRunning {
			// 尝试热修改内存（可能会失败，某些虚拟机不支持）
			_ = c.conn.DomainSetMemory(domain, memoryKB)
		}
	}

	return nil
}

// ModifyDomainVCPU 修改域的 VCPU 数量
// vcpus: 新的 VCPU 数量
// live: true=热修改（如果域正在运行），false=仅修改配置（需要重启生效）
func (c *Client) ModifyDomainVCPU(domain libvirt.Domain, vcpus uint16, live bool) error {
	// 获取持久化配置 XML（使用 DomainXMLInactive 标志）
	xmlDesc, err := c.conn.DomainGetXMLDesc(domain, libvirt.DomainXMLInactive)
	if err != nil {
		return fmt.Errorf("get domain XML: %w", err)
	}

	// 解析 XML
	var domainXML DomainXML
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		return fmt.Errorf("unmarshal domain XML: %w", err)
	}

	// 修改 VCPU
	domainXML.VCPU.Value = int(vcpus)

	// 重新序列化 XML
	xmlBytes, err := xml.MarshalIndent(&domainXML, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal domain XML: %w", err)
	}

	// 更新持久化配置
	_, err = c.conn.DomainDefineXML(string(xmlBytes))
	if err != nil {
		return fmt.Errorf("define domain with new VCPU: %w", err)
	}

	// 如果请求热修改且域正在运行，尝试立即生效
	if live {
		state, _, err := c.conn.DomainGetState(domain, 0)
		if err == nil && libvirt.DomainState(state) == libvirt.DomainRunning {
			// 尝试热修改 VCPU（可能会失败，某些虚拟机不支持）
			_ = c.conn.DomainSetVcpusFlags(domain, uint32(vcpus), uint32(libvirt.DomainVCPULive))
		}
	}

	return nil
}

// SetDomainAutostart 设置域的自动启动状态
// autostart: true=开机自动启动，false=禁用自动启动
func (c *Client) SetDomainAutostart(domain libvirt.Domain, autostart bool) error {
	var value int32 = 0
	if autostart {
		value = 1
	}
	if err := c.conn.DomainSetAutostart(domain, value); err != nil {
		return fmt.Errorf("failed to set domain autostart: %w", err)
	}
	return nil
}

// DeleteDomain 删除域
// flags: 可以组合以下标志
//   - libvirt.DomainUndefineManagedSave: 同时删除托管保存的镜像
//   - libvirt.DomainUndefineSnapshotsMetadata: 同时删除快照元数据
//   - libvirt.DomainUndefineNvram: 同时删除 NVRAM 文件
func (c *Client) DeleteDomain(domain libvirt.Domain, flags libvirt.DomainUndefineFlagsValues) error {
	// 检查域是否在运行
	state, _, err := c.conn.DomainGetState(domain, 0)
	if err != nil {
		return fmt.Errorf("failed to get domain state: %v", err)
	}

	// 如果域正在运行，先强制关闭
	if libvirt.DomainState(state) == libvirt.DomainRunning {
		if err := c.conn.DomainDestroy(domain); err != nil {
			return fmt.Errorf("failed to destroy running domain: %v", err)
		}
	}

	// 删除域定义
	err = c.conn.DomainUndefineFlags(domain, flags)
	if err != nil {
		return fmt.Errorf("failed to undefine domain: %v", err)
	}

	return nil
}

// CreateDomain 创建并可选启动虚拟机域
// autoStart: true=立即启动域，false=仅定义不启动
// 这是一个便捷方法，组合了 DefineDomain 和 StartDomain
func (c *Client) CreateDomain(config *CreateVMConfig, autoStart bool) (libvirt.Domain, error) {
	// 定义域
	domain, err := c.DefineDomain(config)
	if err != nil {
		return libvirt.Domain{}, err
	}

	// 如果需要自动启动
	if autoStart {
		if err := c.StartDomain(domain); err != nil {
			// 启动失败，尝试清理已定义的域
			_ = c.conn.DomainUndefine(domain)
			return libvirt.Domain{}, fmt.Errorf("failed to start domain after definition: %v", err)
		}
	}

	return domain, nil
}

// ============================================================================
// 辅助方法
// ============================================================================

// validateVMConfig 验证虚拟机配置参数
func (c *Client) validateVMConfig(config *CreateVMConfig) error {
	if config.Name == "" {
		return fmt.Errorf("domain name is required")
	}

	if config.Memory == 0 {
		return fmt.Errorf("memory size is required and must be greater than 0")
	}

	if config.VCPUs == 0 {
		return fmt.Errorf("vCPU count is required and must be greater than 0")
	}

	if config.DiskPath == "" {
		return fmt.Errorf("disk path is required")
	}

	return nil
}

// setDefaultVMConfig 设置配置的默认值
func (c *Client) setDefaultVMConfig(config *CreateVMConfig) {
	if config.DiskBus == "" {
		config.DiskBus = "virtio"
	}

	if config.NetworkType == "" {
		config.NetworkType = "bridge"
	}

	if config.NetworkSource == "" {
		config.NetworkSource = "br0"
	}

	if config.OSType == "" {
		config.OSType = "hvm"
	}

	if config.Architecture == "" {
		config.Architecture = "x86_64"
	}

	if config.VNCSocket == "" {
		config.VNCSocket = "/var/lib/jvp/qemu/" + config.Name + ".vnc"
	}
}

// buildDomainXML 根据配置构建 DomainXML 结构
func (c *Client) buildDomainXML(config *CreateVMConfig) (*DomainXML, error) {
	domain := &DomainXML{
		Type: "kvm",
		Name: config.Name,
		Memory: DomainMemory{
			Unit:  "KiB",
			Value: config.Memory,
		},
		CurrentMemory: DomainMemory{
			Unit:  "KiB",
			Value: config.Memory,
		},
		VCPU: DomainVCPU{
			Placement: "static",
			Value:     int(config.VCPUs),
		},
		OS: DomainOS{
			Type: DomainOSType{
				Arch:    config.Architecture,
				Machine: config.MachineType,
				Value:   config.OSType,
			},
			Boot: DomainBoot{
				Dev: "hd",
			},
		},
		Features: &DomainFeatures{
			ACPI: &DomainFeatureEnabled{},
			APIC: &DomainFeatureEnabled{},
		},
		Clock: &DomainClock{
			Offset: "utc",
		},
		OnPoweroff: "destroy",
		OnReboot:   "restart",
		OnCrash:    "destroy",
		Devices:    c.buildDevices(config),
	}

	return domain, nil
}

// buildDevices 构建设备配置
func (c *Client) buildDevices(config *CreateVMConfig) DomainDevices {
	// 构建网络接口配置
	var netSource DomainInterfaceSource
	switch config.NetworkType {
	case "bridge":
		netSource = DomainInterfaceSource{
			Bridge: config.NetworkSource,
		}
	case "network":
		netSource = DomainInterfaceSource{
			Network: config.NetworkSource,
		}
	case "direct":
		netSource = DomainInterfaceSource{
			Dev:  config.NetworkSource,
			Mode: "bridge",
		}
	default:
		// 默认使用 bridge
		netSource = DomainInterfaceSource{
			Bridge: config.NetworkSource,
		}
	}

	devices := DomainDevices{
		Emulator: "/usr/bin/qemu-system-" + config.Architecture,
		Disks:    c.buildDisks(config),
		Interfaces: []DomainInterface{
			{
				Type:   config.NetworkType,
				Source: netSource,
				Model: DomainInterfaceModel{
					Type: "virtio",
				},
			},
		},
		Graphics: DomainGraphics{
			Type:   "vnc",
			Socket: config.VNCSocket,
		},
		Serial: DomainSerial{
			Type: "pty",
			Target: DomainSerialTarget{
				Type: "isa-serial",
				Port: 0,
				Model: DomainSerialTargetModel{
					Name: "isa-serial",
				},
			},
		},
		Console: DomainConsole{
			Type: "pty",
			Target: DomainConsoleTarget{
				Type: "serial",
				Port: 0,
			},
		},
		Controllers: []DomainController{
			{
				Type:  "usb",
				Index: 0,
				Model: "ich9-ehci1",
			},
			{
				Type:  "pci",
				Index: 0,
				Model: "pci-root",
			},
		},
		Videos: []DomainVideo{
			{
				Model: DomainVideoModel{
					Type:  "virtio",
					Heads: 1,
				},
			},
		},
		Inputs: []DomainInput{
			{
				Type: "tablet",
				Bus:  "usb",
			},
			{
				Type: "mouse",
				Bus:  "ps2",
			},
			{
				Type: "keyboard",
				Bus:  "ps2",
			},
		},
		MemBalloon: &DomainMemBalloon{
			Model: "virtio",
		},
		RNG: &DomainRNG{
			Model: "virtio",
			Backend: &DomainRNGBackend{
				Model: "random",
				Value: "/dev/urandom",
			},
		},
	}

	return devices
}

// fixVNCDirOwnership 修复 VNC socket 目录的所有权为 libvirt-qemu:kvm
func (c *Client) fixVNCDirOwnership(dirPath string) error {
	// 查找 libvirt-qemu 用户和 kvm 组
	libvirtUser, err := user.Lookup("libvirt-qemu")
	if err != nil {
		// 如果找不到用户，返回错误（非致命，调用方会记录警告）
		return fmt.Errorf("lookup libvirt-qemu user: %w", err)
	}

	kvmGroup, err := user.LookupGroup("kvm")
	if err != nil {
		return fmt.Errorf("lookup kvm group: %w", err)
	}

	uid, err := strconv.Atoi(libvirtUser.Uid)
	if err != nil {
		return fmt.Errorf("parse libvirt-qemu UID: %w", err)
	}

	gid, err := strconv.Atoi(kvmGroup.Gid)
	if err != nil {
		return fmt.Errorf("parse kvm GID: %w", err)
	}

	// 修改目录所有权
	return os.Chown(dirPath, uid, gid)
}

// buildDisks 构建磁盘配置
func (c *Client) buildDisks(config *CreateVMConfig) []DomainDisk {
	disks := []DomainDisk{
		{
			Type:   "file",
			Device: "disk",
			Driver: DomainDiskDriver{
				Name: "qemu",
				Type: "qcow2",
			},
			Source: DomainDiskSource{
				File: config.DiskPath,
			},
			Target: DomainDiskTarget{
				Dev: "vda",
				Bus: config.DiskBus,
			},
		},
	}

	// 如果提供了 ISO 路径，添加 CDROM 设备
	if config.ISOPath != "" {
		disks = append(disks, DomainDisk{
			Type:   "file",
			Device: "cdrom",
			Driver: DomainDiskDriver{
				Name: "qemu",
				Type: "raw",
			},
			Source: DomainDiskSource{
				File: config.ISOPath,
			},
			Target: DomainDiskTarget{
				Dev: "hda",
				Bus: "ide",
			},
		})

		// 如果有 ISO，从 CDROM 启动
		// 注意：这需要在 OS.Boot 中设置，但我们已经在 buildDomainXML 中设置了
	}

	// 如果有 cloud-init ISO，添加 CDROM 设备
	if config.cloudInitISOPath != "" {
		disks = append(disks, DomainDisk{
			Type:   "file",
			Device: "cdrom",
			Driver: DomainDiskDriver{
				Name: "qemu",
				Type: "raw",
			},
			Source: DomainDiskSource{
				File: config.cloudInitISOPath,
			},
			Target: DomainDiskTarget{
				Dev: "hdb",
				Bus: "ide",
			},
		})
	}

	return disks
}

// QemuAgentCommand 执行 QEMU Guest Agent 命令
// 使用 virsh qemu-agent-command 来执行命令
func (c *Client) QemuAgentCommand(domain libvirt.Domain, command string, timeout uint32, flags uint32) (string, error) {
	// 获取 domain 名称（domain 结构包含 Name 字段）
	domainName := domain.Name
	if domainName == "" {
		return "", fmt.Errorf("domain name is empty")
	}

	// 使用 virsh 命令执行 qemu-agent-command
	cmd := exec.Command("virsh", "qemu-agent-command", domainName, command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qemu agent command failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

// CheckGuestAgentAvailable 检查 Guest Agent 是否可用
func (c *Client) CheckGuestAgentAvailable(domain libvirt.Domain) (bool, error) {
	// 尝试执行一个简单的 ping 命令来检查 guest agent 是否可用
	pingCmd := `{"execute":"guest-ping"}`
	_, err := c.QemuAgentCommand(domain, pingCmd, 5, 0)
	if err != nil {
		return false, nil // Guest agent 不可用，但不返回错误
	}
	return true, nil
}

// DomainSnapshotXML 快照 XML 结构
type DomainSnapshotXML struct {
	XMLName      xml.Name                 `xml:"domainsnapshot"`
	Name         string                   `xml:"name"`
	State        string                   `xml:"state,omitempty"`
	CreationTime int64                    `xml:"creationTime,omitempty"`
	Description  string                   `xml:"description,omitempty"`
	Parent       *DomainSnapshotParentXML `xml:"parent,omitempty"`
	Memory       *DomainSnapshotMemoryXML `xml:"memory,omitempty"`
	Disks        []DomainSnapshotDiskXML  `xml:"disks>disk,omitempty"`
}

type DomainSnapshotParentXML struct {
	Name string `xml:"name"`
}

type DomainSnapshotMemoryXML struct {
	Snapshot string `xml:"snapshot,attr,omitempty"`
	File     string `xml:"file,attr,omitempty"`
}

type DomainSnapshotDiskXML struct {
	XMLName  xml.Name                     `xml:"disk"`
	Name     string                       `xml:"name,attr"`
	Snapshot string                       `xml:"snapshot,attr,omitempty"`
	Driver   *DomainSnapshotDiskDriverXML `xml:"driver,omitempty"`
	Source   *DomainSnapshotDiskSourceXML `xml:"source,omitempty"`
}

type DomainSnapshotDiskDriverXML struct {
	Type string `xml:"type,attr,omitempty"`
}

type DomainSnapshotDiskSourceXML struct {
	File string `xml:"file,attr,omitempty"`
}

// ListSnapshots 列出域的所有快照名称
func (c *Client) ListSnapshots(domainName string) ([]string, error) {
	snapshots, err := c.ListSnapshotXML(domainName)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Name != "" {
			names = append(names, snapshot.Name)
		}
	}
	return names, nil
}

// ListSnapshotXML 获取域的快照 XML 信息
func (c *Client) ListSnapshotXML(domainName string) ([]DomainSnapshotXML, error) {
	// 获取 domain
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %s: %w", domainName, err)
	}

	// 使用 DomainListAllSnapshots 获取所有快照
	snapshots, _, err := c.conn.DomainListAllSnapshots(domain, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("list snapshots for domain %s: %w", domainName, err)
	}

	// 提取快照名称
	result := make([]DomainSnapshotXML, 0, len(snapshots))
	for _, snapshot := range snapshots {
		// 获取快照的 XML 描述
		xmlDesc, err := c.conn.DomainSnapshotGetXMLDesc(snapshot, 0)
		if err != nil {
			continue
		}

		// 解析 XML
		var snapshotXML DomainSnapshotXML
		if err := xml.Unmarshal([]byte(xmlDesc), &snapshotXML); err != nil {
			continue
		}

		if snapshotXML.Name != "" {
			result = append(result, snapshotXML)
		}
	}

	return result, nil
}

// GetSnapshotXML 获取指定快照的 XML 信息
func (c *Client) GetSnapshotXML(domainName, snapshotName string) (*DomainSnapshotXML, error) {
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %s: %w", domainName, err)
	}

	snapshot, err := c.conn.DomainSnapshotLookupByName(domain, snapshotName, 0)
	if err != nil {
		return nil, fmt.Errorf("lookup snapshot %s for domain %s: %w", snapshotName, domainName, err)
	}

	xmlDesc, err := c.conn.DomainSnapshotGetXMLDesc(snapshot, 0)
	if err != nil {
		return nil, fmt.Errorf("get snapshot XML for %s: %w", snapshotName, err)
	}

	var snapshotXML DomainSnapshotXML
	if err := xml.Unmarshal([]byte(xmlDesc), &snapshotXML); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot XML for %s: %w", snapshotName, err)
	}

	return &snapshotXML, nil
}

// CreateSnapshot 创建快照
func (c *Client) CreateSnapshot(domainName string, snapshotXML string, flags libvirt.DomainSnapshotCreateFlags) error {
	// 获取 domain
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return fmt.Errorf("lookup domain %s: %w", domainName, err)
	}

	if _, err := c.conn.DomainSnapshotCreateXML(domain, snapshotXML, uint32(flags)); err != nil {
		return fmt.Errorf("create snapshot for domain %s: %w", domainName, err)
	}
	return nil
}

// DeleteSnapshot 删除快照
func (c *Client) DeleteSnapshot(domainName, snapshotName string, flags libvirt.DomainSnapshotDeleteFlags) error {
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return fmt.Errorf("lookup domain %s: %w", domainName, err)
	}

	snapshot, err := c.conn.DomainSnapshotLookupByName(domain, snapshotName, 0)
	if err != nil {
		return fmt.Errorf("lookup snapshot %s for domain %s: %w", snapshotName, domainName, err)
	}

	if err := c.conn.DomainSnapshotDelete(snapshot, flags); err != nil {
		return fmt.Errorf("delete snapshot %s for domain %s: %w", snapshotName, domainName, err)
	}
	return nil
}

// RevertToSnapshot 回滚快照
func (c *Client) RevertToSnapshot(domainName, snapshotName string, flags libvirt.DomainSnapshotRevertFlags) error {
	domain, err := c.conn.DomainLookupByName(domainName)
	if err != nil {
		return fmt.Errorf("lookup domain %s: %w", domainName, err)
	}

	snapshot, err := c.conn.DomainSnapshotLookupByName(domain, snapshotName, 0)
	if err != nil {
		return fmt.Errorf("lookup snapshot %s for domain %s: %w", snapshotName, domainName, err)
	}

	if err := c.conn.DomainRevertToSnapshot(snapshot, uint32(flags)); err != nil {
		return fmt.Errorf("revert to snapshot %s for domain %s: %w", snapshotName, domainName, err)
	}
	return nil
}

// ListInterfaces 列出所有网络接口
func (c *Client) ListInterfaces() ([]libvirt.Interface, error) {
	// 获取所有活动的网络接口
	activeIfaces, _, err := c.conn.ConnectListAllInterfaces(1, libvirt.ConnectListInterfacesActive)
	if err != nil {
		return nil, fmt.Errorf("failed to list active interfaces: %w", err)
	}

	// 获取所有非活动的网络接口
	inactiveIfaces, _, err := c.conn.ConnectListAllInterfaces(1, libvirt.ConnectListInterfacesInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to list inactive interfaces: %w", err)
	}

	// 合并结果
	allIfaces := append(activeIfaces, inactiveIfaces...)
	return allIfaces, nil
}

// GetInterfaceXMLDesc 获取网络接口的 XML 描述
func (c *Client) GetInterfaceXMLDesc(iface libvirt.Interface) (string, error) {
	xmlDesc, err := c.conn.InterfaceGetXMLDesc(iface, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get interface XML: %w", err)
	}
	return xmlDesc, nil
}

// ListNetworkDHCPLeases 获取指定网络的 DHCP 租约
func (c *Client) ListNetworkDHCPLeases(networkName string) ([]DHCPLease, error) {
	network, err := c.conn.NetworkLookupByName(networkName)
	if err != nil {
		return nil, fmt.Errorf("lookup network %s: %w", networkName, err)
	}
	leases, _, err := c.conn.NetworkGetDhcpLeases(network, libvirt.OptString{}, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get DHCP leases: %w", err)
	}

	result := make([]DHCPLease, 0, len(leases))
	for _, l := range leases {
		result = append(result, DHCPLease{
			IP:        l.Ipaddr,
			MACs:      l.Mac,
			Hostnames: l.Hostname,
			Expiry:    l.Expirytime,
			ClientIDs: l.Clientid,
			IAIDs:     l.Iaid,
		})
	}
	return result, nil
}

// ListNetworks 列出所有网络名称
func (c *Client) ListNetworks() ([]string, error) {
	nets, _, err := c.conn.ConnectListAllNetworks(0, 0)
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	names := make([]string, 0, len(nets))
	for _, n := range nets {
		names = append(names, n.Name)
	}
	return names, nil
}

// ListNodeDevices 列出指定类型的节点设备
// cap 参数可以是："pci", "usb", "storage", "net" 等
func (c *Client) ListNodeDevices(cap string) ([]libvirt.NodeDevice, error) {
	devices, _, err := c.conn.ConnectListAllNodeDevices(1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list node devices: %w", err)
	}

	// 如果没有指定 capability，返回所有设备
	if cap == "" {
		return devices, nil
	}

	// 过滤指定类型的设备
	filtered := make([]libvirt.NodeDevice, 0)
	for _, dev := range devices {
		// 获取设备 XML 描述
		xmlDesc, err := c.GetNodeDeviceXMLDesc(dev)
		if err != nil {
			continue
		}

		// 解析 XML 来判断设备类型
		deviceXML, err := ParseNodeDeviceXML(xmlDesc)
		if err != nil {
			continue
		}

		// 根据指定的类型过滤
		matched := false
		switch cap {
		case "pci":
			matched = deviceXML.IsPCIDevice()
		case "usb":
			matched = deviceXML.IsUSBDevice()
		case "net":
			matched = deviceXML.IsNetworkInterface()
		case "storage":
			matched = deviceXML.IsStorageDevice()
		}

		if matched {
			filtered = append(filtered, dev)
		}
	}

	return filtered, nil
}

// GetNodeDeviceXMLDesc 获取节点设备的 XML 描述
func (c *Client) GetNodeDeviceXMLDesc(dev libvirt.NodeDevice) (string, error) {
	xmlDesc, err := c.conn.NodeDeviceGetXMLDesc(dev.Name, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get node device XML: %w", err)
	}
	return xmlDesc, nil
}
