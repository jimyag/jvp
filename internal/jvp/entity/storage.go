package entity

// StoragePool 存储池信息
type StoragePool struct {
	Name        string `json:"name"`
	State       string `json:"state"`        // Active, Inactive, Building, Degraded, Inaccessible
	CapacityB   uint64 `json:"capacity_b"`   // 总容量（字节）
	AllocationB uint64 `json:"allocation_b"` // 已分配（字节）
	AvailableB  uint64 `json:"available_b"`  // 可用容量（字节）
	Path        string `json:"path"`         // Pool 路径
}

// Volume 存储卷信息
type Volume struct {
	ID          string `json:"id"`           // Volume ID: vol-{uuid}
	Name        string `json:"name"`         // Volume 名称
	Pool        string `json:"pool"`         // 所属 Pool 名称
	Path        string `json:"path"`         // 文件路径
	CapacityB   uint64 `json:"capacity_b"`   // 容量（字节）
	AllocationB uint64 `json:"allocation_b"` // 已分配（字节）
	Format      string `json:"format"`       // 格式：qcow2, raw
}

// CreateInternalVolumeRequest 创建内部 Volume 请求（用于 StorageService）
type CreateInternalVolumeRequest struct {
	PoolName string `json:"pool_name"` // Pool 名称
	VolumeID string `json:"volume_id"` // Volume ID: vol-{uuid}
	SizeGB   uint64 `json:"size_gb"`   // 大小（GB）
	Format   string `json:"format"`    // 格式：qcow2, raw（默认：qcow2）
}

// CreateVolumeFromImageRequest 从镜像创建 Volume 请求
type CreateVolumeFromImageRequest struct {
	ImageID  string `json:"image_id"`  // 镜像 ID: ami-{uuid}
	VolumeID string `json:"volume_id"` // Volume ID: vol-{uuid}
	SizeGB   uint64 `json:"size_gb"`   // 目标大小（GB）
}

// PoolConfig Pool 配置
type PoolConfig struct {
	Name string `json:"name"` // Pool 名称
	Type string `json:"type"` // Pool 类型：dir, fs（默认：dir）
	Path string `json:"path"` // Pool 路径
}
