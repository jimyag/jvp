package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVolumeRepository(t *testing.T) {
	t.Parallel()

	repo := setupTestDB(t)

	volumeRepo := NewVolumeRepository(repo.DB())
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		volume := &model.Volume{
			ID:         "vol-123",
			SizeGB:     20,
			State:      "available",
			VolumeType: "gp2",
			CreateTime: time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := volumeRepo.Create(ctx, volume)
		assert.NoError(t, err)

		got, err := volumeRepo.GetByID(ctx, "vol-123")
		assert.NoError(t, err)
		assert.Equal(t, volume.ID, got.ID)
		assert.Equal(t, volume.SizeGB, got.SizeGB)
		assert.Equal(t, volume.State, got.State)
	})

	t.Run("Update", func(t *testing.T) {
		volume := &model.Volume{
			ID:         "vol-456",
			SizeGB:     20,
			State:      "available",
			VolumeType: "gp2",
			CreateTime: time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := volumeRepo.Create(ctx, volume)
		require.NoError(t, err)

		volume.State = "in-use"
		volume.SizeGB = 30
		err = volumeRepo.Update(ctx, volume)
		assert.NoError(t, err)

		got, err := volumeRepo.GetByID(ctx, "vol-456")
		assert.NoError(t, err)
		assert.Equal(t, "in-use", got.State)
		assert.Equal(t, uint64(30), got.SizeGB)
	})

	t.Run("List with filters", func(t *testing.T) {
		// 使用唯一的 ID 前缀避免与其他测试冲突
		volumes := []*model.Volume{
			{ID: "vol-filter-111", SizeGB: 10, State: "available", VolumeType: "gp2", CreateTime: time.Now(), UpdatedAt: time.Now()},
			{ID: "vol-filter-222", SizeGB: 20, State: "in-use", VolumeType: "gp3", CreateTime: time.Now(), UpdatedAt: time.Now()},
			{ID: "vol-filter-333", SizeGB: 30, State: "available", VolumeType: "gp2", CreateTime: time.Now(), UpdatedAt: time.Now()},
		}

		for _, vol := range volumes {
			err := volumeRepo.Create(ctx, vol)
			require.NoError(t, err)
		}

		// 测试按状态过滤（只查询刚创建的数据）
		available, err := volumeRepo.List(ctx, map[string]interface{}{"state": "available"})
		assert.NoError(t, err)
		// 过滤出我们刚创建的卷
		filtered := make([]*model.Volume, 0)
		for _, v := range available {
			if v.ID == "vol-filter-111" || v.ID == "vol-filter-333" {
				filtered = append(filtered, v)
			}
		}
		assert.Len(t, filtered, 2)

		// 测试按 volume_type 过滤
		gp2, err := volumeRepo.List(ctx, map[string]interface{}{"volume_type": "gp2"})
		assert.NoError(t, err)
		// 过滤出我们刚创建的卷
		filtered = make([]*model.Volume, 0)
		for _, v := range gp2 {
			if v.ID == "vol-filter-111" || v.ID == "vol-filter-333" {
				filtered = append(filtered, v)
			}
		}
		assert.Len(t, filtered, 2)
	})

	t.Run("Delete and soft delete", func(t *testing.T) {
		volume := &model.Volume{
			ID:         "vol-delete",
			SizeGB:     10,
			State:      "available",
			VolumeType: "gp2",
			CreateTime: time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := volumeRepo.Create(ctx, volume)
		require.NoError(t, err)

		// 软删除
		err = volumeRepo.Delete(ctx, "vol-delete")
		assert.NoError(t, err)

		// 应该查询不到
		_, err = volumeRepo.GetByID(ctx, "vol-delete")
		assert.Error(t, err)

		// 但可以通过 Unscoped 查询到
		deleted, err := volumeRepo.GetByIDWithDeleted(ctx, "vol-delete")
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("HardDelete", func(t *testing.T) {
		volume := &model.Volume{
			ID:         "vol-hard-delete",
			SizeGB:     10,
			State:      "available",
			VolumeType: "gp2",
			CreateTime: time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := volumeRepo.Create(ctx, volume)
		require.NoError(t, err)

		// 硬删除
		err = volumeRepo.HardDelete(ctx, "vol-hard-delete")
		assert.NoError(t, err)

		// 应该完全查询不到
		_, err = volumeRepo.GetByIDWithDeleted(ctx, "vol-hard-delete")
		assert.Error(t, err)
	})
}
