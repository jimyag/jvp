package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentRepository(t *testing.T) {
	t.Parallel()

	repo := setupTestDB(t)

	attachmentRepo := NewAttachmentRepository(repo.DB())
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		attachment := &model.VolumeAttachment{
			VolumeID:   "vol-123",
			InstanceID: "i-123",
			Device:     "/dev/vdb",
			State:      "attached",
			AttachTime: time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := attachmentRepo.Create(ctx, attachment)
		assert.NoError(t, err)
		require.NotZero(t, attachment.ID, "attachment ID should be set after creation")

		got, err := attachmentRepo.GetByID(ctx, attachment.ID)
		assert.NoError(t, err)
		assert.Equal(t, attachment.VolumeID, got.VolumeID)
		assert.Equal(t, attachment.InstanceID, got.InstanceID)
		assert.Equal(t, attachment.Device, got.Device)
	})

	t.Run("GetByVolumeID and GetByInstanceID", func(t *testing.T) {
		attachments := []*model.VolumeAttachment{
			{VolumeID: "vol-111", InstanceID: "i-111", Device: "/dev/vdb", State: "attached", AttachTime: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{VolumeID: "vol-111", InstanceID: "i-222", Device: "/dev/vdc", State: "attached", AttachTime: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{VolumeID: "vol-222", InstanceID: "i-111", Device: "/dev/vdd", State: "attached", AttachTime: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, att := range attachments {
			err := attachmentRepo.Create(ctx, att)
			require.NoError(t, err)
		}

		// 测试按 VolumeID 查询
		byVolume, err := attachmentRepo.GetByVolumeID(ctx, "vol-111")
		assert.NoError(t, err)
		assert.Len(t, byVolume, 2)

		// 测试按 InstanceID 查询
		byInstance, err := attachmentRepo.GetByInstanceID(ctx, "i-111")
		assert.NoError(t, err)
		assert.Len(t, byInstance, 2)
	})

	t.Run("GetByVolumeAndInstance", func(t *testing.T) {
		attachment := &model.VolumeAttachment{
			VolumeID:   "vol-456",
			InstanceID: "i-456",
			Device:     "/dev/vdb",
			State:      "attached",
			AttachTime: time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := attachmentRepo.Create(ctx, attachment)
		require.NoError(t, err)

		got, err := attachmentRepo.GetByVolumeAndInstance(ctx, "vol-456", "i-456")
		assert.NoError(t, err)
		assert.Equal(t, attachment.ID, got.ID)
	})

	t.Run("Update", func(t *testing.T) {
		attachment := &model.VolumeAttachment{
			VolumeID:   "vol-789",
			InstanceID: "i-789",
			Device:     "/dev/vdb",
			State:      "attaching",
			AttachTime: time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := attachmentRepo.Create(ctx, attachment)
		require.NoError(t, err)

		attachment.State = "attached"
		err = attachmentRepo.Update(ctx, attachment)
		assert.NoError(t, err)

		got, err := attachmentRepo.GetByID(ctx, attachment.ID)
		assert.NoError(t, err)
		assert.Equal(t, "attached", got.State)
	})

	t.Run("Delete by ID", func(t *testing.T) {
		attachment := &model.VolumeAttachment{
			VolumeID:   "vol-delete-id",
			InstanceID: "i-delete-id",
			Device:     "/dev/vdb",
			State:      "attached",
			AttachTime: time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := attachmentRepo.Create(ctx, attachment)
		require.NoError(t, err)
		require.NotZero(t, attachment.ID)

		// 测试按 ID 删除
		err = attachmentRepo.Delete(ctx, attachment.ID)
		assert.NoError(t, err)

		// 应该查询不到
		_, err = attachmentRepo.GetByID(ctx, attachment.ID)
		assert.Error(t, err)
	})

	t.Run("Delete and DeleteByVolumeAndInstance", func(t *testing.T) {
		attachment := &model.VolumeAttachment{
			VolumeID:   "vol-delete",
			InstanceID: "i-delete",
			Device:     "/dev/vdb",
			State:      "attached",
			AttachTime: time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := attachmentRepo.Create(ctx, attachment)
		require.NoError(t, err)

		// 测试 DeleteByVolumeAndInstance
		err = attachmentRepo.DeleteByVolumeAndInstance(ctx, "vol-delete", "i-delete")
		assert.NoError(t, err)

		// 应该查询不到
		_, err = attachmentRepo.GetByVolumeAndInstance(ctx, "vol-delete", "i-delete")
		assert.Error(t, err)
	})

	t.Run("HardDelete", func(t *testing.T) {
		attachment := &model.VolumeAttachment{
			VolumeID:   "vol-hard-delete",
			InstanceID: "i-hard-delete",
			Device:     "/dev/vdb",
			State:      "attached",
			AttachTime: time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := attachmentRepo.Create(ctx, attachment)
		require.NoError(t, err)
		require.NotZero(t, attachment.ID)

		// 硬删除
		err = attachmentRepo.HardDelete(ctx, attachment.ID)
		assert.NoError(t, err)

		// 应该完全查询不到
		_, err = attachmentRepo.GetByID(ctx, attachment.ID)
		assert.Error(t, err)
	})

	t.Run("List with filters", func(t *testing.T) {
		// 使用唯一的 ID 前缀避免与其他测试冲突
		attachments := []*model.VolumeAttachment{
			{VolumeID: "vol-filter-111", InstanceID: "i-filter-111", Device: "/dev/vdb", State: "attached", AttachTime: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{VolumeID: "vol-filter-222", InstanceID: "i-filter-222", Device: "/dev/vdc", State: "detaching", AttachTime: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{VolumeID: "vol-filter-333", InstanceID: "i-filter-333", Device: "/dev/vdd", State: "attached", AttachTime: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, att := range attachments {
			err := attachmentRepo.Create(ctx, att)
			require.NoError(t, err)
		}

		// 测试按状态过滤（只查询刚创建的数据）
		attached, err := attachmentRepo.List(ctx, map[string]interface{}{"state": "attached"})
		assert.NoError(t, err)
		filtered := make([]*model.VolumeAttachment, 0)
		for _, att := range attached {
			if att.VolumeID == "vol-filter-111" || att.VolumeID == "vol-filter-333" {
				filtered = append(filtered, att)
			}
		}
		assert.Len(t, filtered, 2)
	})
}
