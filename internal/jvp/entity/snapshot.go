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
