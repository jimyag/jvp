package entity

// StoragePool 存储池实体
type StoragePool struct {
	Name        string `json:"name"`         // 存储池名称
	UUID        string `json:"uuid"`         // UUID
	State       string `json:"state"`        // 状态：active, inactive
	Type        string `json:"type"`         // 类型：dir, fs, netfs, disk, iscsi, logical
	Capacity    uint64 `json:"capacity"`     // 总容量（字节）
	Allocation  uint64 `json:"allocation"`   // 已分配（字节）
	Available   uint64 `json:"available"`    // 可用容量（字节）
	Path        string `json:"path"`         // 存储池路径
	VolumeCount int    `json:"volume_count"` // 卷数量
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

// VolumeAttachment Volume 附加信息
type VolumeAttachment struct {
	InstanceID string `json:"instanceId"` // 实例 ID
	Device     string `json:"device"`     // /dev/vdb, /dev/vdc 等
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

// ============================================================================
// API 请求和响应
// ============================================================================

// ListStoragePoolsRequest 列举存储池请求
type ListStoragePoolsRequest struct {
	NodeName string `json:"node_name"` // 节点名称（可选，为空表示本地节点）
}

// ListStoragePoolsResponse 列举存储池响应
type ListStoragePoolsResponse struct {
	Pools []StoragePool `json:"pools"`
}

// DescribeStoragePoolRequest 查询存储池详情请求
type DescribeStoragePoolRequest struct {
	NodeName string `json:"node_name"` // 节点名称（可选，为空表示本地节点）
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
}

// DescribeStoragePoolResponse 查询存储池详情响应
type DescribeStoragePoolResponse struct {
	Pool *StoragePool `json:"pool"`
}

// CreateStoragePoolRequest 创建存储池请求
type CreateStoragePoolRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称（可选，为空表示本地节点）
	Name     string `json:"name" binding:"required"`      // 存储池名称
	Type     string `json:"type"`                         // 类型：dir, fs, netfs（默认：dir）
	Path     string `json:"path" binding:"required"`      // 存储池路径
}

// CreateStoragePoolResponse 创建存储池响应
type CreateStoragePoolResponse struct {
	Pool *StoragePool `json:"pool"`
}

// DeleteStoragePoolRequest 删除存储池请求
type DeleteStoragePoolRequest struct {
	NodeName      string `json:"node_name"`                    // 节点名称（可选，为空表示本地节点）
	PoolName      string `json:"pool_name" binding:"required"` // 存储池名称
	DeleteVolumes bool   `json:"delete_volumes"`               // 是否删除存储池中的所有卷和目录（默认 false）
}

// DeleteStoragePoolResponse 删除存储池响应
type DeleteStoragePoolResponse struct {
	Message string `json:"message"`
}

// StartStoragePoolRequest 启动存储池请求
type StartStoragePoolRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称（可选，为空表示本地节点）
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
}

// StartStoragePoolResponse 启动存储池响应
type StartStoragePoolResponse struct {
	Pool *StoragePool `json:"pool"`
}

// StopStoragePoolRequest 停止存储池请求
type StopStoragePoolRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称（可选，为空表示本地节点）
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
}

// StopStoragePoolResponse 停止存储池响应
type StopStoragePoolResponse struct {
	Pool *StoragePool `json:"pool"`
}

// RefreshStoragePoolRequest 刷新存储池请求
type RefreshStoragePoolRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称（可选，为空表示本地节点）
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
}

// RefreshStoragePoolResponse 刷新存储池响应
type RefreshStoragePoolResponse struct {
	Pool *StoragePool `json:"pool"`
}
