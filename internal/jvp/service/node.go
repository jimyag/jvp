package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
)

// NodeService 节点管理服务
type NodeService struct {
	storage *NodeStorage
}

// NewNodeService 创建节点服务
func NewNodeService(storage *NodeStorage) (*NodeService, error) {
	return &NodeService{
		storage: storage,
	}, nil
}

// ListNodes 列举节点
func (s *NodeService) ListNodes(ctx context.Context) ([]*entity.Node, error) {
	// 从存储获取所有节点配置
	configs, err := s.storage.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodes := make([]*entity.Node, 0, len(configs))
	for _, config := range configs {
		// 如果节点被手动设置为维护模式，直接使用该状态
		if config.State == entity.NodeStateMaintenance {
			node := &entity.Node{
				Name:      config.Name,
				UUID:      fmt.Sprintf("%s-node", config.Name),
				URI:       config.URI,
				Type:      config.Type,
				State:     entity.NodeStateMaintenance,
				CreatedAt: config.CreatedAt,
				UpdatedAt: config.UpdatedAt,
			}
			nodes = append(nodes, node)
			continue
		}

		// 尝试连接以检查状态
		state := entity.NodeStateOffline
		var hostname string

		conn, err := s.storage.GetConnection(config.Name)
		if err == nil {
			// 连接成功，获取实际的 hostname
			hostname, err = conn.GetHostname()
			if err == nil {
				state = entity.NodeStateOnline
			}
		}

		// 如果无法获取 hostname，使用配置的名称
		if hostname == "" {
			hostname = config.Name
		}

		node := &entity.Node{
			Name:      hostname,
			UUID:      fmt.Sprintf("%s-node", config.Name),
			URI:       config.URI,
			Type:      config.Type,
			State:     state,
			CreatedAt: config.CreatedAt,
			UpdatedAt: config.UpdatedAt,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// DescribeNode 查询节点详情
func (s *NodeService) DescribeNode(ctx context.Context, nodeName string) (*entity.Node, error) {
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.Name == nodeName {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", nodeName)
}

// DescribeNodeSummary 查询节点概要信息
func (s *NodeService) DescribeNodeSummary(ctx context.Context, nodeName string) (*entity.NodeSummary, error) {
	// 获取节点的 libvirt 连接
	conn, err := s.storage.GetConnection(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node connection: %w", err)
	}

	// 获取 capabilities XML（包含详细的 CPU、内存、NUMA 信息）
	capsXML, err := conn.GetCapabilities()
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities: %w", err)
	}

	// 解析 capabilities XML
	caps, err := libvirt.ParseCapabilities(capsXML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse capabilities: %w", err)
	}

	// 获取 sysinfo XML（包含真实的 CPU 型号）
	sysinfoXML, err := conn.GetSysinfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get sysinfo: %w", err)
	}

	// 解析 sysinfo XML
	sysinfo, err := libvirt.ParseSysinfo(sysinfoXML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sysinfo: %w", err)
	}

	// 提取 CPU 特性/flags
	flags := make([]string, 0, len(caps.Host.CPU.Features))
	for _, feature := range caps.Host.CPU.Features {
		flags = append(flags, feature.Name)
	}

	// 解析 CPU 频率（从 Hz 转换为 MHz）
	frequency := 0
	if caps.Host.CPU.Counter.Frequency != "" {
		// frequency 格式: "3293797000" (Hz)
		var freqHz int64
		fmt.Sscanf(caps.Host.CPU.Counter.Frequency, "%d", &freqHz)
		frequency = int(freqHz / 1000000) // 转换为 MHz
	}

	// 解析 cache 大小
	cacheSize := 0
	if len(caps.Host.Cache.Banks) > 0 {
		// 取 L3 cache (最大的那个)
		for _, bank := range caps.Host.Cache.Banks {
			var size int
			unit := bank.Unit
			fmt.Sscanf(bank.Size, "%d", &size)
			if unit == "MiB" {
				cacheSize = size * 1024 // 转换为 KiB
			} else if unit == "KiB" {
				cacheSize = size
			}
		}
	}

	// 解析 topology
	var cores, threads int
	fmt.Sscanf(caps.Host.CPU.Topology.Cores, "%d", &cores)
	fmt.Sscanf(caps.Host.CPU.Topology.Threads, "%d", &threads)

	// 从 sysinfo 获取真实的 CPU 型号
	cpuModel := sysinfo.GetProcessorVersion()
	if cpuModel == "" {
		// 如果 sysinfo 没有，使用 capabilities 的值
		cpuModel = caps.Host.CPU.Model
	}

	// 构建 CPU 信息
	cpuInfo := entity.CPUInfo{
		Cores:     cores,
		Threads:   threads,
		Model:     cpuModel, // 使用真实的 CPU 型号
		Vendor:    caps.Host.CPU.Vendor,
		Frequency: frequency,
		Arch:      caps.Host.CPU.Arch,
		CacheSize: cacheSize,
		Flags:     flags,
	}

	// 解析内存信息（从 NUMA topology 中获取）
	var totalMemoryKB int64
	if len(caps.Host.Topology.Cells.Cells) > 0 {
		for _, cell := range caps.Host.Topology.Cells.Cells {
			var memKB int64
			fmt.Sscanf(cell.Memory.Value, "%d", &memKB)
			totalMemoryKB += memKB
		}
	}
	totalMemoryBytes := totalMemoryKB * 1024

	// 构建内存信息
	memoryInfo := entity.MemoryInfo{
		Total:        totalMemoryBytes,
		Available:    totalMemoryBytes / 2, // 简化估计：假设一半可用
		Used:         totalMemoryBytes / 2,
		UsagePercent: 50.0,
		SwapTotal:    0, // capabilities 不提供 swap 信息
		SwapUsed:     0,
	}

	// 构建 NUMA 信息
	var numaNodeCount int
	fmt.Sscanf(caps.Host.Topology.Cells.Num, "%d", &numaNodeCount)

	numaNodes := make([]entity.NUMANode, 0, len(caps.Host.Topology.Cells.Cells))
	for _, cell := range caps.Host.Topology.Cells.Cells {
		var nodeID int
		var memKB int64
		fmt.Sscanf(cell.ID, "%d", &nodeID)
		fmt.Sscanf(cell.Memory.Value, "%d", &memKB)

		numaNodes = append(numaNodes, entity.NUMANode{
			ID:     nodeID,
			Memory: memKB * 1024, // 转换为 bytes
			CPUs:   []int{},      // 简化处理，不解析 CPU 列表
		})
	}

	numaInfo := entity.NUMAInfo{
		NodeCount: numaNodeCount,
		Nodes:     numaNodes,
	}

	// 构建 HugePages 信息
	hugePagesInfo := entity.HugePagesInfo{
		Enabled:   false,
		PageSizes: []entity.HugePageSize{},
	}
	// 检查是否有 huge pages
	for _, page := range caps.Host.CPU.Pages {
		var sizeKB int
		fmt.Sscanf(page.Size, "%d", &sizeKB)
		if sizeKB >= 2048 { // 2MB 或更大的页
			hugePagesInfo.Enabled = true
			// 转换为可读格式
			sizeStr := ""
			if sizeKB >= 1048576 {
				sizeStr = fmt.Sprintf("%dGB", sizeKB/1048576)
			} else if sizeKB >= 1024 {
				sizeStr = fmt.Sprintf("%dMB", sizeKB/1024)
			} else {
				sizeStr = fmt.Sprintf("%dKB", sizeKB)
			}
			hugePagesInfo.PageSizes = append(hugePagesInfo.PageSizes, entity.HugePageSize{
				Size:  sizeStr,
				Total: 0, // capabilities 不提供 total 信息
				Free:  0,
			})
		}
	}

	// 构建虚拟化信息
	virtualizationInfo := entity.VirtualizationInfo{
		VTx:        true,                              // 假设支持（因为能运行 libvirt）
		EPT:        true,                              // 假设支持
		IOMMU:      caps.Host.IOMMU.Support == "yes",  // 从 capabilities 获取
		NestedVirt: false,                             // 简化假设
	}

	summary := &entity.NodeSummary{
		CPU:            cpuInfo,
		Memory:         memoryInfo,
		NUMA:           numaInfo,
		HugePages:      hugePagesInfo,
		Virtualization: virtualizationInfo,
	}

	return summary, nil
}

// DescribeNodePCI 查询节点 PCI 设备
func (s *NodeService) DescribeNodePCI(ctx context.Context, nodeName string) ([]entity.PCIDevice, error) {
	// 获取节点的 libvirt 连接
	conn, err := s.storage.GetConnection(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node connection: %w", err)
	}

	// 先获取所有设备看看
	allDevices, err := conn.ListNodeDevices("")
	if err != nil {
		return nil, fmt.Errorf("failed to list all devices: %w", err)
	}
	fmt.Printf("Total devices in libvirt: %d\n", len(allDevices))

	// 获取所有 PCI 设备
	devices, err := conn.ListNodeDevices("pci")
	if err != nil {
		return nil, fmt.Errorf("failed to list PCI devices: %w", err)
	}

	fmt.Printf("Filtered PCI devices: %d\n", len(devices))

	pciDevices := make([]entity.PCIDevice, 0)
	for _, dev := range devices {
		// 获取设备 XML 描述
		xmlDesc, err := conn.GetNodeDeviceXMLDesc(dev)
		if err != nil {
			fmt.Printf("Failed to get XML for PCI device %s: %v\n", dev.Name, err)
			continue
		}

		// 解析 XML 获取详细信息
		deviceXML, err := libvirt.ParseNodeDeviceXML(xmlDesc)
		if err != nil {
			fmt.Printf("Failed to parse XML for PCI device %s: %v\n", dev.Name, err)
			continue
		}

		pciDevice := entity.PCIDevice{
			Address:    deviceXML.GetPCIAddress(),
			Vendor:     deviceXML.Capability.Vendor.Name,
			Device:     deviceXML.Capability.Product.Name,
			Class:      deviceXML.Capability.Product.ID,
			IOMMUGroup: deviceXML.GetIOMMUGroup(),
		}
		pciDevices = append(pciDevices, pciDevice)
	}

	return pciDevices, nil
}

// DescribeNodeUSB 查询节点 USB 设备
func (s *NodeService) DescribeNodeUSB(ctx context.Context, nodeName string) ([]entity.USBDevice, error) {
	// 获取节点的 libvirt 连接
	conn, err := s.storage.GetConnection(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node connection: %w", err)
	}

	// 获取所有 USB 设备
	devices, err := conn.ListNodeDevices("usb")
	if err != nil {
		return nil, fmt.Errorf("failed to list USB devices: %w", err)
	}

	usbDevices := make([]entity.USBDevice, 0)
	for _, dev := range devices {
		// 获取设备 XML 描述
		xmlDesc, err := conn.GetNodeDeviceXMLDesc(dev)
		if err != nil {
			continue
		}

		// 解析 XML 获取详细信息
		deviceXML, err := libvirt.ParseNodeDeviceXML(xmlDesc)
		if err != nil {
			continue
		}

		usbDevice := entity.USBDevice{
			VendorID:  deviceXML.Capability.Vendor.ID,
			ProductID: deviceXML.Capability.Product.ID,
			Vendor:    deviceXML.Capability.Vendor.Name,
			Product:   deviceXML.Capability.Product.Name,
		}
		usbDevices = append(usbDevices, usbDevice)
	}

	return usbDevices, nil
}

// DescribeNodeNet 查询节点网络接口
func (s *NodeService) DescribeNodeNet(ctx context.Context, nodeName string) (*NodeNetworkInfo, error) {
	// 获取节点的 libvirt 连接
	conn, err := s.storage.GetConnection(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node connection: %w", err)
	}

	// 通过 node devices 获取网络接口（包含更详细的信息）
	devices, err := conn.ListNodeDevices("net")
	if err != nil {
		return nil, fmt.Errorf("failed to list network devices: %w", err)
	}

	interfaces := make([]entity.NetworkInterface, 0)

	for _, dev := range devices {
		// 获取设备 XML 描述
		xmlDesc, err := conn.GetNodeDeviceXMLDesc(dev)
		if err != nil {
			continue
		}

		// 解析 XML 获取详细信息
		deviceXML, err := libvirt.ParseNodeDeviceXML(xmlDesc)
		if err != nil {
			continue
		}

		netIface := entity.NetworkInterface{
			Name:  deviceXML.Capability.Interface,
			MAC:   deviceXML.Capability.Address,
			State: deviceXML.Capability.Link.State,
			Speed: deviceXML.Capability.Link.Speed,
		}
		interfaces = append(interfaces, netIface)
	}

	return &NodeNetworkInfo{
		Interfaces: interfaces,
		Bridges:    []entity.Bridge{},
		Bonds:      []entity.Bond{},
		SRIOV:      []entity.SRIOVInfo{},
	}, nil
}

// DescribeNodeDisks 查询节点物理磁盘
func (s *NodeService) DescribeNodeDisks(ctx context.Context, nodeName string) ([]entity.Disk, error) {
	// 获取节点的 libvirt 连接
	conn, err := s.storage.GetConnection(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node connection: %w", err)
	}

	// 获取所有存储设备
	devices, err := conn.ListNodeDevices("storage")
	if err != nil {
		return nil, fmt.Errorf("failed to list storage devices: %w", err)
	}

	disks := make([]entity.Disk, 0)
	for _, dev := range devices {
		// 获取设备 XML 描述
		xmlDesc, err := conn.GetNodeDeviceXMLDesc(dev)
		if err != nil {
			continue
		}

		// 解析 XML 获取详细信息
		deviceXML, err := libvirt.ParseNodeDeviceXML(xmlDesc)
		if err != nil {
			continue
		}

		// 确保是磁盘设备（drive_type 为 disk）
		if !deviceXML.IsStorageDevice() {
			continue
		}

		// 解析容量
		var size int64
		if deviceXML.Capability.Size != "" {
			fmt.Sscanf(deviceXML.Capability.Size, "%d", &size)
		}

		// 判断磁盘类型（简化版）
		diskType := "HDD"
		if len(deviceXML.Capability.Block) > 0 {
			// 如果设备名包含 nvme，认为是 NVMe
			if len(deviceXML.Capability.Block) > 4 && deviceXML.Capability.Block[:4] == "nvme" {
				diskType = "NVMe"
			} else if len(deviceXML.Capability.Block) > 2 && deviceXML.Capability.Block[:2] == "sd" {
				// sd* 设备，可能是 HDD 或 SSD（需要进一步判断）
				diskType = "SSD/HDD"
			}
		}

		disk := entity.Disk{
			Name:   deviceXML.Capability.Block,
			Type:   diskType,
			Size:   size,
			Model:  deviceXML.Capability.Model,
			Serial: deviceXML.Capability.Serial,
		}
		disks = append(disks, disk)
	}

	return disks, nil
}

// CreateNode 创建（添加）新节点
func (s *NodeService) CreateNode(ctx context.Context, name, uri string, nodeType entity.NodeType) (*entity.Node, error) {
	// 检查节点是否已存在
	if s.storage.Exists(name) {
		return nil, fmt.Errorf("node %s already exists", name)
	}

	// 验证连接 - 尝试连接以确保 URI 有效
	conn, err := libvirt.NewWithURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", uri, err)
	}

	// 获取实际的 hostname
	hostname, err := conn.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// 创建节点配置
	now := time.Now()
	config := &NodeConfig{
		Name:      name,
		URI:       uri,
		Type:      nodeType,
		State:     entity.NodeStateOnline, // 新创建的节点默认为 online
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 保存配置
	if err := s.storage.Save(config); err != nil {
		return nil, fmt.Errorf("failed to save node config: %w", err)
	}

	// 返回节点信息
	node := &entity.Node{
		Name:      hostname,
		UUID:      fmt.Sprintf("%s-node", name),
		URI:       uri,
		Type:      nodeType,
		State:     entity.NodeStateOnline,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return node, nil
}

// DeleteNode 删除节点
func (s *NodeService) DeleteNode(ctx context.Context, nodeName string) error {
	// 检查节点是否存在
	if !s.storage.Exists(nodeName) {
		return fmt.Errorf("node %s not found", nodeName)
	}

	// 删除节点配置
	if err := s.storage.Delete(nodeName); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// EnableNode 启用节点（退出维护模式）
func (s *NodeService) EnableNode(ctx context.Context, nodeName string) error {
	// 获取节点配置
	config, err := s.storage.Get(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node config: %w", err)
	}

	// 更新状态为在线
	config.State = entity.NodeStateOnline
	config.UpdatedAt = time.Now()

	// 保存配置
	if err := s.storage.Save(config); err != nil {
		return fmt.Errorf("failed to save node config: %w", err)
	}

	return nil
}

// DisableNode 禁用节点（进入维护模式）
func (s *NodeService) DisableNode(ctx context.Context, nodeName string) error {
	// 获取节点配置
	config, err := s.storage.Get(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node config: %w", err)
	}

	// 更新状态为维护模式
	config.State = entity.NodeStateMaintenance
	config.UpdatedAt = time.Now()

	// 保存配置
	if err := s.storage.Save(config); err != nil {
		return fmt.Errorf("failed to save node config: %w", err)
	}

	return nil
}

// NodeNetworkInfo 节点网络信息
type NodeNetworkInfo struct {
	Interfaces []entity.NetworkInterface `json:"interfaces"`
	Bridges    []entity.Bridge           `json:"bridges"`
	Bonds      []entity.Bond             `json:"bonds"`
	SRIOV      []entity.SRIOVInfo        `json:"sriov"`
}
