package model

import (
	"time"

	"gorm.io/gorm"
)

// Snapshot 快照表
type Snapshot struct {
	ID           string         `gorm:"primaryKey;type:text;column:id" json:"id"`                                          // snap-{uuid}
	VolumeID     string         `gorm:"type:text;not null;index:idx_snapshots_volume_id;column:volume_id" json:"volumeID"` // 关联 volumes.id
	State        string         `gorm:"type:text;not null;index:idx_snapshots_state;column:state" json:"state"`            // pending, completed, error
	StartTime    time.Time      `gorm:"type:datetime;not null;index:idx_snapshots_start_time;column:start_time" json:"startTime"`
	Progress     string         `gorm:"type:text;column:progress" json:"progress"` // 0-100%
	OwnerID      string         `gorm:"type:text;not null;index:idx_snapshots_owner_id;column:owner_id" json:"ownerID"`
	Description  string         `gorm:"type:text;column:description" json:"description"`
	Encrypted    bool           `gorm:"type:boolean;default:0;column:encrypted" json:"encrypted"`
	VolumeSizeGB uint64         `gorm:"type:integer;not null;column:volume_size_gb" json:"volumeSizeGB"`
	CreatedAt    time.Time      `gorm:"type:datetime;not null;column:created_at" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"type:datetime;not null;column:updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"type:datetime;index:idx_snapshots_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (Snapshot) TableName() string {
	return "snapshots"
}
