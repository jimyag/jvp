package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageRepository(t *testing.T) {
	t.Parallel()

	repo := setupTestDB(t)

	imageRepo := NewImageRepository(repo.DB())
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		image := &model.Image{
			ID:        "ami-123",
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
		assert.NoError(t, err)

		got, err := imageRepo.GetByID(ctx, "ami-123")
		assert.NoError(t, err)
		assert.Equal(t, image.ID, got.ID)
		assert.Equal(t, image.Name, got.Name)
		assert.Equal(t, image.State, got.State)
	})

	t.Run("Update", func(t *testing.T) {
		image := &model.Image{
			ID:        "ami-456",
			Name:      "test-image-2",
			Pool:      "images",
			Path:      "/path/to/image2.qcow2",
			SizeGB:    20,
			Format:    "qcow2",
			State:     "available",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := imageRepo.Create(ctx, image)
		require.NoError(t, err)

		image.State = "pending"
		image.SizeGB = 30
		err = imageRepo.Update(ctx, image)
		assert.NoError(t, err)

		got, err := imageRepo.GetByID(ctx, "ami-456")
		assert.NoError(t, err)
		assert.Equal(t, "pending", got.State)
		assert.Equal(t, uint64(30), got.SizeGB)
	})

	t.Run("List with filters", func(t *testing.T) {
		// 使用唯一的 ID 前缀避免与其他测试冲突
		images := []*model.Image{
			{ID: "ami-filter-111", Name: "img1", Pool: "images", Path: "/path1", SizeGB: 10, Format: "qcow2", State: "available", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "ami-filter-222", Name: "img2", Pool: "images", Path: "/path2", SizeGB: 20, Format: "qcow2", State: "pending", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "ami-filter-333", Name: "img3", Pool: "snapshots", Path: "/path3", SizeGB: 30, Format: "qcow2", State: "available", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, img := range images {
			err := imageRepo.Create(ctx, img)
			require.NoError(t, err)
		}

		// 测试按状态过滤（只查询刚创建的数据）
		available, err := imageRepo.List(ctx, map[string]interface{}{"state": "available"})
		assert.NoError(t, err)
		filtered := make([]*model.Image, 0)
		for _, img := range available {
			if img.ID == "ami-filter-111" || img.ID == "ami-filter-333" {
				filtered = append(filtered, img)
			}
		}
		assert.Len(t, filtered, 2)

		// 测试按 pool 过滤
		poolImages, err := imageRepo.List(ctx, map[string]interface{}{"pool": "images"})
		assert.NoError(t, err)
		filtered = make([]*model.Image, 0)
		for _, img := range poolImages {
			if img.ID == "ami-filter-111" || img.ID == "ami-filter-222" {
				filtered = append(filtered, img)
			}
		}
		assert.Len(t, filtered, 2)
	})

	t.Run("Delete and soft delete", func(t *testing.T) {
		image := &model.Image{
			ID:        "ami-delete",
			Name:      "to-delete",
			Pool:      "images",
			Path:      "/path/to/delete.qcow2",
			SizeGB:    10,
			Format:    "qcow2",
			State:     "available",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := imageRepo.Create(ctx, image)
		require.NoError(t, err)

		// 软删除
		err = imageRepo.Delete(ctx, "ami-delete")
		assert.NoError(t, err)

		// 应该查询不到
		_, err = imageRepo.GetByID(ctx, "ami-delete")
		assert.Error(t, err)

		// 但可以通过 Unscoped 查询到
		deleted, err := imageRepo.GetByIDWithDeleted(ctx, "ami-delete")
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("HardDelete", func(t *testing.T) {
		image := &model.Image{
			ID:        "ami-hard-delete",
			Name:      "to-hard-delete",
			Pool:      "images",
			Path:      "/path/to/hard-delete.qcow2",
			SizeGB:    10,
			Format:    "qcow2",
			State:     "available",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := imageRepo.Create(ctx, image)
		require.NoError(t, err)

		// 硬删除
		err = imageRepo.HardDelete(ctx, "ami-hard-delete")
		assert.NoError(t, err)

		// 应该完全查询不到
		_, err = imageRepo.GetByIDWithDeleted(ctx, "ami-hard-delete")
		assert.Error(t, err)
	})
}
