package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// VolumeRepository 卷仓库接口
type VolumeRepository interface {
	Create(ctx context.Context, volume *model.Volume) error
	GetByID(ctx context.Context, id string) (*model.Volume, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*model.Volume, error)
	Update(ctx context.Context, volume *model.Volume) error
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
	GetByIDWithDeleted(ctx context.Context, id string) (*model.Volume, error)
}

type volumeRepository struct {
	db *gorm.DB
}

// NewVolumeRepository 创建卷仓库
func NewVolumeRepository(db *gorm.DB) VolumeRepository {
	return &volumeRepository{db: db}
}

// Create 创建卷
func (r *volumeRepository) Create(ctx context.Context, volume *model.Volume) error {
	return r.db.WithContext(ctx).Create(volume).Error
}

// GetByID 根据 ID 获取卷
func (r *volumeRepository) GetByID(ctx context.Context, id string) (*model.Volume, error) {
	var volume model.Volume
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&volume).Error; err != nil {
		return nil, err
	}
	return &volume, nil
}

// List 列出卷
func (r *volumeRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.Volume, error) {
	var volumes []*model.Volume
	query := r.db.WithContext(ctx).Model(&model.Volume{})

	// 应用过滤器
	if state, ok := filters["state"]; ok {
		query = query.Where("state = ?", state)
	}
	if snapshotID, ok := filters["snapshot_id"]; ok {
		query = query.Where("snapshot_id = ?", snapshotID)
	}
	if volumeType, ok := filters["volume_type"]; ok {
		query = query.Where("volume_type = ?", volumeType)
	}

	if err := query.Find(&volumes).Error; err != nil {
		return nil, err
	}

	return volumes, nil
}

// Update 更新卷
func (r *volumeRepository) Update(ctx context.Context, volume *model.Volume) error {
	return r.db.WithContext(ctx).Save(volume).Error
}

// Delete 软删除卷
func (r *volumeRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Volume{}, "id = ?", id).Error
}

// HardDelete 硬删除卷
func (r *volumeRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.Volume{}, "id = ?", id).Error
}

// GetByIDWithDeleted 根据 ID 获取卷（包含已删除的记录）
func (r *volumeRepository) GetByIDWithDeleted(ctx context.Context, id string) (*model.Volume, error) {
	var volume model.Volume
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&volume).Error; err != nil {
		return nil, err
	}
	return &volume, nil
}
