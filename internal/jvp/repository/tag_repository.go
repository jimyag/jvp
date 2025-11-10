package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// TagRepository 标签仓库接口
type TagRepository interface {
	Create(ctx context.Context, tag *model.Tag) error
	GetByResource(ctx context.Context, resourceType, resourceID string) ([]*model.Tag, error)
	GetByKey(ctx context.Context, resourceType, resourceID, tagKey string) (*model.Tag, error)
	Update(ctx context.Context, tag *model.Tag) error
	Delete(ctx context.Context, resourceType, resourceID, tagKey string) error
	DeleteByResource(ctx context.Context, resourceType, resourceID string) error
	HardDelete(ctx context.Context, resourceType, resourceID, tagKey string) error
}

type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建标签仓库
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// Create 创建标签
func (r *tagRepository) Create(ctx context.Context, tag *model.Tag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

// GetByResource 根据资源获取所有标签
func (r *tagRepository) GetByResource(ctx context.Context, resourceType, resourceID string) ([]*model.Tag, error) {
	var tags []*model.Tag
	if err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// GetByKey 根据资源类型、资源ID和标签key获取标签
func (r *tagRepository) GetByKey(ctx context.Context, resourceType, resourceID, tagKey string) (*model.Tag, error) {
	var tag model.Tag
	if err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ? AND tag_key = ?", resourceType, resourceID, tagKey).
		First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// Update 更新标签
func (r *tagRepository) Update(ctx context.Context, tag *model.Tag) error {
	return r.db.WithContext(ctx).Save(tag).Error
}

// Delete 软删除标签
func (r *tagRepository) Delete(ctx context.Context, resourceType, resourceID, tagKey string) error {
	return r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ? AND tag_key = ?", resourceType, resourceID, tagKey).
		Delete(&model.Tag{}).Error
}

// DeleteByResource 删除资源的所有标签
func (r *tagRepository) DeleteByResource(ctx context.Context, resourceType, resourceID string) error {
	return r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Delete(&model.Tag{}).Error
}

// HardDelete 硬删除标签
func (r *tagRepository) HardDelete(ctx context.Context, resourceType, resourceID, tagKey string) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("resource_type = ? AND resource_id = ? AND tag_key = ?", resourceType, resourceID, tagKey).
		Delete(&model.Tag{}).Error
}
