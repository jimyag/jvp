package repository

import (
	"context"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"gorm.io/gorm"
)

// SnapshotRepository 快照仓库接口
type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *model.Snapshot) error
	GetByID(ctx context.Context, id string) (*model.Snapshot, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*model.Snapshot, error)
	Update(ctx context.Context, snapshot *model.Snapshot) error
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
	GetByIDWithDeleted(ctx context.Context, id string) (*model.Snapshot, error)
}

type snapshotRepository struct {
	db *gorm.DB
}

// NewSnapshotRepository 创建快照仓库
func NewSnapshotRepository(db *gorm.DB) SnapshotRepository {
	return &snapshotRepository{db: db}
}

// Create 创建快照
func (r *snapshotRepository) Create(ctx context.Context, snapshot *model.Snapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

// GetByID 根据 ID 获取快照
func (r *snapshotRepository) GetByID(ctx context.Context, id string) (*model.Snapshot, error) {
	var snapshot model.Snapshot
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// List 列出快照
func (r *snapshotRepository) List(ctx context.Context, filters map[string]interface{}) ([]*model.Snapshot, error) {
	var snapshots []*model.Snapshot
	query := r.db.WithContext(ctx).Model(&model.Snapshot{})

	// 应用过滤器
	if state, ok := filters["state"]; ok {
		query = query.Where("state = ?", state)
	}
	if volumeID, ok := filters["volume_id"]; ok {
		query = query.Where("volume_id = ?", volumeID)
	}
	if ownerID, ok := filters["owner_id"]; ok {
		query = query.Where("owner_id = ?", ownerID)
	}

	if err := query.Find(&snapshots).Error; err != nil {
		return nil, err
	}

	return snapshots, nil
}

// Update 更新快照
func (r *snapshotRepository) Update(ctx context.Context, snapshot *model.Snapshot) error {
	return r.db.WithContext(ctx).Save(snapshot).Error
}

// Delete 软删除快照
func (r *snapshotRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Snapshot{}, "id = ?", id).Error
}

// HardDelete 硬删除快照
func (r *snapshotRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.Snapshot{}, "id = ?", id).Error
}

// GetByIDWithDeleted 根据 ID 获取快照（包含已删除的记录）
func (r *snapshotRepository) GetByIDWithDeleted(ctx context.Context, id string) (*model.Snapshot, error) {
	var snapshot model.Snapshot
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}
