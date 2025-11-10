package model

import (
	"time"

	"gorm.io/gorm"
)

// Tag 标签表（通用设计，支持多资源类型）
type Tag struct {
	ID           uint           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ResourceType string         `gorm:"type:text;not null;index:idx_tags_resource_type;column:resource_type" json:"resourceType"` // instance, image, volume, snapshot
	ResourceID   string         `gorm:"type:text;not null;index:idx_tags_resource_id;column:resource_id" json:"resourceID"`       // 对应资源的 ID
	TagKey       string         `gorm:"type:text;not null;column:tag_key" json:"tagKey"`                                          // 不在 tag_key 上建索引（tag 太长）
	TagValue     string         `gorm:"type:text;not null;column:tag_value" json:"tagValue"`                                      // 不在 tag_value 上建索引（tag 太长）
	CreatedAt    time.Time      `gorm:"type:datetime;not null;column:created_at" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"type:datetime;index:idx_tags_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}
