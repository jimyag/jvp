package entity

// Snapshot 描述快照信息
type Snapshot struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	VMName      string         `json:"vm_name"`
	NodeName    string         `json:"node_name"`
	CreatedAt   string         `json:"created_at,omitempty"`
	State       string         `json:"state,omitempty"`
	Description string         `json:"description,omitempty"`
	Parent      string         `json:"parent,omitempty"`
	Memory      bool           `json:"memory"`
	DiskOnly    bool           `json:"disk_only"`
	Disks       []SnapshotDisk `json:"disks,omitempty"`
}

// SnapshotDisk 描述快照关联的磁盘路径
type SnapshotDisk struct {
	Target string `json:"target"`
	Path   string `json:"path,omitempty"`
	Format string `json:"format,omitempty"`
}

// CreateSnapshotRequest 创建快照请求
type CreateSnapshotRequest struct {
	NodeName     string `json:"node_name" binding:"required"`
	VMName       string `json:"vm_name" binding:"required"`
	SnapshotName string `json:"snapshot_name,omitempty"`
	Description  string `json:"description,omitempty"`
	WithMemory   bool   `json:"with_memory,omitempty"`
}

type CreateSnapshotResponse struct {
	Snapshot *Snapshot `json:"snapshot"`
}

// ListSnapshotsRequest 列举快照请求
type ListSnapshotsRequest struct {
	NodeName string `json:"node_name" binding:"required"`
	VMName   string `json:"vm_name" binding:"required"`
}

type ListSnapshotsResponse struct {
	Snapshots []Snapshot `json:"snapshots"`
}

// DescribeSnapshotRequest 查询快照详情请求
type DescribeSnapshotRequest struct {
	NodeName     string `json:"node_name" binding:"required"`
	VMName       string `json:"vm_name" binding:"required"`
	SnapshotName string `json:"snapshot_name" binding:"required"`
}

type DescribeSnapshotResponse struct {
	Snapshot *Snapshot `json:"snapshot"`
}

// DeleteSnapshotRequest 删除快照请求
type DeleteSnapshotRequest struct {
	NodeName        string `json:"node_name" binding:"required"`
	VMName          string `json:"vm_name" binding:"required"`
	SnapshotName    string `json:"snapshot_name" binding:"required"`
	DeleteChildren  bool   `json:"delete_children,omitempty"`
	MetadataOnly    bool   `json:"metadata_only,omitempty"`
	DisksOnly       bool   `json:"disks_only,omitempty"`
	UnsafeIgnoreAll bool   `json:"unsafe_ignore_all,omitempty"`
}

type DeleteSnapshotResponse struct {
	Message string `json:"message"`
}

// RevertSnapshotRequest 回滚到快照请求
type RevertSnapshotRequest struct {
	NodeName         string `json:"node_name" binding:"required"`
	VMName           string `json:"vm_name" binding:"required"`
	SnapshotName     string `json:"snapshot_name" binding:"required"`
	StartAfterRevert bool   `json:"start_after_revert,omitempty"`
	Force            bool   `json:"force,omitempty"`
}

type RevertSnapshotResponse struct {
	Message string `json:"message"`
}

// CloneFromSnapshotRequest 基于快照克隆创建新实例
type CloneFromSnapshotRequest struct {
	NodeName       string `json:"node_name" binding:"required"`        // 节点名称
	SourceVMName   string `json:"source_vm_name" binding:"required"`   // 源虚拟机名称
	SnapshotName   string `json:"snapshot_name" binding:"required"`    // 快照名称
	PoolName       string `json:"pool_name" binding:"required"`        // 存储池名称（必须与源 VM 相同的存储池）
	NewVMName      string `json:"new_vm_name,omitempty"`               // 新虚拟机名称，可选，自动生成
	VCPUs          int    `json:"vcpus,omitempty"`                     // vCPU 数量，可选，默认继承源 VM
	MemoryMB       int    `json:"memory_mb,omitempty"`                 // 内存大小（MB），可选，默认继承源 VM
	NetworkType    string `json:"network_type,omitempty"`              // 网络类型（bridge/network），可选，默认继承源 VM
	NetworkSource  string `json:"network_source,omitempty"`            // 网络源，可选，默认继承源 VM
	Flatten        bool   `json:"flatten"`                             // 是否合并增量链（true=独立磁盘，false=保留增量链）
	StartAfterClone bool  `json:"start_after_clone,omitempty"`         // 克隆后是否启动
}

type CloneFromSnapshotResponse struct {
	Instance *Instance `json:"instance"`
	Message  string    `json:"message"`
}
