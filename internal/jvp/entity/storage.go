package entity

// StoragePool 存储池信息
type StoragePool struct {
	Name        string   `json:"name"`
	UUID        string   `json:"uuid,omitempty"`
	State       string   `json:"state"`        // Active, Inactive, Building, Degraded, Inaccessible
	CapacityB   uint64   `json:"capacity_b"`   // 总容量（字节）
	AllocationB uint64   `json:"allocation_b"` // 已分配（字节）
	AvailableB  uint64   `json:"available_b"`  // 可用容量（字节）
	Path        string   `json:"path"`         // Pool 路径
	Type        string   `json:"type,omitempty"`        // dir, fs, netfs, disk, iscsi, logical, etc
	VolumeCount int      `json:"volumeCount,omitempty"` // 卷数量
	Volumes     []Volume `json:"volumes,omitempty"`     // 池中的卷列表
}

// Volume 存储卷信息
type Volume struct {
	ID          string             `json:"volumeID"`     // Volume ID: vol-{uuid}
	Name        string             `json:"name"`         // Volume 名称
	Pool        string             `json:"pool"`         // 所属 Pool 名称
	Path        string             `json:"path"`         // 文件路径
	CapacityB   uint64             `json:"capacity_b"`   // 容量（字节）
	SizeGB      uint64             `json:"sizeGB"`       // 容量（GB）- 前端展示用
	AllocationB uint64             `json:"allocation_b"` // 已分配（字节）
	Format      string             `json:"format"`       // 格式：qcow2, raw, iso
	State       string             `json:"state"`        // 状态：available, in-use, creating, deleting
	VolumeType  string             `json:"volumeType"`   // 类型：disk, template, iso
	CreateTime  string             `json:"createTime,omitempty"` // 创建时间
	Attachments []VolumeAttachment `json:"attachments,omitempty"` // 附加到的实例列表
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
