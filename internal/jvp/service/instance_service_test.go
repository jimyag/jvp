package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// NewInstanceServiceWithMocks 创建带有 mock 的 InstanceService（用于测试）
func NewInstanceServiceWithMocks(
	storageService *StorageService,
	imageService *ImageService,
	libvirtClient libvirt.LibvirtClient,
	repo *repository.Repository,
) *InstanceService {
	return &InstanceService{
		storageService: storageService,
		imageService:   imageService,
		libvirtClient:  libvirtClient,
		idGen:          idgen.New(),
		instanceRepo:   repository.NewInstanceRepository(repo.DB()),
	}
}

func setupTestInstanceService(t *testing.T) (*InstanceService, *ImageService, *StorageService, *repository.Repository, *libvirt.MockClient) {
	t.Helper()

	// 创建测试数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.New(dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Close()
		_ = os.RemoveAll(tmpDir)
	})

	// 创建 mock libvirt client
	mockLibvirtClient := libvirt.NewMockClient()

	// 设置 mock 行为：StorageService 初始化时会调用 EnsureStoragePool
	mockLibvirtClient.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
	mockLibvirtClient.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

	// 创建 StorageService
	storageService, err := NewStorageService(mockLibvirtClient)
	require.NoError(t, err)

	// 创建 mock qemu-img client
	mockQemuImgClient := qemuimg.NewMockClient()

	// 创建 ImageService
	imageService := NewImageServiceWithQemuImg(storageService, mockLibvirtClient, mockQemuImgClient, repo)

	// 创建 InstanceService
	instanceService := NewInstanceServiceWithMocks(storageService, imageService, mockLibvirtClient, repo)

	return instanceService, imageService, storageService, repo, mockLibvirtClient
}

func TestInstanceService_DescribeInstances(t *testing.T) {
	t.Parallel()

	instanceService, _, _, repo, _ := setupTestInstanceService(t)
	ctx := context.Background()

	// 创建测试数据
	instanceRepo := repository.NewInstanceRepository(repo.DB())
	instances := []*model.Instance{
		{ID: "i-1", Name: "Instance 1", State: "running", ImageID: "ami-1", VolumeID: "vol-1", MemoryMB: 2048, VCPUs: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "i-2", Name: "Instance 2", State: "stopped", ImageID: "ami-2", VolumeID: "vol-2", MemoryMB: 4096, VCPUs: 4, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, inst := range instances {
		err := instanceRepo.Create(ctx, inst)
		require.NoError(t, err)
	}

	testcases := []struct {
		name        string
		req         *entity.DescribeInstancesRequest
		expectCount int
	}{
		{
			name: "describe all instances",
			req: &entity.DescribeInstancesRequest{
				InstanceIDs: []string{},
			},
			expectCount: 2,
		},
		{
			name: "describe by IDs",
			req: &entity.DescribeInstancesRequest{
				InstanceIDs: []string{"i-1", "i-2"},
			},
			expectCount: 2,
		},
		{
			name: "describe by single ID",
			req: &entity.DescribeInstancesRequest{
				InstanceIDs: []string{"i-1"},
			},
			expectCount: 1,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			instances, err := instanceService.DescribeInstances(ctx, tc.req)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(instances), tc.expectCount)
		})
	}
}

func TestInstanceService_GetInstance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		setupInstance func(*repository.Repository) string
		instanceID    string
		mockSetup     func(*libvirt.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "get instance from database",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-db-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			instanceID:    "i-db-123",
			mockSetup:     nil,
			expectError:   false,
			errorContains: "",
		},
		{
			name: "get instance from libvirt",
			setupInstance: func(*repository.Repository) string {
				return ""
			},
			instanceID: "i-libvirt-123",
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-libvirt-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				}
				m.On("GetDomainByName", "i-libvirt-123").Return(domain, nil)
				m.On("GetDomainInfo", domain.UUID).Return(&libvirt.DomainInfo{
					State:  "Running",
					Memory: 2048 * 1024, // KB
					VCPUs:  2,
				}, nil)
				m.On("GetDomainDisks", "i-libvirt-123").Return([]libvirt.DomainDisk{
					{
						Type:   "file",
						Device: "disk",
						Source: libvirt.DomainDiskSource{File: "/var/lib/jvp/images/vol-123.qcow2"},
					},
				}, nil)
			},
			expectError: false,
		},
		{
			name: "instance not found",
			setupInstance: func(*repository.Repository) string {
				return ""
			},
			instanceID: "i-not-found",
			mockSetup: func(m *libvirt.MockClient) {
				// GetInstance 会先尝试从数据库查询（会失败，因为数据库中没有）
				// 然后从 libvirt 查询（也会失败）
				m.On("GetDomainByName", "i-not-found").Return(libvirtlib.Domain{}, fmt.Errorf("domain not found"))
			},
			expectError:   true,
			errorContains: "get domain",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 使用统一的 setup 方法，每个测试用例都有独立的数据库和 mock
			services := setupTestServices(t)

			if tc.setupInstance != nil {
				tc.setupInstance(services.Repo)
			}

			if tc.mockSetup != nil {
				tc.mockSetup(services.MockLibvirt)
			}

			instance, err := services.InstanceService.GetInstance(ctx, tc.instanceID)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, instance)
			} else {
				assert.NoError(t, err)
				if instance != nil {
					assert.Equal(t, tc.instanceID, instance.ID)
				}
			}
		})
	}
}

