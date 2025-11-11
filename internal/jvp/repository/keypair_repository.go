package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// KeyPairRepository 密钥对仓库接口
type KeyPairRepository interface {
	Create(ctx context.Context, keypair *model.KeyPair) error
	GetByID(ctx context.Context, id string) (*model.KeyPair, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*model.KeyPair, error)
	Update(ctx context.Context, keypair *model.KeyPair) error
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
}

type keyPairRepository struct {
	db *gorm.DB
}

// NewKeyPairRepository 创建密钥对仓库
func NewKeyPairRepository(db *gorm.DB) KeyPairRepository {
	return &keyPairRepository{db: db}
}

// Create 创建密钥对
func (r *keyPairRepository) Create(ctx context.Context, keypair *model.KeyPair) error {
	return r.db.WithContext(ctx).Create(keypair).Error
}

// GetByID 根据 ID 获取密钥对
func (r *keyPairRepository) GetByID(ctx context.Context, id string) (*model.KeyPair, error) {
	var keypair model.KeyPair
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&keypair).Error; err != nil {
		return nil, err
	}
	return &keypair, nil
}

// List 列出密钥对
func (r *keyPairRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.KeyPair, error) {
	var keypairs []*model.KeyPair
	query := r.db.WithContext(ctx).Model(&model.KeyPair{})

	// 应用过滤器
	if name, ok := filters["name"]; ok {
		query = query.Where("name = ?", name)
	}
	if algorithm, ok := filters["algorithm"]; ok {
		query = query.Where("algorithm = ?", algorithm)
	}
	if fingerprint, ok := filters["fingerprint"]; ok {
		query = query.Where("fingerprint = ?", fingerprint)
	}

	if err := query.Find(&keypairs).Error; err != nil {
		return nil, err
	}

	return keypairs, nil
}

// Update 更新密钥对
func (r *keyPairRepository) Update(ctx context.Context, keypair *model.KeyPair) error {
	return r.db.WithContext(ctx).Save(keypair).Error
}

// Delete 软删除密钥对
func (r *keyPairRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.KeyPair{}, "id = ?", id).Error
}

// HardDelete 硬删除密钥对
func (r *keyPairRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.KeyPair{}, "id = ?", id).Error
}
