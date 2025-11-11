package model

import (
	"time"

	"gorm.io/gorm"
)

// KeyPair 密钥对表
type KeyPair struct {
	ID          string         `gorm:"primaryKey;type:text;column:id" json:"id"`                                                // kp-{uuid}
	Name        string         `gorm:"type:text;not null;index:idx_keypairs_name;column:name" json:"name"`                      // 密钥对名称（允许重复，不同用户可能使用相同名称）
	Algorithm   string         `gorm:"type:text;not null;column:algorithm" json:"algorithm"`                                    // rsa, ed25519
	PublicKey   string         `gorm:"type:text;not null;column:public_key" json:"public_key"`                                  // 公钥内容
	Fingerprint string         `gorm:"type:text;not null;index:idx_keypairs_fingerprint;column:fingerprint" json:"fingerprint"` // 公钥指纹（允许重复，不同用户可能使用相同公钥）
	CreatedAt   time.Time      `gorm:"type:datetime;not null;index:idx_keypairs_created_at;column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:datetime;not null;column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"type:datetime;index:idx_keypairs_deleted_at;column:deleted_at" json:"deleted_at,omitempty"` // 软删除
}

// TableName 指定表名
func (KeyPair) TableName() string {
	return "keypairs"
}
