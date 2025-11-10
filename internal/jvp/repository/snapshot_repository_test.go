package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRepository(t *testing.T) {
	t.Parallel()

	repo := setupTestDB(t)

	snapshotRepo := NewSnapshotRepository(repo.DB())
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		snapshot := &model.Snapshot{
			ID:           "snap-123",
			VolumeID:     "vol-123",
			State:        "completed",
			StartTime:    time.Now(),
			Progress:     "100%",
			OwnerID:      "owner-123",
			VolumeSizeGB: 20,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := snapshotRepo.Create(ctx, snapshot)
		assert.NoError(t, err)

		got, err := snapshotRepo.GetByID(ctx, "snap-123")
		assert.NoError(t, err)
		assert.Equal(t, snapshot.ID, got.ID)
		assert.Equal(t, snapshot.VolumeID, got.VolumeID)
		assert.Equal(t, snapshot.State, got.State)
	})

	t.Run("Update", func(t *testing.T) {
		snapshot := &model.Snapshot{
			ID:           "snap-456",
			VolumeID:     "vol-456",
			State:        "pending",
			StartTime:    time.Now(),
			Progress:     "50%",
			OwnerID:      "owner-456",
			VolumeSizeGB: 20,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := snapshotRepo.Create(ctx, snapshot)
		require.NoError(t, err)

		snapshot.State = "completed"
		snapshot.Progress = "100%"
		err = snapshotRepo.Update(ctx, snapshot)
		assert.NoError(t, err)

		got, err := snapshotRepo.GetByID(ctx, "snap-456")
		assert.NoError(t, err)
		assert.Equal(t, "completed", got.State)
		assert.Equal(t, "100%", got.Progress)
	})

	t.Run("List with filters", func(t *testing.T) {
		// 使用唯一的 ID 前缀避免与其他测试冲突
		snapshots := []*model.Snapshot{
			{ID: "snap-filter-111", VolumeID: "vol-filter-111", State: "completed", StartTime: time.Now(), OwnerID: "owner-filter-1", VolumeSizeGB: 10, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "snap-filter-222", VolumeID: "vol-filter-111", State: "pending", StartTime: time.Now(), OwnerID: "owner-filter-1", VolumeSizeGB: 20, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "snap-filter-333", VolumeID: "vol-filter-222", State: "completed", StartTime: time.Now(), OwnerID: "owner-filter-2", VolumeSizeGB: 30, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, snap := range snapshots {
			err := snapshotRepo.Create(ctx, snap)
			require.NoError(t, err)
		}

		// 测试按状态过滤（只查询刚创建的数据）
		completed, err := snapshotRepo.List(ctx, map[string]interface{}{"state": "completed"})
		assert.NoError(t, err)
		filtered := make([]*model.Snapshot, 0)
		for _, snap := range completed {
			if snap.ID == "snap-filter-111" || snap.ID == "snap-filter-333" {
				filtered = append(filtered, snap)
			}
		}
		assert.Len(t, filtered, 2)

		// 测试按 volume_id 过滤
		byVolume, err := snapshotRepo.List(ctx, map[string]interface{}{"volume_id": "vol-filter-111"})
		assert.NoError(t, err)
		filtered = make([]*model.Snapshot, 0)
		for _, snap := range byVolume {
			if snap.ID == "snap-filter-111" || snap.ID == "snap-filter-222" {
				filtered = append(filtered, snap)
			}
		}
		assert.Len(t, filtered, 2)

		// 测试按 owner_id 过滤
		byOwner, err := snapshotRepo.List(ctx, map[string]interface{}{"owner_id": "owner-filter-1"})
		assert.NoError(t, err)
		filtered = make([]*model.Snapshot, 0)
		for _, snap := range byOwner {
			if snap.ID == "snap-filter-111" || snap.ID == "snap-filter-222" {
				filtered = append(filtered, snap)
			}
		}
		assert.Len(t, filtered, 2)
	})

	t.Run("Delete and soft delete", func(t *testing.T) {
		snapshot := &model.Snapshot{
			ID:           "snap-delete",
			VolumeID:     "vol-delete",
			State:        "completed",
			StartTime:    time.Now(),
			Progress:     "100%",
			OwnerID:      "owner-delete",
			VolumeSizeGB: 20,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := snapshotRepo.Create(ctx, snapshot)
		require.NoError(t, err)

		// 软删除
		err = snapshotRepo.Delete(ctx, "snap-delete")
		assert.NoError(t, err)

		// 应该查询不到
		_, err = snapshotRepo.GetByID(ctx, "snap-delete")
		assert.Error(t, err)

		// 但可以通过 Unscoped 查询到
		deleted, err := snapshotRepo.GetByIDWithDeleted(ctx, "snap-delete")
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("HardDelete", func(t *testing.T) {
		snapshot := &model.Snapshot{
			ID:           "snap-hard-delete",
			VolumeID:     "vol-hard-delete",
			State:        "completed",
			StartTime:    time.Now(),
			Progress:     "100%",
			OwnerID:      "owner-hard-delete",
			VolumeSizeGB: 20,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := snapshotRepo.Create(ctx, snapshot)
		require.NoError(t, err)

		// 硬删除
		err = snapshotRepo.HardDelete(ctx, "snap-hard-delete")
		assert.NoError(t, err)

		// 应该完全查询不到
		_, err = snapshotRepo.GetByIDWithDeleted(ctx, "snap-hard-delete")
		assert.Error(t, err)
	})
}
