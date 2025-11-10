package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagRepository(t *testing.T) {
	t.Parallel()

	repo := setupTestDB(t)

	tagRepo := NewTagRepository(repo.DB())
	ctx := context.Background()

	t.Run("Create and GetByResource", func(t *testing.T) {
		tags := []*model.Tag{
			{ResourceType: "instance", ResourceID: "i-123", TagKey: "Name", TagValue: "test-instance", CreatedAt: time.Now()},
			{ResourceType: "instance", ResourceID: "i-123", TagKey: "Environment", TagValue: "production", CreatedAt: time.Now()},
			{ResourceType: "volume", ResourceID: "vol-123", TagKey: "Name", TagValue: "test-volume", CreatedAt: time.Now()},
		}

		for _, tag := range tags {
			err := tagRepo.Create(ctx, tag)
			require.NoError(t, err)
		}

		// 获取 instance 的所有标签
		instanceTags, err := tagRepo.GetByResource(ctx, "instance", "i-123")
		assert.NoError(t, err)
		assert.Len(t, instanceTags, 2)

		// 获取 volume 的所有标签
		volumeTags, err := tagRepo.GetByResource(ctx, "volume", "vol-123")
		assert.NoError(t, err)
		assert.Len(t, volumeTags, 1)
	})

	t.Run("GetByKey", func(t *testing.T) {
		tag := &model.Tag{
			ResourceType: "instance",
			ResourceID:   "i-456",
			TagKey:       "Name",
			TagValue:     "test-instance-2",
			CreatedAt:    time.Now(),
		}

		err := tagRepo.Create(ctx, tag)
		require.NoError(t, err)

		got, err := tagRepo.GetByKey(ctx, "instance", "i-456", "Name")
		assert.NoError(t, err)
		assert.Equal(t, "test-instance-2", got.TagValue)
	})

	t.Run("Update", func(t *testing.T) {
		tag := &model.Tag{
			ResourceType: "instance",
			ResourceID:   "i-789",
			TagKey:       "Name",
			TagValue:     "old-value",
			CreatedAt:    time.Now(),
		}

		err := tagRepo.Create(ctx, tag)
		require.NoError(t, err)

		tag.TagValue = "new-value"
		err = tagRepo.Update(ctx, tag)
		assert.NoError(t, err)

		got, err := tagRepo.GetByKey(ctx, "instance", "i-789", "Name")
		assert.NoError(t, err)
		assert.Equal(t, "new-value", got.TagValue)
	})

	t.Run("Delete", func(t *testing.T) {
		tag := &model.Tag{
			ResourceType: "instance",
			ResourceID:   "i-delete",
			TagKey:       "Name",
			TagValue:     "to-delete",
			CreatedAt:    time.Now(),
		}

		err := tagRepo.Create(ctx, tag)
		require.NoError(t, err)

		// 软删除
		err = tagRepo.Delete(ctx, "instance", "i-delete", "Name")
		assert.NoError(t, err)

		// 应该查询不到
		_, err = tagRepo.GetByKey(ctx, "instance", "i-delete", "Name")
		assert.Error(t, err)
	})

	t.Run("DeleteByResource", func(t *testing.T) {
		tags := []*model.Tag{
			{ResourceType: "instance", ResourceID: "i-batch", TagKey: "Name", TagValue: "test", CreatedAt: time.Now()},
			{ResourceType: "instance", ResourceID: "i-batch", TagKey: "Environment", TagValue: "test", CreatedAt: time.Now()},
		}

		for _, tag := range tags {
			err := tagRepo.Create(ctx, tag)
			require.NoError(t, err)
		}

		// 删除资源的所有标签
		err := tagRepo.DeleteByResource(ctx, "instance", "i-batch")
		assert.NoError(t, err)

		// 应该查询不到任何标签
		got, err := tagRepo.GetByResource(ctx, "instance", "i-batch")
		assert.NoError(t, err)
		assert.Len(t, got, 0)
	})

	t.Run("HardDelete", func(t *testing.T) {
		tag := &model.Tag{
			ResourceType: "instance",
			ResourceID:   "i-hard-delete",
			TagKey:       "Name",
			TagValue:     "to-hard-delete",
			CreatedAt:    time.Now(),
		}

		err := tagRepo.Create(ctx, tag)
		require.NoError(t, err)

		// 硬删除
		err = tagRepo.HardDelete(ctx, "instance", "i-hard-delete", "Name")
		assert.NoError(t, err)

		// 应该完全查询不到
		_, err = tagRepo.GetByKey(ctx, "instance", "i-hard-delete", "Name")
		assert.Error(t, err)
	})
}
