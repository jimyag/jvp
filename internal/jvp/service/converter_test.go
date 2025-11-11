package service

import (
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstanceEntityToModel(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name      string
		instance  *entity.Instance
		expectNil bool
	}{
		{
			name: "convert instance with valid CreatedAt",
			instance: &entity.Instance{
				ID:        "i-123",
				Name:      "test-instance",
				State:     "running",
				ImageID:   "ami-123",
				VolumeID:  "vol-123",
				MemoryMB:  2048,
				VCPUs:     2,
				CreatedAt: time.Now().Format(time.RFC3339),
			},
			expectNil: false,
		},
		{
			name: "convert instance with empty CreatedAt",
			instance: &entity.Instance{
				ID:        "i-123",
				Name:      "test-instance",
				State:     "running",
				ImageID:   "ami-123",
				VolumeID:  "vol-123",
				MemoryMB:  2048,
				VCPUs:     2,
				CreatedAt: "",
			},
			expectNil: false,
		},
		{
			name: "convert instance with invalid CreatedAt",
			instance: &entity.Instance{
				ID:        "i-123",
				Name:      "test-instance",
				State:     "running",
				ImageID:   "ami-123",
				VolumeID:  "vol-123",
				MemoryMB:  2048,
				VCPUs:     2,
				CreatedAt: "invalid-date",
			},
			expectNil: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model, err := instanceEntityToModel(tc.instance)
			if tc.expectNil {
				assert.Nil(t, model)
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, model)
				assert.Equal(t, tc.instance.ID, model.ID)
				assert.Equal(t, tc.instance.Name, model.Name)
				assert.Equal(t, tc.instance.State, model.State)
				assert.Equal(t, tc.instance.ImageID, model.ImageID)
				assert.Equal(t, tc.instance.VolumeID, model.VolumeID)
				assert.Equal(t, tc.instance.MemoryMB, model.MemoryMB)
				assert.Equal(t, tc.instance.VCPUs, model.VCPUs)
				assert.NotZero(t, model.CreatedAt)
				assert.NotZero(t, model.UpdatedAt)
			}
		})
	}
}

func TestNewInstanceService(t *testing.T) {
	t.Parallel()

	t.Run("create instance service", func(t *testing.T) {
		t.Parallel()

		// 创建测试数据库
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/test.db"
		repo, err := repository.New(dbPath)
		require.NoError(t, err)
		defer repo.Close()

		// 创建 mock libvirt client
		mockLibvirtClient := libvirt.NewMockClient()
		mockLibvirtClient.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
		mockLibvirtClient.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

		// 创建 StorageService
		storageService, err := NewStorageService(mockLibvirtClient)
		require.NoError(t, err)

		// 创建 ImageService
		imageService, err := NewImageService(storageService, mockLibvirtClient, repo)
		require.NoError(t, err)

		// 创建 KeyPairService
		keyPairService := NewKeyPairService(repo)

		// 创建 InstanceService
		instanceService, err := NewInstanceService(storageService, imageService, keyPairService, mockLibvirtClient, repo)
		require.NoError(t, err)
		assert.NotNil(t, instanceService)
		assert.NotNil(t, instanceService.storageService)
		assert.NotNil(t, instanceService.imageService)
		assert.NotNil(t, instanceService.libvirtClient)
		assert.NotNil(t, instanceService.idGen)
		assert.NotNil(t, instanceService.instanceRepo)
	})
}

func TestMapLibvirtStateToInstanceState(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		libvirtState  string
		expectedState string
	}{
		{
			name:          "Running state",
			libvirtState:  "Running",
			expectedState: "running",
		},
		{
			name:          "ShutOff state",
			libvirtState:  "ShutOff",
			expectedState: "stopped",
		},
		{
			name:          "ShuttingDown state",
			libvirtState:  "ShuttingDown",
			expectedState: "stopping",
		},
		{
			name:          "Paused state",
			libvirtState:  "Paused",
			expectedState: "paused",
		},
		{
			name:          "Crashed state",
			libvirtState:  "Crashed",
			expectedState: "failed",
		},
		{
			name:          "unknown state",
			libvirtState:  "Unknown",
			expectedState: "pending",
		},
		{
			name:          "empty state",
			libvirtState:  "",
			expectedState: "pending",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := mapLibvirtStateToInstanceState(tc.libvirtState)
			assert.Equal(t, tc.expectedState, result)
		})
	}
}
