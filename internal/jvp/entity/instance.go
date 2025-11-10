// Package entity 定义业务实体
package entity

// Instance 实例信息
type Instance struct {
	ID         string `json:"id"`          // Instance ID: i-{uuid}
	Name       string `json:"name"`        // 实例名称
	State      string `json:"state"`       // 状态：running, stopped, pending, failed
	ImageID    string `json:"image_id"`    // 使用的镜像 ID
	VolumeID   string `json:"volume_id"`   // 使用的 Volume ID
	MemoryMB   uint64 `json:"memory_mb"`   // 内存大小（MB）
	VCPUs      uint16 `json:"vcpus"`       // 虚拟 CPU 数量
	CreatedAt  string `json:"created_at"`  // 创建时间
	DomainUUID string `json:"domain_uuid"` // Libvirt Domain UUID
	DomainName string `json:"domain_name"` // Libvirt Domain 名称
}

// RunInstanceRequest 创建实例请求
type RunInstanceRequest struct {
	ImageID  string `json:"image_id"`  // 镜像 ID（可选，默认使用 ubuntu-jammy）
	SizeGB   uint64 `json:"size_gb"`   // 磁盘大小（GB）（可选，默认 20GB）
	MemoryMB uint64 `json:"memory_mb"` // 内存大小（MB）（可选，默认 2048MB）
	VCPUs    uint16 `json:"vcpus"`     // 虚拟 CPU 数量（可选，默认 2）
}

// RunInstanceResponse 创建实例响应
type RunInstanceResponse struct {
	Instance *Instance `json:"instance"`
}