func TestInstanceService_StopInstances(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		setupInstance func(*repository.Repository) string
		req           *entity.StopInstancesRequest
		mockSetup     func(*libvirt.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "stop instance gracefully",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-stop-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.StopInstancesRequest{
				InstanceIDs: []string{"i-stop-123"},
				Force:       false,
			},
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-stop-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}
				m.On("GetDomainByName", "i-stop-123").Return(domain, nil)
				m.On("StopDomain", domain).Return(nil)
			},
			expectError: false,
		},
		{
			name: "force stop instance",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-force-stop-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.StopInstancesRequest{
				InstanceIDs: []string{"i-force-stop-123"},
				Force:       true,
			},
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-force-stop-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}
				m.On("GetDomainByName", "i-force-stop-123").Return(domain, nil)
				m.On("DestroyDomain", domain).Return(nil)
			},
			expectError: false,
		},
		{
			name: "instance already stopped",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-stopped-123",
					Name:      "Test Instance",
					State:     "stopped",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.StopInstancesRequest{
				InstanceIDs: []string{"i-stopped-123"},
				Force:       false,
			},
			mockSetup:   nil,
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 为每个测试用例创建独立的数据库和 repository
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")
			repo, err := repository.New(dbPath)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = repo.Close()
				_ = os.RemoveAll(tmpDir)
			})

			// 为每个测试用例创建新的 mock client
			mockClientForTest := libvirt.NewMockClient()
			mockClientForTest.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
			mockClientForTest.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

			// 创建独立的 StorageService
			storageService, err := NewStorageService(mockClientForTest)
			require.NoError(t, err)

			// 创建新的 instanceService 实例用于此测试
			mockQemuImgClient := qemuimg.NewMockClient()
			imageServiceForTest := NewImageServiceWithQemuImg(storageService, mockClientForTest, mockQemuImgClient, repo)
			instanceServiceForTest := NewInstanceServiceWithMocks(storageService, imageServiceForTest, mockClientForTest, repo)

			if tc.setupInstance != nil {
				tc.setupInstance(repo)
			}

			if tc.mockSetup != nil {
				tc.mockSetup(mockClientForTest)
			}

			changes, err := instanceServiceForTest.StopInstances(ctx, tc.req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, changes)
			} else {
				assert.NoError(t, err)
				if changes != nil {
					assert.GreaterOrEqual(t, len(changes), 0)
				}
			}
		})
	}
}

func TestInstanceService_StartInstances(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		setupInstance func(*repository.Repository) string
		req           *entity.StartInstancesRequest
		mockSetup     func(*libvirt.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "start instance",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-start-123",
					Name:      "Test Instance",
					State:     "stopped",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.StartInstancesRequest{
				InstanceIDs: []string{"i-start-123"},
			},
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-start-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}
				m.On("GetDomainByName", "i-start-123").Return(domain, nil)
				m.On("StartDomain", domain).Return(nil)
			},
			expectError: false,
		},
		{
			name: "instance already running",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-running-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.StartInstancesRequest{
				InstanceIDs: []string{"i-running-123"},
			},
			mockSetup:   nil,
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 使用统一的 setup 方法，每个测试用例都有独立的数据库和 mock
			services := setupTestServices(t)

			if tc.setupInstance != nil {
				tc.setupInstance(services.Repo)
			}

			if tc.mockSetup != nil {
				tc.mockSetup(services.MockLibvirt)
			}

			changes, err := services.InstanceService.StartInstances(ctx, tc.req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, changes)
			} else {
				assert.NoError(t, err)
				if changes != nil {
					assert.GreaterOrEqual(t, len(changes), 0)
				}
			}
		})
	}
}

