package model

import (
	"time"

	"gorm.io/gorm"
)

// Volume 卷表
type Volume struct {
	ID               string         `gorm:"primaryKey;type:text;column:id" json:"id"` // vol-{uuid}
	SizeGB           uint64         `gorm:"type:integer;not null;column:size_gb" json:"sizeGB"`
	SnapshotID       string         `gorm:"type:text;index:idx_volumes_snapshot_id;column:snapshot_id" json:"snapshotID"` // 关联 snapshots.id
	AvailabilityZone string         `gorm:"type:text;column:availability_zone" json:"availabilityZone"`
	State            string         `gorm:"type:text;not null;index:idx_volumes_state;column:state" json:"state"` // creating, available, in-use, deleting, deleted, error
	VolumeType       string         `gorm:"type:text;column:volume_type" json:"volumeType"`                       // standard, io1, gp2, gp3
	Iops             int            `gorm:"type:integer;column:iops" json:"iops"`
	Encrypted        bool           `gorm:"type:boolean;default:0;column:encrypted" json:"encrypted"`
	KmsKeyID         string         `gorm:"type:text;column:kms_key_id" json:"kmsKeyID"`
	CreateTime       time.Time      `gorm:"type:datetime;not null;index:idx_volumes_create_time;column:create_time" json:"createTime"`
	UpdatedAt        time.Time      `gorm:"type:datetime;not null;column:updated_at" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"type:datetime;index:idx_volumes_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (Volume) TableName() string {
	return "volumes"
}
