package model

import (
	"time"

	"gorm.io/gorm"
)

// Image 镜像表
type Image struct {
	ID          string         `gorm:"primaryKey;type:text;column:id" json:"id"`                            // ami-{uuid}
	Name        string         `gorm:"type:text;not null;column:name" json:"name"`                          // 镜像名称
	Description string         `gorm:"type:text;column:description" json:"description"`                     // 描述
	Pool        string         `gorm:"type:text;not null;column:pool" json:"pool"`                          // 所属 Pool 名称
	Path        string         `gorm:"type:text;not null;column:path" json:"path"`                          // 文件路径
	SizeGB      uint64         `gorm:"type:integer;not null;column:size_gb" json:"size_gb"`                 // 大小（GB）
	Format      string         `gorm:"type:text;not null;column:format" json:"format"`                      // qcow2, raw
	State       string         `gorm:"type:text;not null;index:idx_images_state;column:state" json:"state"` // available, pending, failed
	CreatedAt   time.Time      `gorm:"type:datetime;not null;index:idx_images_created_at;column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:datetime;not null;column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"type:datetime;index:idx_images_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (Image) TableName() string {
	return "images"
}