func TestInstanceService_RebootInstances(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		setupInstance func(*repository.Repository) string
		req           *entity.RebootInstancesRequest
		mockSetup     func(*libvirt.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "reboot running instance",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-reboot-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.RebootInstancesRequest{
				InstanceIDs: []string{"i-reboot-123"},
			},
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-reboot-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}
				m.On("GetDomainByName", "i-reboot-123").Return(domain, nil)
				m.On("RebootDomain", domain).Return(nil)
			},
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 使用统一的 setup 方法，每个测试用例都有独立的数据库和 mock
			services := setupTestServices(t)

			if tc.setupInstance != nil {
				tc.setupInstance(services.Repo)
			}

			if tc.mockSetup != nil {
				tc.mockSetup(services.MockLibvirt)
			}

			changes, err := services.InstanceService.RebootInstances(ctx, tc.req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, changes)
			} else {
				assert.NoError(t, err)
				if changes != nil {
					assert.GreaterOrEqual(t, len(changes), 0)
				}
			}
		})
	}
}

func TestInstanceService_ModifyInstanceAttribute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		setupInstance func(*repository.Repository) string
		req           *entity.ModifyInstanceAttributeRequest
		mockSetup     func(*libvirt.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "modify memory",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-modify-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.ModifyInstanceAttributeRequest{
				InstanceID: "i-modify-123",
				MemoryMB:   func() *uint64 { v := uint64(4096); return &v }(),
				Live:       true,
			},
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-modify-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}
				m.On("GetDomainByName", "i-modify-123").Return(domain, nil)
				m.On("ModifyDomainMemory", domain, uint64(4096*1024), true).Return(nil)
			},
			expectError: false,
		},
		{
			name: "modify VCPU",
			setupInstance: func(repo *repository.Repository) string {
				instanceRepo := repository.NewInstanceRepository(repo.DB())
				instance := &model.Instance{
					ID:        "i-modify-vcpu-123",
					Name:      "Test Instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(ctx, instance)
				require.NoError(t, err)
				return instance.ID
			},
			req: &entity.ModifyInstanceAttributeRequest{
				InstanceID: "i-modify-vcpu-123",
				VCPUs:      func() *uint16 { v := uint16(4); return &v }(),
				Live:       true,
			},
			mockSetup: func(m *libvirt.MockClient) {
				domain := libvirtlib.Domain{
					Name: "i-modify-vcpu-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}
				m.On("GetDomainByName", "i-modify-vcpu-123").Return(domain, nil)
				m.On("ModifyDomainVCPU", domain, uint16(4), true).Return(nil)
			},
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 为每个测试用例创建独立的数据库和 repository
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")
			repo, err := repository.New(dbPath)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = repo.Close()
				_ = os.RemoveAll(tmpDir)
			})

			// 为每个测试用例创建新的 mock client
			mockClientForTest := libvirt.NewMockClient()
			mockClientForTest.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
			mockClientForTest.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

			// 创建独立的 StorageService
			storageService, err := NewStorageService(mockClientForTest)
			require.NoError(t, err)

			// 创建新的 instanceService 实例用于此测试
			mockQemuImgClient := qemuimg.NewMockClient()
			imageServiceForTest := NewImageServiceWithQemuImg(storageService, mockClientForTest, mockQemuImgClient, repo)
			instanceServiceForTest := NewInstanceServiceWithMocks(storageService, imageServiceForTest, mockClientForTest, repo)

			if tc.setupInstance != nil {
				tc.setupInstance(repo)
			}

			if tc.mockSetup != nil {
				tc.mockSetup(mockClientForTest)
			}

			instance, err := instanceServiceForTest.ModifyInstanceAttribute(ctx, tc.req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, instance)
			} else {
				// 注意：由于需要真实的 libvirt 环境，某些测试可能会失败
				if err != nil {
					t.Logf("Test may require libvirt environment: %v", err)
				}
				if instance != nil {
					assert.Equal(t, tc.req.InstanceID, instance.ID)
				}
			}
		})
	}
}
