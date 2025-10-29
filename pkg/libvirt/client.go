package libvirt

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/pkg/cloudinit"
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
	VNCSocket         string              // VNC Unix socket 路径（可选，默认：/var/lib/libvirt/qemu/{name}.vnc）
	Autostart         bool                // 是否开机自动启动（默认：false）
	CloudInit         *cloudinit.Config   // cloud-init 配置（可选）
	CloudInitUserData *cloudinit.UserData // cloud-init 用户数据（可选）
	cloudInitISOPath  string              // cloud-init ISO 路径（内部使用）
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
			Name:  iface.Source.Dev,
			Type:  iface.Type,
			MAC:   iface.Source.MAC,
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

	// 如果配置了 cloud-init，生成 cloud-init ISO
	if config.CloudInit != nil {
		isoPath, err := c.generateCloudInitISO(config)
		if err != nil {
			return libvirt.Domain{}, fmt.Errorf("failed to generate cloud-init ISO: %v", err)
		}
		config.cloudInitISOPath = isoPath
		log.Printf("Generated cloud-init ISO: %s", config.cloudInitISOPath)
	}
	if config.CloudInitUserData != nil {
		isoPath, err := c.generateCloudInitISO(config)
		if err != nil {
			return libvirt.Domain{}, fmt.Errorf("failed to generate cloud-init ISO: %v", err)
		}
		config.cloudInitISOPath = isoPath
		log.Printf("Generated cloud-init ISO: %s", config.cloudInitISOPath)
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
			log.Printf("Warning: failed to set autostart for domain %s: %v", config.Name, err)
		}
	}

	return domain, nil
}

// StartDomain 启动已定义的域
func (c *Client) StartDomain(domain libvirt.Domain) error {
	err := c.conn.DomainCreate(domain)
	if err != nil {
		return fmt.Errorf("failed to start domain %s: %v", domain.Name, err)
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
		log.Printf("Domain %s was running and has been forcefully stopped", domain.Name)
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
		config.VNCSocket = "/var/lib/libvirt/qemu/" + config.Name + ".vnc"
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
					Type:  "qxl",
					VRam:  65536,
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
