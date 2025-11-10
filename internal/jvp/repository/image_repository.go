package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// ImageRepository 镜像仓库接口
type ImageRepository interface {
	Create(ctx context.Context, image *model.Image) error
	GetByID(ctx context.Context, id string) (*model.Image, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*model.Image, error)
	Update(ctx context.Context, image *model.Image) error
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
	GetByIDWithDeleted(ctx context.Context, id string) (*model.Image, error)
}

type imageRepository struct {
	db *gorm.DB
}

// NewImageRepository 创建镜像仓库
func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

// Create 创建镜像
func (r *imageRepository) Create(ctx context.Context, image *model.Image) error {
	return r.db.WithContext(ctx).Create(image).Error
}

// GetByID 根据 ID 获取镜像
func (r *imageRepository) GetByID(ctx context.Context, id string) (*model.Image, error) {
	var image model.Image
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&image).Error; err != nil {
		return nil, err
	}
	return &image, nil
}

// List 列出镜像
func (r *imageRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.Image, error) {
	var images []*model.Image
	query := r.db.WithContext(ctx).Model(&model.Image{})

	// 应用过滤器
	if state, ok := filters["state"]; ok {
		query = query.Where("state = ?", state)
	}
	if pool, ok := filters["pool"]; ok {
		query = query.Where("pool = ?", pool)
	}

	if err := query.Find(&images).Error; err != nil {
		return nil, err
	}

	return images, nil
}

// Update 更新镜像
func (r *imageRepository) Update(ctx context.Context, image *model.Image) error {
	return r.db.WithContext(ctx).Save(image).Error
}

// Delete 软删除镜像
func (r *imageRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Image{}, "id = ?", id).Error
}

// HardDelete 硬删除镜像
func (r *imageRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.Image{}, "id = ?", id).Error
}

// GetByIDWithDeleted 根据 ID 获取镜像（包含已删除的记录）
func (r *imageRepository) GetByIDWithDeleted(ctx context.Context, id string) (*model.Image, error) {
	var image model.Image
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&image).Error; err != nil {
		return nil, err
	}
	return &image, nil
}
