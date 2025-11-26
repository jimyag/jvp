package entity

import "time"

// NodeType 节点类型
type NodeType string

const (
	NodeTypeLocal   NodeType = "local"   // 本地节点
	NodeTypeRemote  NodeType = "remote"  // 远程节点
	NodeTypeCompute NodeType = "compute" // 计算节点
	NodeTypeStorage NodeType = "storage" // 存储节点
	NodeTypeHybrid  NodeType = "hybrid"  // 混合节点
)

// NodeState 节点状态
type NodeState string

const (
	NodeStateOnline      NodeState = "online"      // 在线
	NodeStateOffline     NodeState = "offline"     // 离线
	NodeStateMaintenance NodeState = "maintenance" // 维护模式
)

// Node 节点信息
type Node struct {
	Name      string    `json:"name"`       // 节点名称
	UUID      string    `json:"uuid"`       // 节点 UUID
	URI       string    `json:"uri"`        // Libvirt 连接 URI
	Type      NodeType  `json:"type"`       // 节点类型
	State     NodeState `json:"state"`      // 节点状态
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// NodeSummary 节点概要信息
type NodeSummary struct {
	CPU            CPUInfo            `json:"cpu"`
	Memory         MemoryInfo         `json:"memory"`
	NUMA           NUMAInfo           `json:"numa"`
	HugePages      HugePagesInfo      `json:"hugepages"`
	Virtualization VirtualizationInfo `json:"virtualization"`
}

// CPUInfo CPU 信息
type CPUInfo struct {
	Cores     int      `json:"cores"`      // 物理核心数
	Threads   int      `json:"threads"`    // 逻辑线程数
	Model     string   `json:"model"`      // CPU 型号
	Vendor    string   `json:"vendor"`     // CPU 厂商
	Frequency int      `json:"frequency"`  // 主频 (MHz)
	Arch      string   `json:"arch"`       // 架构
	CacheSize int      `json:"cache_size"` // 缓存大小 (KB)
	Flags     []string `json:"flags"`      // CPU 特性标志
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Total        int64   `json:"total"`         // 总内存 (bytes)
	Available    int64   `json:"available"`     // 可用内存 (bytes)
	Used         int64   `json:"used"`          // 已用内存 (bytes)
	UsagePercent float64 `json:"usage_percent"` // 使用率 (%)
	SwapTotal    int64   `json:"swap_total"`    // Swap 总量 (bytes)
	SwapUsed     int64   `json:"swap_used"`     // Swap 已用 (bytes)
}

// NUMAInfo NUMA 信息
type NUMAInfo struct {
	NodeCount int        `json:"node_count"` // NUMA 节点数量
	Nodes     []NUMANode `json:"nodes"`      // NUMA 节点列表
}

// NUMANode NUMA 节点
type NUMANode struct {
	ID       int   `json:"id"`       // NUMA 节点 ID
	CPUs     []int `json:"cpus"`     // CPU 列表
	Memory   int64 `json:"memory"`   // 内存大小 (bytes)
	Distance []int `json:"distance"` // 到其他节点的距离
}

// HugePagesInfo 大页内存信息
type HugePagesInfo struct {
	Enabled   bool           `json:"enabled"`    // 是否启用
	PageSizes []HugePageSize `json:"page_sizes"` // 页面大小列表
}

// HugePageSize 大页内存大小配置
type HugePageSize struct {
	Size  string `json:"size"`  // 页面大小 (2MB/1GB)
	Total int    `json:"total"` // 总数
	Free  int    `json:"free"`  // 空闲数
	Used  int    `json:"used"`  // 已用数
}

// VirtualizationInfo 虚拟化特性
type VirtualizationInfo struct {
	VTx        bool `json:"vtx"`         // VT-x / AMD-V
	EPT        bool `json:"ept"`         // EPT / NPT
	IOMMU      bool `json:"iommu"`       // IOMMU (VT-d / AMD-Vi)
	NestedVirt bool `json:"nested_virt"` // 嵌套虚拟化
}

// PCIDevice PCI 设备
type PCIDevice struct {
	Address     string `json:"address"`     // PCI 地址
	Vendor      string `json:"vendor"`      // 厂商
	Device      string `json:"device"`      // 设备型号
	Class       string `json:"class"`       // 设备类型
	Driver      string `json:"driver"`      // 驱动程序
	IOMMUGroup  int    `json:"iommu_group"` // IOMMU 组
	Passthrough bool   `json:"passthrough"` // 是否支持直通
	InUse       bool   `json:"in_use"`      // 是否正在使用
}

// GPUDevice GPU 设备
type GPUDevice struct {
	PCIDevice
	Memory int `json:"memory"` // 显存大小 (MB)
}

// NetworkController 网卡设备
type NetworkController struct {
	PCIDevice
	Speed    string `json:"speed"`     // 速率 (1G/10G/25G)
	SRIOVCap bool   `json:"sriov_cap"` // 是否支持 SR-IOV
	VFCount  int    `json:"vf_count"`  // VF 数量
	MaxVFs   int    `json:"max_vfs"`   // 最大 VF 数量
}

// USBDevice USB 设备
type USBDevice struct {
	Bus         int    `json:"bus"`         // USB 总线
	Device      int    `json:"device"`      // 设备号
	VendorID    string `json:"vendor_id"`   // 厂商 ID
	ProductID   string `json:"product_id"`  // 产品 ID
	Vendor      string `json:"vendor"`      // 厂商名称
	Product     string `json:"product"`     // 产品名称
	Passthrough bool   `json:"passthrough"` // 是否支持直通
}

// NetworkInterface 网络接口
type NetworkInterface struct {
	Name      string   `json:"name"`       // 接口名称
	MAC       string   `json:"mac"`        // MAC 地址
	Speed     string   `json:"speed"`      // 速率
	Duplex    string   `json:"duplex"`     // 双工模式
	State     string   `json:"state"`      // 状态 (up/down)
	IP        []string `json:"ip"`         // IP 地址列表
	RXBytes   int64    `json:"rx_bytes"`   // 接收字节数
	TXBytes   int64    `json:"tx_bytes"`   // 发送字节数
	RXPackets int64    `json:"rx_packets"` // 接收包数
	TXPackets int64    `json:"tx_packets"` // 发送包数
	Errors    int64    `json:"errors"`     // 错误数
	Dropped   int64    `json:"dropped"`    // 丢包数
}

// Bridge 网桥
type Bridge struct {
	Name       string   `json:"name"`       // 网桥名称
	Interfaces []string `json:"interfaces"` // 绑定的物理接口
	STP        bool     `json:"stp"`        // STP 状态
}

// Bond 绑定接口
type Bond struct {
	Name        string   `json:"name"`         // 绑定名称
	Mode        string   `json:"mode"`         // 绑定模式
	Slaves      []string `json:"slaves"`       // 成员接口
	ActiveSlave string   `json:"active_slave"` // 当前活动接口
}

// SRIOVInfo SR-IOV 配置
type SRIOVInfo struct {
	PF       string `json:"pf"`         // 物理功能
	VFCount  int    `json:"vf_count"`   // VF 数量
	MaxVFs   int    `json:"max_vfs"`    // 最大 VF 数量
	VFsInUse int    `json:"vfs_in_use"` // 已分配 VF 数量
}

// Disk 物理磁盘
type Disk struct {
	Name       string      `json:"name"`       // 设备名称 (sda/nvme0n1)
	Type       string      `json:"type"`       // 类型 (HDD/SSD/NVMe)
	Size       int64       `json:"size"`       // 容量 (bytes)
	Model      string      `json:"model"`      // 型号
	Serial     string      `json:"serial"`     // 序列号
	Firmware   string      `json:"firmware"`   // 固件版本
	RPM        int         `json:"rpm"`        // 转速 (HDD)
	Interface  string      `json:"interface"`  // 接口类型 (SATA/SAS/NVMe)
	SMART      SMARTInfo   `json:"smart"`      // SMART 信息
	InUse      bool        `json:"in_use"`     // 是否被使用
	Partitions []Partition `json:"partitions"` // 分区列表
}

// SMARTInfo SMART 健康信息
type SMARTInfo struct {
	Health       string  `json:"health"`         // 健康状态
	Temperature  int     `json:"temperature"`    // 温度 (°C)
	PowerOnHours int64   `json:"power_on_hours"` // 通电时间
	ReadCount    int64   `json:"read_count"`     // 读取次数
	WriteCount   int64   `json:"write_count"`    // 写入次数
	WearLevel    float64 `json:"wear_level"`     // 剩余寿命 (%)
}

// Partition 分区
type Partition struct {
	Name       string `json:"name"`        // 分区名称
	Size       int64  `json:"size"`        // 大小 (bytes)
	Filesystem string `json:"filesystem"`  // 文件系统类型
	MountPoint string `json:"mount_point"` // 挂载点
	InUse      bool   `json:"in_use"`      // 是否被 Storage Pool 使用
}
