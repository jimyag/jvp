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

// DescribeInstancesRequest 描述实例请求
type DescribeInstancesRequest struct {
	InstanceIDs []string `json:"instanceIDs,omitempty"`
	Filters     []Filter `json:"filters,omitempty"`
	MaxResults  int      `json:"maxResults,omitempty"`
	NextToken   string   `json:"nextToken,omitempty"`
}

// DescribeInstancesResponse 描述实例响应
type DescribeInstancesResponse struct {
	Instances []Instance `json:"instances"`
	NextToken string     `json:"nextToken,omitempty"`
}

// TerminateInstancesRequest 终止实例请求
type TerminateInstancesRequest struct {
	InstanceIDs []string `json:"instanceIDs" binding:"required"`
}

// TerminateInstancesResponse 终止实例响应
type TerminateInstancesResponse struct {
	TerminatingInstances []InstanceStateChange `json:"terminatingInstances"`
}

// StopInstancesRequest 停止实例请求
type StopInstancesRequest struct {
	InstanceIDs []string `json:"instanceIDs"     binding:"required"`
	Force       bool     `json:"force,omitempty"`
}

// StopInstancesResponse 停止实例响应
type StopInstancesResponse struct {
	StoppingInstances []InstanceStateChange `json:"stoppingInstances"`
}

// StartInstancesRequest 启动实例请求
type StartInstancesRequest struct {
	InstanceIDs []string `json:"instanceIDs" binding:"required"`
}

// StartInstancesResponse 启动实例响应
type StartInstancesResponse struct {
	StartingInstances []InstanceStateChange `json:"startingInstances"`
}

// RebootInstancesRequest 重启实例请求
type RebootInstancesRequest struct {
	InstanceIDs []string `json:"instanceIDs" binding:"required"`
}

// RebootInstancesResponse 重启实例响应
type RebootInstancesResponse struct {
	RebootingInstances []InstanceStateChange `json:"rebootingInstances"`
}

// InstanceStateChange 实例状态变更信息
type InstanceStateChange struct {
	InstanceID    string `json:"instanceID"`
	CurrentState  string `json:"currentState"`  // 当前状态
	PreviousState string `json:"previousState"` // 之前的状态
}

// ModifyInstanceAttributeRequest 修改实例属性请求
type ModifyInstanceAttributeRequest struct {
	InstanceID string  `json:"instanceID"         binding:"required"`
	MemoryMB   *uint64 `json:"memoryMB,omitempty"` // 内存大小（MB），nil 表示不修改
	VCPUs      *uint16 `json:"vcpus,omitempty"`    // VCPU 数量，nil 表示不修改
	Name       *string `json:"name,omitempty"`     // 实例名称，nil 表示不修改
	Live       bool    `json:"live,omitempty"`     // 是否热修改（如果实例正在运行）
}

// ModifyInstanceAttributeResponse 修改实例属性响应
type ModifyInstanceAttributeResponse struct {
	Instance *Instance `json:"instance"`
}
