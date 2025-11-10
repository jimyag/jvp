package model

import (
	"time"

	"gorm.io/gorm"
)

// VolumeAttachment 卷附加关系表
type VolumeAttachment struct {
	ID                  uint           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	VolumeID            string         `gorm:"type:text;not null;index:idx_attachments_volume_id;column:volume_id" json:"volumeID"`       // 关联 volumes.id
	InstanceID          string         `gorm:"type:text;not null;index:idx_attachments_instance_id;column:instance_id" json:"instanceID"` // 关联 instances.id
	Device              string         `gorm:"type:text;not null;column:device" json:"device"`                                            // /dev/vdb, /dev/vdc 等
	State               string         `gorm:"type:text;not null;index:idx_attachments_state;column:state" json:"state"`                  // attaching, attached, detaching, detached
	AttachTime          time.Time      `gorm:"type:datetime;not null;column:attach_time" json:"attachTime"`
	DeleteOnTermination bool           `gorm:"type:boolean;default:0;column:delete_on_termination" json:"deleteOnTermination"`
	CreatedAt           time.Time      `gorm:"type:datetime;not null;column:created_at" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"type:datetime;not null;column:updated_at" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"type:datetime;index:idx_attachments_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (VolumeAttachment) TableName() string {
	return "volume_attachments"
}
