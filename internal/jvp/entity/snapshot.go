package entity

// CreateSnapshotRequest 创建快照请求
type CreateSnapshotRequest struct {
	VolumeID          string             `json:"volumeID" binding:"required"`
	Description       string             `json:"description,omitempty"`
	TagSpecifications []TagSpecification `json:"tagSpecifications,omitempty"`
}

// CreateSnapshotResponse 创建快照响应
type CreateSnapshotResponse struct {
	Snapshot *EBSSnapshot `json:"snapshot"`
}

// DeleteSnapshotRequest 删除快照请求
type DeleteSnapshotRequest struct {
	SnapshotID string `json:"snapshotID" binding:"required"`
}

// DeleteSnapshotResponse 删除快照响应
type DeleteSnapshotResponse struct {
	Return bool `json:"return"`
}

// DescribeSnapshotsRequest 描述快照请求
type DescribeSnapshotsRequest struct {
	SnapshotIDs         []string `json:"snapshotIDs,omitempty"`
	OwnerIDs            []string `json:"ownerIDs,omitempty"`
	RestorableByUserIDs []string `json:"restorableByUserIDs,omitempty"`
	Filters             []Filter `json:"filters,omitempty"`
	MaxResults          int      `json:"maxResults,omitempty"`
	NextToken           string   `json:"nextToken,omitempty"`
}

// DescribeSnapshotsResponse 描述快照响应
type DescribeSnapshotsResponse struct {
	Snapshots []EBSSnapshot `json:"snapshots"`
	NextToken string        `json:"nextToken,omitempty"`
}

// CopySnapshotRequest 复制快照请求
type CopySnapshotRequest struct {
	SourceSnapshotID  string             `json:"sourceSnapshotID" binding:"required"`
	SourceRegion      string             `json:"sourceRegion,omitempty"`
	Description       string             `json:"description,omitempty"`
	Encrypted         bool               `json:"encrypted,omitempty"`
	KmsKeyID          string             `json:"kmsKeyID,omitempty"`
	TagSpecifications []TagSpecification `json:"tagSpecifications,omitempty"`
}

// CopySnapshotResponse 复制快照响应
type CopySnapshotResponse struct {
	SnapshotID string       `json:"snapshotID"`
	Snapshot   *EBSSnapshot `json:"snapshot"`
}

// EBSSnapshot EBS 快照信息
type EBSSnapshot struct {
	SnapshotID   string `json:"snapshotID"` // snap-{uuid}
	VolumeID     string `json:"volumeID"`
	State        string `json:"state"` // pending, completed, error
	StartTime    string `json:"startTime"`
	Progress     string `json:"progress"` // 0-100%
	OwnerID      string `json:"ownerID"`
	Description  string `json:"description"`
	Encrypted    bool   `json:"encrypted"`
	VolumeSizeGB uint64 `json:"volumeSizeGB"`
	Tags         []Tag  `json:"tags"`
}
