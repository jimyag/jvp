package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	tmpDir := t.TempDir()
	// 使用简单的数据库文件名
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := New(dbPath)
	require.NoError(t, err)

	// 使用 t.Cleanup 确保在测试真正结束时清理，支持并发测试
	t.Cleanup(func() {
		_ = repo.Close()
		_ = os.RemoveAll(tmpDir)
	})

	return repo
}

func TestInstanceRepository(t *testing.T) {
	t.Parallel()

	repo := setupTestDB(t)

	instanceRepo := NewInstanceRepository(repo.DB())
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		instance := &model.Instance{
			ID:        "i-123456",
			Name:      "test-instance",
			State:     "running",
			ImageID:   "ami-123",
			VolumeID:  "vol-123",
			MemoryMB:  2048,
			VCPUs:     2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := instanceRepo.Create(ctx, instance)
		assert.NoError(t, err)

		got, err := instanceRepo.GetByID(ctx, "i-123456")
		assert.NoError(t, err)
		assert.Equal(t, instance.ID, got.ID)
		assert.Equal(t, instance.Name, got.Name)
		assert.Equal(t, instance.State, got.State)
	})

	t.Run("Update", func(t *testing.T) {
		instance := &model.Instance{
			ID:        "i-789012",
			Name:      "test-instance-2",
			State:     "running",
			MemoryMB:  1024,
			VCPUs:     1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := instanceRepo.Create(ctx, instance)
		require.NoError(t, err)

		instance.State = "stopped"
		instance.MemoryMB = 4096
		err = instanceRepo.Update(ctx, instance)
		assert.NoError(t, err)

		got, err := instanceRepo.GetByID(ctx, "i-789012")
		assert.NoError(t, err)
		assert.Equal(t, "stopped", got.State)
		assert.Equal(t, uint64(4096), got.MemoryMB)
	})

	t.Run("List with filters", func(t *testing.T) {
		// 使用唯一的 ID 前缀避免与其他测试冲突
		instances := []*model.Instance{
			{ID: "i-filter-111", Name: "inst1", State: "running", MemoryMB: 1024, VCPUs: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "i-filter-222", Name: "inst2", State: "stopped", MemoryMB: 2048, VCPUs: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "i-filter-333", Name: "inst3", State: "running", MemoryMB: 4096, VCPUs: 4, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, inst := range instances {
			err := instanceRepo.Create(ctx, inst)
			require.NoError(t, err)
		}

		// 测试按状态过滤（只查询刚创建的数据）
		running, err := instanceRepo.List(ctx, map[string]interface{}{"state": "running"})
		assert.NoError(t, err)
		filtered := make([]*model.Instance, 0)
		for _, inst := range running {
			if inst.ID == "i-filter-111" || inst.ID == "i-filter-333" {
				filtered = append(filtered, inst)
			}
		}
		assert.Len(t, filtered, 2)

		// 测试按 image_id 过滤
		instances[0].ImageID = "ami-123"
		err = instanceRepo.Update(ctx, instances[0])
		require.NoError(t, err)

		byImage, err := instanceRepo.List(ctx, map[string]interface{}{"image_id": "ami-123"})
		assert.NoError(t, err)
		filtered = make([]*model.Instance, 0)
		for _, inst := range byImage {
			if inst.ID == "i-filter-111" {
				filtered = append(filtered, inst)
			}
		}
		assert.Len(t, filtered, 1)
		assert.Equal(t, "i-filter-111", filtered[0].ID)
	})

	t.Run("Delete and soft delete", func(t *testing.T) {
		instance := &model.Instance{
			ID:        "i-delete",
			Name:      "to-delete",
			State:     "running",
			MemoryMB:  1024,
			VCPUs:     1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := instanceRepo.Create(ctx, instance)
		require.NoError(t, err)

		// 软删除
		err = instanceRepo.Delete(ctx, "i-delete")
		assert.NoError(t, err)

		// 应该查询不到（已软删除）
		_, err = instanceRepo.GetByID(ctx, "i-delete")
		assert.Error(t, err)

		// 但可以通过 Unscoped 查询到
		deleted, err := instanceRepo.GetByIDWithDeleted(ctx, "i-delete")
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("HardDelete", func(t *testing.T) {
		instance := &model.Instance{
			ID:        "i-hard-delete",
			Name:      "to-hard-delete",
			State:     "running",
			MemoryMB:  1024,
			VCPUs:     1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := instanceRepo.Create(ctx, instance)
		require.NoError(t, err)

		// 硬删除
		err = instanceRepo.HardDelete(ctx, "i-hard-delete")
		assert.NoError(t, err)

		// 应该完全查询不到
		_, err = instanceRepo.GetByIDWithDeleted(ctx, "i-hard-delete")
		assert.Error(t, err)
	})

	t.Run("GetByIDWithRelations", func(t *testing.T) {
		// 创建关联数据
		imageRepo := NewImageRepository(repo.DB())
		image := &model.Image{
			ID:        "ami-456",
			Name:      "test-image",
			Pool:      "images",
			Path:      "/path/to/image.qcow2",
			SizeGB:    10,
			Format:    "qcow2",
			State:     "available",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := imageRepo.Create(ctx, image)
		require.NoError(t, err)

		volumeRepo := NewVolumeRepository(repo.DB())
		volume := &model.Volume{
			ID:         "vol-456",
			SizeGB:     20,
			State:      "available",
			VolumeType: "gp2",
			CreateTime: time.Now(),
			UpdatedAt:  time.Now(),
		}
		err = volumeRepo.Create(ctx, volume)
		require.NoError(t, err)

		// 创建实例
		instance := &model.Instance{
			ID:        "i-relations",
			Name:      "test-relations",
			State:     "running",
			ImageID:   "ami-456",
			VolumeID:  "vol-456",
			MemoryMB:  2048,
			VCPUs:     2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = instanceRepo.Create(ctx, instance)
		require.NoError(t, err)

		// 创建标签
		tagRepo := NewTagRepository(repo.DB())
		tag := &model.Tag{
			ResourceType: "instance",
			ResourceID:   "i-relations",
			TagKey:       "Name",
			TagValue:     "test-instance",
			CreatedAt:    time.Now(),
		}
		err = tagRepo.Create(ctx, tag)
		require.NoError(t, err)

		// 测试获取关联数据
		withRelations, err := instanceRepo.GetByIDWithRelations(ctx, "i-relations")
		assert.NoError(t, err)
		assert.NotNil(t, withRelations.Instance)
		assert.NotNil(t, withRelations.Image)
		assert.NotNil(t, withRelations.Volume)
		assert.Len(t, withRelations.Tags, 1)
		assert.Equal(t, "ami-456", withRelations.Image.ID)
		assert.Equal(t, "vol-456", withRelations.Volume.ID)
	})
}
