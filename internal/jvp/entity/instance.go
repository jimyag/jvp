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
	ImageID    string          `json:"image_id"`              // 镜像 ID（可选，默认使用 ubuntu-jammy）
	SizeGB     uint64          `json:"size_gb"`               // 磁盘大小（GB）（可选，默认 20GB）
	MemoryMB   uint64          `json:"memory_mb"`             // 内存大小（MB）（可选，默认 2048MB）
	VCPUs      uint16          `json:"vcpus"`                 // 虚拟 CPU 数量（可选，默认 2）
	UserData   *UserDataConfig `json:"user_data,omitempty"`   // UserData 配置（可选）
	KeyPairIDs []string        `json:"keypair_ids,omitempty"` // 密钥对 ID 列表（可选）
}

// UserDataConfig UserData 配置
// 支持两种方式：
// 1. RawUserData: 直接提供原始 YAML 字符串（完全控制）
// 2. StructuredUserData: 提供结构化配置（推荐，更安全）
type UserDataConfig struct {
	// 原始 YAML 字符串（如果提供，将优先使用，忽略其他字段）
	RawUserData string `json:"raw_user_data,omitempty"`

	// 结构化配置（如果 RawUserData 为空，则使用此配置）
	StructuredUserData *StructuredUserData `json:"structured_user_data,omitempty"`
}

// StructuredUserData 结构化 UserData 配置
// 对应 cloudinit.Config 的简化版本，只暴露常用字段
type StructuredUserData struct {
	Hostname    string   `json:"hostname,omitempty"`     // 主机名
	Users       []User   `json:"users,omitempty"`        // 用户列表
	Groups      []Group  `json:"groups,omitempty"`       // 组列表
	Packages    []string `json:"packages,omitempty"`     // 要安装的软件包
	RunCmd      []string `json:"run_cmd,omitempty"`      // 启动后执行的命令
	WriteFiles  []File   `json:"write_files,omitempty"`  // 要写入的文件
	Timezone    string   `json:"timezone,omitempty"`     // 时区
	DisableRoot bool     `json:"disable_root,omitempty"` // 禁用 root 登录
}

// User 用户配置（简化版）
type User struct {
	Name              string   `json:"name,omitempty"`                // 用户名
	Groups            string   `json:"groups,omitempty"`              // 附加组（逗号分隔）
	SSHAuthorizedKeys []string `json:"ssh_authorized_keys,omitempty"` // SSH 公钥列表
	PlainTextPasswd   string   `json:"plain_text_passwd,omitempty"`   // 明文密码（不推荐，但支持）
	HashedPasswd      string   `json:"hashed_passwd,omitempty"`       // 密码哈希（推荐）
	Sudo              string   `json:"sudo,omitempty"`                // sudo 规则（如："ALL=(ALL) NOPASSWD:ALL"）
	Shell             string   `json:"shell,omitempty"`               // Shell（默认：/bin/bash）
}

// Group 组配置
type Group struct {
	Name    string   `json:"name"`    // 组名
	Members []string `json:"members"` // 组成员列表
}

// File 文件配置
type File struct {
	Path        string `json:"path"`                  // 文件路径
	Content     string `json:"content"`               // 文件内容
	Owner       string `json:"owner,omitempty"`       // 文件所有者（默认：root:root）
	Permissions string `json:"permissions,omitempty"` // 文件权限（默认：0644）
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

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	InstanceID string          `json:"instance_id"          binding:"required"` // 实例 ID
	Users      []PasswordReset `json:"users"                binding:"required"` // 用户密码重置列表
	AutoStart  bool            `json:"auto_start,omitempty"`                    // 重置后是否自动启动（如果之前是运行状态）
}

// PasswordReset 密码重置信息
type PasswordReset struct {
	Username    string `json:"username"     binding:"required"` // 用户名
	NewPassword string `json:"new_password" binding:"required"` // 新密码（明文，传输时加密）
}

// ResetPasswordResponse 重置密码响应
type ResetPasswordResponse struct {
	InstanceID string   `json:"instance_id"` // 实例 ID
	Success    bool     `json:"success"`     // 是否成功
	Message    string   `json:"message"`     // 操作结果消息
	Users      []string `json:"users"`       // 成功重置密码的用户列表
}

// VMTemplate VM 模板信息
// VM Template 是指带有快照的虚拟机,可以基于快照克隆新的 VM
type VMTemplate struct {
	ID          string `json:"id"`          // VM UUID
	Name        string `json:"name"`        // 模板名称 (VM名称-template)
	Description string `json:"description"` // 模板描述
	SourceVM    string `json:"sourceVM"`    // 源 VM 名称
	VCPUs       uint16 `json:"vcpus"`       // 虚拟 CPU 数量
	Memory      uint64 `json:"memory"`      // 内存大小（MB）
	DiskSize    uint64 `json:"diskSize"`    // 磁盘大小（GB）
	CreatedAt   string `json:"createdAt"`   // 创建时间
}

// ListVMTemplatesResponse 列出 VM 模板响应
type ListVMTemplatesResponse struct {
	Templates []VMTemplate `json:"templates"`
}
