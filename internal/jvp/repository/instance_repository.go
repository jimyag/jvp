package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// InstanceRepository 实例仓库接口
type InstanceRepository interface {
	Create(ctx context.Context, instance *model.Instance) error
	GetByID(ctx context.Context, id string) (*model.Instance, error)
	GetByIDWithRelations(ctx context.Context, id string) (*InstanceWithRelations, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*model.Instance, error)
	Update(ctx context.Context, instance *model.Instance) error
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
	GetByIDWithDeleted(ctx context.Context, id string) (*model.Instance, error)
}

// InstanceWithRelations 带关联数据的实例
type InstanceWithRelations struct {
	Instance    *model.Instance
	Image       *model.Image
	Volume      *model.Volume
	Tags        []*model.Tag
	Attachments []*model.VolumeAttachment
}

type instanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository 创建实例仓库
func NewInstanceRepository(db *gorm.DB) InstanceRepository {
	return &instanceRepository{db: db}
}

// Create 创建实例
func (r *instanceRepository) Create(ctx context.Context, instance *model.Instance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

// GetByID 根据 ID 获取实例（不包含关联数据，自动过滤已删除）
func (r *instanceRepository) GetByID(ctx context.Context, id string) (*model.Instance, error) {
	var instance model.Instance
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByIDWithRelations 根据 ID 获取实例（包含所有关联数据）
func (r *instanceRepository) GetByIDWithRelations(ctx context.Context, id string) (*InstanceWithRelations, error) {
	// 1. 获取实例
	instance, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &InstanceWithRelations{
		Instance: instance,
	}

	// 2. 获取关联的 Image（如果存在）
	if instance.ImageID != "" {
		var image model.Image
		if err := r.db.WithContext(ctx).Where("id = ?", instance.ImageID).First(&image).Error; err == nil {
			result.Image = &image
		}
	}

	// 3. 获取关联的 Volume（如果存在）
	if instance.VolumeID != "" {
		var volume model.Volume
		if err := r.db.WithContext(ctx).Where("id = ?", instance.VolumeID).First(&volume).Error; err == nil {
			result.Volume = &volume
		}
	}

	// 4. 获取 Tags
	var tags []*model.Tag
	if err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", "instance", id).
		Find(&tags).Error; err == nil {
		result.Tags = tags
	}

	// 5. 获取 Volume Attachments
	var attachments []*model.VolumeAttachment
	if err := r.db.WithContext(ctx).
		Where("instance_id = ?", id).
		Find(&attachments).Error; err == nil {
		result.Attachments = attachments
	}

	return result, nil
}

// List 列出实例
func (r *instanceRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.Instance, error) {
	var instances []*model.Instance
	query := r.db.WithContext(ctx).Model(&model.Instance{})

	// 应用过滤器
	if state, ok := filters["state"]; ok {
		query = query.Where("state = ?", state)
	}
	if imageID, ok := filters["image_id"]; ok {
		query = query.Where("image_id = ?", imageID)
	}
	if volumeID, ok := filters["volume_id"]; ok {
		query = query.Where("volume_id = ?", volumeID)
	}

	if err := query.Find(&instances).Error; err != nil {
		return nil, err
	}

	return instances, nil
}

// Update 更新实例
func (r *instanceRepository) Update(ctx context.Context, instance *model.Instance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

// Delete 软删除实例
func (r *instanceRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Instance{}, "id = ?", id).Error
}

// HardDelete 硬删除实例（永久删除）
func (r *instanceRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.Instance{}, "id = ?", id).Error
}

// GetByIDWithDeleted 根据 ID 获取实例（包含已删除的记录）
func (r *instanceRepository) GetByIDWithDeleted(ctx context.Context, id string) (*model.Instance, error) {
	var instance model.Instance
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}
