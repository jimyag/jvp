package entity

import "time"

// Template 描述一个可用于创建虚拟机的模板
type Template struct {
	ID          string           `json:"id" yaml:"id"`
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	NodeName    string           `json:"node_name" yaml:"node_name"`
	PoolName    string           `json:"pool_name" yaml:"pool_name"`
	VolumeName  string           `json:"volume_name" yaml:"volume_name"`
	Path        string           `json:"path" yaml:"path"`
	Format      string           `json:"format" yaml:"format"`
	SizeBytes   uint64           `json:"size_bytes" yaml:"size_bytes"`
	SizeGB      uint64           `json:"size_gb" yaml:"size_gb"`
	Source      *TemplateSource  `json:"source,omitempty" yaml:"source,omitempty"`
	OS          TemplateOS       `json:"os" yaml:"os"`
	Features    TemplateFeatures `json:"features" yaml:"features"`
	Usage       TemplateUsage    `json:"usage" yaml:"usage"`
	Tags        []string         `json:"tags" yaml:"tags"`
	CreatedAt   time.Time        `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" yaml:"updated_at"`
}

// TemplateSource 描述模板的来源
type TemplateSource struct {
	Type       string `json:"type" yaml:"type"`                   // url | file | snapshot | volume
	URL        string `json:"url,omitempty" yaml:"url,omitempty"` // 当 type=url 时的下载地址
	LocalPath  string `json:"local_path,omitempty" yaml:"local_path,omitempty"`
	SnapshotID string `json:"snapshot_id,omitempty" yaml:"snapshot_id,omitempty"`
	VMID       string `json:"vm_id,omitempty" yaml:"vm_id,omitempty"`
	VolumeID   string `json:"volume_id,omitempty" yaml:"volume_id,omitempty"`
}

// TemplateOS 描述模板的操作系统信息
type TemplateOS struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Arch    string `json:"arch" yaml:"arch"`
	Kernel  string `json:"kernel" yaml:"kernel"`
}

// TemplateFeatures 描述模板的特性
type TemplateFeatures struct {
	CloudInit       bool `json:"cloud_init" yaml:"cloud_init"`
	Virtio          bool `json:"virtio" yaml:"virtio"`
	QemuGuestAgent  bool `json:"qemu_guest_agent" yaml:"qemu_guest_agent"`
	RequiresRestart bool `json:"requires_restart" yaml:"requires_restart"`
}

// TemplateUsage 描述模板的使用情况
type TemplateUsage struct {
	VMCount  uint64     `json:"vm_count" yaml:"vm_count"`
	LastUsed *time.Time `json:"last_used,omitempty" yaml:"last_used,omitempty"`
}

// RegisterTemplateRequest 注册模板请求
type RegisterTemplateRequest struct {
	NodeName    string           `json:"node_name"`                      // 节点名称,可选,默认 local
	PoolName    string           `json:"pool_name" binding:"required"`   // 存储池名称
	VolumeName  string           `json:"volume_name" binding:"required"` // 存储卷名称(包含扩展名)
	Name        string           `json:"name" binding:"required"`        // 模板名称
	Description string           `json:"description"`                    // 模板描述
	Tags        []string         `json:"tags"`                           // 标签
	OS          TemplateOS       `json:"os"`                             // 操作系统信息
	Features    TemplateFeatures `json:"features"`                       // 特性
	Source      *TemplateSource  `json:"source"`                         // 模板来源
}

// RegisterTemplateResponse 注册模板响应
type RegisterTemplateResponse struct {
	Template *Template `json:"template"`
}

// ListTemplatesRequest 列举模板请求
type ListTemplatesRequest struct {
	NodeName string `json:"node_name"` // 节点名称,可选
	PoolName string `json:"pool_name"` // 存储池过滤,可选
}

// ListTemplatesResponse 列举模板响应
type ListTemplatesResponse struct {
	Templates []Template `json:"templates"`
}

// DescribeTemplateRequest 查询模板详情请求
type DescribeTemplateRequest struct {
	NodeName   string `json:"node_name" binding:"required"`   // 节点名称
	TemplateID string `json:"template_id" binding:"required"` // 模板 ID
}

// DescribeTemplateResponse 查询模板详情响应
type DescribeTemplateResponse struct {
	Template *Template `json:"template"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	NodeName    string            `json:"node_name" binding:"required"`
	TemplateID  string            `json:"template_id" binding:"required"`
	Description *string           `json:"description"`
	Tags        *[]string         `json:"tags"`
	Features    *TemplateFeatures `json:"features"`
	OS          *TemplateOS       `json:"os"`
}

// UpdateTemplateResponse 更新模板响应
type UpdateTemplateResponse struct {
	Template *Template `json:"template"`
}

// DeleteTemplateRequest 删除模板请求
type DeleteTemplateRequest struct {
	NodeName     string `json:"node_name" binding:"required"`
	TemplateID   string `json:"template_id" binding:"required"`
	DeleteVolume bool   `json:"delete_volume"`
}

// DeleteTemplateResponse 删除模板响应
type DeleteTemplateResponse struct {
	Deleted bool `json:"deleted"`
}
