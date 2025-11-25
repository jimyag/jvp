package service

import (
	"context"
	"fmt"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
)

// NodeService 节点管理服务
type NodeService struct {
	libvirtClient libvirt.LibvirtClient
}

// NewNodeService 创建节点服务
func NewNodeService(libvirtClient libvirt.LibvirtClient) (*NodeService, error) {
	return &NodeService{
		libvirtClient: libvirtClient,
	}, nil
}

// ListNodes 列举节点
// 单机部署时只返回本地节点
func (s *NodeService) ListNodes(ctx context.Context) ([]*entity.Node, error) {
	// 获取 libvirt 主机名
	hostname, err := s.libvirtClient.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// 获取 libvirt 版本来验证连接
	version, err := s.libvirtClient.GetLibvirtVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt version: %w", err)
	}

	// 创建节点信息
	node := &entity.Node{
		Name:  hostname,
		UUID:  fmt.Sprintf("%s-node", hostname),
		URI:   fmt.Sprintf("libvirt-%s", version),
		Type:  entity.NodeTypeLocal,
		State: entity.NodeStateOnline,
	}

	return []*entity.Node{node}, nil
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
	// 获取 capabilities XML（包含详细的 CPU、内存、NUMA 信息）
	capsXML, err := s.libvirtClient.GetCapabilities()
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities: %w", err)
	}

	// 解析 capabilities XML
	caps, err := libvirt.ParseCapabilities(capsXML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse capabilities: %w", err)
	}

	// 获取 sysinfo XML（包含真实的 CPU 型号）
	sysinfoXML, err := s.libvirtClient.GetSysinfo()
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
	// TODO: 实现 PCI 设备查询
	// 需要通过 sysfs 或其他方式获取 PCI 设备信息
	return []entity.PCIDevice{}, nil
}

// DescribeNodeUSB 查询节点 USB 设备
func (s *NodeService) DescribeNodeUSB(ctx context.Context, nodeName string) ([]entity.USBDevice, error) {
	// TODO: 实现 USB 设备查询
	return []entity.USBDevice{}, nil
}

// DescribeNodeNet 查询节点网络接口
func (s *NodeService) DescribeNodeNet(ctx context.Context, nodeName string) (*NodeNetworkInfo, error) {
	// TODO: 实现网络接口查询
	return &NodeNetworkInfo{
		Interfaces: []entity.NetworkInterface{},
		Bridges:    []entity.Bridge{},
		Bonds:      []entity.Bond{},
		SRIOV:      []entity.SRIOVInfo{},
	}, nil
}

// DescribeNodeDisks 查询节点物理磁盘
func (s *NodeService) DescribeNodeDisks(ctx context.Context, nodeName string) ([]entity.Disk, error) {
	// TODO: 实现物理磁盘查询
	return []entity.Disk{}, nil
}

// EnableNode 启用节点
func (s *NodeService) EnableNode(ctx context.Context, nodeName string) error {
	// TODO: 实现节点启用逻辑（如有需要）
	return nil
}

// DisableNode 禁用节点（进入维护模式）
func (s *NodeService) DisableNode(ctx context.Context, nodeName string) error {
	// TODO: 实现节点禁用逻辑（如有需要）
	return nil
}

// NodeNetworkInfo 节点网络信息
type NodeNetworkInfo struct {
	Interfaces []entity.NetworkInterface `json:"interfaces"`
	Bridges    []entity.Bridge           `json:"bridges"`
	Bonds      []entity.Bond             `json:"bonds"`
	SRIOV      []entity.SRIOVInfo        `json:"sriov"`
}
