package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// AttachmentRepository 卷附加关系仓库接口
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *model.VolumeAttachment) error
	GetByID(ctx context.Context, id uint) (*model.VolumeAttachment, error)
	GetByVolumeID(ctx context.Context, volumeID string) ([]*model.VolumeAttachment, error)
	GetByInstanceID(ctx context.Context, instanceID string) ([]*model.VolumeAttachment, error)
	GetByVolumeAndInstance(ctx context.Context, volumeID, instanceID string) (*model.VolumeAttachment, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*model.VolumeAttachment, error)
	Update(ctx context.Context, attachment *model.VolumeAttachment) error
	Delete(ctx context.Context, id uint) error
	DeleteByVolumeAndInstance(ctx context.Context, volumeID, instanceID string) error
	HardDelete(ctx context.Context, id uint) error
}

type attachmentRepository struct {
	db *gorm.DB
}

// NewAttachmentRepository 创建卷附加关系仓库
func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

// Create 创建卷附加关系
func (r *attachmentRepository) Create(ctx context.Context, attachment *model.VolumeAttachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

// GetByID 根据 ID 获取卷附加关系
func (r *attachmentRepository) GetByID(ctx context.Context, id uint) (*model.VolumeAttachment, error) {
	var attachment model.VolumeAttachment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&attachment).Error; err != nil {
		return nil, err
	}
	return &attachment, nil
}

// GetByVolumeID 根据 Volume ID 获取所有附加关系
func (r *attachmentRepository) GetByVolumeID(ctx context.Context, volumeID string) ([]*model.VolumeAttachment, error) {
	var attachments []*model.VolumeAttachment
	if err := r.db.WithContext(ctx).
		Where("volume_id = ?", volumeID).
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

// GetByInstanceID 根据 Instance ID 获取所有附加关系
func (r *attachmentRepository) GetByInstanceID(ctx context.Context, instanceID string) ([]*model.VolumeAttachment, error) {
	var attachments []*model.VolumeAttachment
	if err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

// GetByVolumeAndInstance 根据 Volume ID 和 Instance ID 获取附加关系
func (r *attachmentRepository) GetByVolumeAndInstance(ctx context.Context, volumeID, instanceID string) (*model.VolumeAttachment, error) {
	var attachment model.VolumeAttachment
	if err := r.db.WithContext(ctx).
		Where("volume_id = ? AND instance_id = ?", volumeID, instanceID).
		First(&attachment).Error; err != nil {
		return nil, err
	}
	return &attachment, nil
}

// List 列出卷附加关系
func (r *attachmentRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.VolumeAttachment, error) {
	var attachments []*model.VolumeAttachment
	query := r.db.WithContext(ctx).Model(&model.VolumeAttachment{})

	// 应用过滤器
	if state, ok := filters["state"]; ok {
		query = query.Where("state = ?", state)
	}
	if volumeID, ok := filters["volume_id"]; ok {
		query = query.Where("volume_id = ?", volumeID)
	}
	if instanceID, ok := filters["instance_id"]; ok {
		query = query.Where("instance_id = ?", instanceID)
	}

	if err := query.Find(&attachments).Error; err != nil {
		return nil, err
	}

	return attachments, nil
}

// Update 更新卷附加关系
func (r *attachmentRepository) Update(ctx context.Context, attachment *model.VolumeAttachment) error {
	return r.db.WithContext(ctx).Save(attachment).Error
}

// Delete 软删除卷附加关系
func (r *attachmentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.VolumeAttachment{}, "id = ?", id).Error
}

// DeleteByVolumeAndInstance 根据 Volume ID 和 Instance ID 软删除
func (r *attachmentRepository) DeleteByVolumeAndInstance(ctx context.Context, volumeID, instanceID string) error {
	return r.db.WithContext(ctx).
		Where("volume_id = ? AND instance_id = ?", volumeID, instanceID).
		Delete(&model.VolumeAttachment{}).Error
}

// HardDelete 硬删除卷附加关系
func (r *attachmentRepository) HardDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.VolumeAttachment{}, "id = ?", id).Error
}
