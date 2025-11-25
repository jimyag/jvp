package entity

// CreateVolumeRequest 创建卷请求
type CreateVolumeRequest struct {
	SizeGB uint64 `json:"sizeGB" binding:"required"`
}

// CreateVolumeResponse 创建卷响应
type CreateVolumeResponse struct {
	Volume *Volume `json:"volume"`
}

// DeleteVolumeRequest 删除卷请求
type DeleteVolumeRequest struct {
	VolumeID string `json:"volumeID" binding:"required"`
}

// DeleteVolumeResponse 删除卷响应
type DeleteVolumeResponse struct {
	Return bool `json:"return"`
}

// AttachVolumeRequest 附加卷请求
type AttachVolumeRequest struct {
	VolumeID   string `json:"volumeID" binding:"required"`
	InstanceID string `json:"instanceID" binding:"required"`
	Device     string `json:"device,omitempty"` // 可选，如果不指定则自动分配
}

// AttachVolumeResponse 附加卷响应
type AttachVolumeResponse struct {
	Attachment *VolumeAttachment `json:"attachment"`
}

// DetachVolumeRequest 分离卷请求
type DetachVolumeRequest struct {
	VolumeID   string `json:"volumeID" binding:"required"`
	InstanceID string `json:"instanceID,omitempty"`
}

// DetachVolumeResponse 分离卷响应
type DetachVolumeResponse struct {
	Return bool `json:"return"`
}

// ListVolumesRequest 列出卷请求
type ListVolumesRequest struct {
	// 暂时为空，未来可以添加过滤参数
}

// ListVolumesResponse 列出卷响应
type ListVolumesResponse struct {
	Volumes []Volume `json:"volumes"`
}

// DescribeVolumeRequest 描述卷请求
type DescribeVolumeRequest struct {
	VolumeID string `json:"volumeID" binding:"required"`
}

// DescribeVolumeResponse 描述卷响应
type DescribeVolumeResponse struct {
	Volume *Volume `json:"volume"`
}

// ResizeVolumeRequest 调整卷大小请求
type ResizeVolumeRequest struct {
	VolumeID  string `json:"volumeID" binding:"required"`
	NewSizeGB uint64 `json:"newSizeGB" binding:"required"`
}

// ResizeVolumeResponse 调整卷大小响应
type ResizeVolumeResponse struct {
	Return bool `json:"return"`
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
