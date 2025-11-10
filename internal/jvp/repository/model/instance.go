package model

import (
	"time"

	"gorm.io/gorm"
)

// Instance 实例表
type Instance struct {
	ID         string         `gorm:"primaryKey;type:text;column:id" json:"id"`                                  // i-{uuid}
	Name       string         `gorm:"type:text;not null;column:name" json:"name"`                                // 实例名称
	State      string         `gorm:"type:text;not null;index:idx_instances_state;column:state" json:"state"`    // running, stopped, pending, failed
	ImageID    string         `gorm:"type:text;index:idx_instances_image_id;column:image_id" json:"image_id"`    // 关联 images.id
	VolumeID   string         `gorm:"type:text;index:idx_instances_volume_id;column:volume_id" json:"volume_id"` // 关联 volumes.id（主卷）
	MemoryMB   uint64         `gorm:"type:integer;not null;column:memory_mb" json:"memory_mb"`                   // 内存大小（MB）
	VCPUs      uint16         `gorm:"type:integer;not null;column:vcpus" json:"vcpus"`                           // 虚拟 CPU 数量
	CreatedAt  time.Time      `gorm:"type:datetime;not null;index:idx_instances_created_at;column:created_at" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"type:datetime;not null;column:updated_at" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"type:datetime;index:idx_instances_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
	DomainUUID string         `gorm:"type:text;column:domain_uuid" json:"domain_uuid"`                                            // Libvirt Domain UUID
	DomainName string         `gorm:"type:text;column:domain_name" json:"domain_name"`                                            // Libvirt Domain 名称
}

// TableName 指定表名
func (Instance) TableName() string {
	return "instances"
}
