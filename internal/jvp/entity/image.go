package entity

// Image 镜像信息
type Image struct {
	ID          string `json:"id"`          // 镜像 ID: ami-{uuid}
	Name        string `json:"name"`        // 镜像名称
	Description string `json:"description"` // 描述
	Pool        string `json:"pool"`        // 所属 Pool 名称（通常是 images）
	Path        string `json:"path"`        // 文件路径
	SizeGB      uint64 `json:"size_gb"`     // 大小（GB）
	Format      string `json:"format"`      // 格式：qcow2, raw
	State       string `json:"state"`       // 状态：available, pending, failed
	CreatedAt   string `json:"created_at"`  // 创建时间
}

// RegisterImageRequest 注册镜像请求
type RegisterImageRequest struct {
	Name        string `json:"name"`        // 镜像名称
	Description string `json:"description"` // 描述
	Path        string `json:"path"`        // 镜像文件路径（必须已存在于 images pool 中）
	Pool        string `json:"pool"`        // Pool 名称（默认：images）
}

// CreateImageFromInstanceRequest 从 Instance 创建镜像请求
type CreateImageFromInstanceRequest struct {
	InstanceID  string `json:"instance_id"` // Instance ID: i-{uuid}
	ImageName   string `json:"image_name"`  // 镜像名称
	Description string `json:"description"` // 描述
}
