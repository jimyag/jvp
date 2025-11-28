package entity

// ==================== Volume CRUD API ====================

// CreateVolumeRequest 创建卷请求
type CreateVolumeRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称(可选,默认本地节点)
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
	Name     string `json:"name"`                         // 卷名称(可选,不提供则自动生成)
	SizeGB   uint64 `json:"size_gb" binding:"required"`   // 大小(GB)
	Format   string `json:"format"`                       // 格式: qcow2/raw (默认: qcow2)
}

// CreateVolumeResponse 创建卷响应
type CreateVolumeResponse struct {
	Volume *Volume `json:"volume"`
}

// ListVolumesRequest 列举卷请求
type ListVolumesRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称(可选,默认本地节点)
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
}

// ListVolumesResponse 列举卷响应
type ListVolumesResponse struct {
	Volumes []Volume `json:"volumes"`
}

// DescribeVolumeRequest 查询卷详情请求
type DescribeVolumeRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称(可选,默认本地节点)
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
	VolumeID string `json:"volume_id" binding:"required"` // 卷 ID
}

// DescribeVolumeResponse 查询卷详情响应
type DescribeVolumeResponse struct {
	Volume *Volume `json:"volume"`
}

// ResizeVolumeRequest 扩容卷请求
type ResizeVolumeRequest struct {
	NodeName  string `json:"node_name"`                      // 节点名称(可选,默认本地节点)
	PoolName  string `json:"pool_name" binding:"required"`   // 存储池名称
	VolumeID  string `json:"volume_id" binding:"required"`   // 卷 ID
	NewSizeGB uint64 `json:"new_size_gb" binding:"required"` // 新大小(GB)
}

// ResizeVolumeResponse 扩容卷响应
type ResizeVolumeResponse struct {
	Volume *Volume `json:"volume"`
}

// DeleteVolumeRequest 删除卷请求
type DeleteVolumeRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称(可选,默认本地节点)
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
	VolumeID string `json:"volume_id" binding:"required"` // 卷 ID
}

// DeleteVolumeResponse 删除卷响应
type DeleteVolumeResponse struct {
	Message string `json:"message"`
}

// CreateVolumeFromURLRequest 从 URL 下载并创建卷请求
type CreateVolumeFromURLRequest struct {
	NodeName string `json:"node_name"`                    // 节点名称(可选,默认本地节点)
	PoolName string `json:"pool_name" binding:"required"` // 存储池名称
	Name     string `json:"name" binding:"required"`      // 卷名称(文件名)
	URL      string `json:"url" binding:"required"`       // 下载 URL
}

// CreateVolumeFromURLResponse 从 URL 下载并创建卷响应
type CreateVolumeFromURLResponse struct {
	Volume *Volume `json:"volume"`
}

// ==================== Storage Pool 相关 ====================
// ==================== 通用类型 ====================

// Filter 过滤器
type Filter struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Tag 标签
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TagSpecification 标签规范
type TagSpecification struct {
	ResourceType string `json:"resourceType"`
	Tags         []Tag  `json:"tags"`
}
