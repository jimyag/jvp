package entity

// CreateVolumeRequest 创建卷请求
type CreateVolumeRequest struct {
	AvailabilityZone  string             `json:"availabilityZone,omitempty"`
	SizeGB            uint64             `json:"sizeGB"`
	VolumeType        string             `json:"volumeType,omitempty"` // standard, io1, gp2, gp3（默认：gp2）
	Iops              int                `json:"iops,omitempty"`
	Encrypted         bool               `json:"encrypted,omitempty"`
	KmsKeyID          string             `json:"kmsKeyID,omitempty"`
	SnapshotID        string             `json:"snapshotID,omitempty"`
	TagSpecifications []TagSpecification `json:"tagSpecifications,omitempty"`
}

// CreateVolumeResponse 创建卷响应
type CreateVolumeResponse struct {
	Volume *EBSVolume `json:"volume"`
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
	Device     string `json:"device,omitempty"`
}

// AttachVolumeResponse 附加卷响应
type AttachVolumeResponse struct {
	Attachment *VolumeAttachment `json:"attachment"`
}

// DetachVolumeRequest 分离卷请求
type DetachVolumeRequest struct {
	VolumeID   string `json:"volumeID" binding:"required"`
	InstanceID string `json:"instanceID,omitempty"`
	Force      bool   `json:"force,omitempty"`
}

// DetachVolumeResponse 分离卷响应
type DetachVolumeResponse struct {
	Attachment *VolumeAttachment `json:"attachment"`
}

// DescribeVolumesRequest 描述卷请求
type DescribeVolumesRequest struct {
	VolumeIDs  []string `json:"volumeIDs,omitempty"`
	Filters    []Filter `json:"filters,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
	NextToken  string   `json:"nextToken,omitempty"`
}

// DescribeVolumesResponse 描述卷响应
type DescribeVolumesResponse struct {
	Volumes   []EBSVolume `json:"volumes"`
	NextToken string      `json:"nextToken,omitempty"`
}

// ModifyVolumeRequest 修改卷请求
type ModifyVolumeRequest struct {
	VolumeID   string `json:"volumeID" binding:"required"`
	SizeGB     uint64 `json:"sizeGB,omitempty"`
	VolumeType string `json:"volumeType,omitempty"`
	Iops       int    `json:"iops,omitempty"`
}

// ModifyVolumeResponse 修改卷响应
type ModifyVolumeResponse struct {
	VolumeModification *VolumeModification `json:"volumeModification"`
}

// EBSVolume EBS 卷信息
type EBSVolume struct {
	VolumeID         string             `json:"volumeID"` // vol-{uuid}
	SizeGB           uint64             `json:"sizeGB"`
	SnapshotID       string             `json:"snapshotID"`
	AvailabilityZone string             `json:"availabilityZone"`
	State            string             `json:"state"` // creating, available, in-use, deleting, deleted, error
	VolumeType       string             `json:"volumeType"`
	Iops             int                `json:"iops"`
	Encrypted        bool               `json:"encrypted"`
	KmsKeyID         string             `json:"kmsKeyID"`
	Attachments      []VolumeAttachment `json:"attachments"`
	CreateTime       string             `json:"createTime"`
	Tags             []Tag              `json:"tags"`
}

// VolumeAttachment 卷附加信息
type VolumeAttachment struct {
	VolumeID            string `json:"volumeID"`
	InstanceID          string `json:"instanceID"`
	Device              string `json:"device"` // /dev/vdb, /dev/vdc 等
	State               string `json:"state"`  // attaching, attached, detaching, detached
	AttachTime          string `json:"attachTime"`
	DeleteOnTermination bool   `json:"deleteOnTermination"`
}

// VolumeModification 卷修改信息
type VolumeModification struct {
	VolumeID          string `json:"volumeID"`
	ModificationState string `json:"modificationState"` // modifying, optimizing, completed, failed
	StatusMessage     string `json:"statusMessage"`
	TargetSizeGB      uint64 `json:"targetSizeGB"`
	TargetVolumeType  string `json:"targetVolumeType"`
	TargetIops        int    `json:"targetIops"`
	StartTime         string `json:"startTime"`
	EndTime           string `json:"endTime,omitempty"`
}

// TagSpecification 标签规范
type TagSpecification struct {
	ResourceType string `json:"resourceType"` // volume, snapshot
	Tags         []Tag  `json:"tags"`
}

// Tag 标签
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Filter 过滤器
type Filter struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}
