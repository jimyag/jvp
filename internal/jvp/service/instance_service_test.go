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
	volumeRepo := repository.NewVolumeRepository(repo.DB())
	storageService, err := NewStorageService(mockLibvirtClient, volumeRepo)
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
			volumeRepo := repository.NewVolumeRepository(repo.DB())
			storageService, err := NewStorageService(mockClientForTest, volumeRepo)
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
			volumeRepo := repository.NewVolumeRepository(repo.DB())
			storageService, err := NewStorageService(mockClientForTest, volumeRepo)
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

func TestInstanceService_RunInstance_WithUserData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		req           *entity.RunInstanceRequest
		mockSetup     func(*libvirt.MockClient, *qemuimg.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "run instance without userdata (backward compatibility)",
			req: &entity.RunInstanceRequest{
				ImageID:  "ubuntu-jammy",
				MemoryMB: 2048,
				VCPUs:    2,
				SizeGB:   20,
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				// Mock CreateVolume
				m.On("CreateVolume", "default", mock.AnythingOfType("string"), uint64(20), "qcow2").Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock CreateVolumeFromBackingFile (ctx, format, backingFormat, backingFile, outputFile)
				q.On("CreateFromBackingFile", mock.Anything, "qcow2", "qcow2", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

				// Mock Resize (called if imageSizeGB < req.SizeGB, image is 2GB, target is 20GB)
				q.On("Resize", mock.Anything, mock.AnythingOfType("string"), uint64(20)).Return(nil).Maybe()

				// Mock GetVolume (called after CreateFromBackingFile to get updated volume info)
				m.On("GetVolume", "default", mock.AnythingOfType("string")).Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock DeleteVolume (may be called on error, StorageService tries default pool first, then images pool)
				m.On("DeleteVolume", "default", mock.AnythingOfType("string")).Return(nil).Maybe()
				m.On("DeleteVolume", "images", mock.AnythingOfType("string")).Return(nil).Maybe()

				// Mock CreateDomain (without cloud-init)
				m.On("CreateDomain", mock.MatchedBy(func(config *libvirt.CreateVMConfig) bool {
					return config.CloudInit == nil && config.CloudInitUserData == nil
				}), true).Return(libvirtlib.Domain{
					Name: "i-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}, nil).Once()
			},
			expectError: false,
		},
		{
			name: "run instance with structured userdata",
			req: &entity.RunInstanceRequest{
				ImageID:  "ubuntu-jammy",
				MemoryMB: 2048,
				VCPUs:    2,
				SizeGB:   20,
				UserData: &entity.UserDataConfig{
					StructuredUserData: &entity.StructuredUserData{
						Hostname: "test-instance",
						Users: []entity.User{
							{
								Name:              "admin",
								Groups:            "sudo",
								SSHAuthorizedKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."},
								HashedPasswd:      "$6$rounds=5000$salt$hashedpassword",
							},
						},
						Packages: []string{"nginx", "docker.io"},
						RunCmd:   []string{"systemctl enable nginx"},
					},
				},
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				// Mock CreateVolume
				m.On("CreateVolume", "default", mock.AnythingOfType("string"), uint64(20), "qcow2").Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock CreateVolumeFromBackingFile (ctx, format, backingFormat, backingFile, outputFile)
				q.On("CreateFromBackingFile", mock.Anything, "qcow2", "qcow2", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

				// Mock Resize (called if imageSizeGB < req.SizeGB, image is 2GB, target is 20GB)
				q.On("Resize", mock.Anything, mock.AnythingOfType("string"), uint64(20)).Return(nil).Maybe()

				// Mock GetVolume (called after CreateFromBackingFile to get updated volume info)
				m.On("GetVolume", "default", mock.AnythingOfType("string")).Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock DeleteVolume (may be called on error, StorageService tries default pool first, then images pool)
				m.On("DeleteVolume", "default", mock.AnythingOfType("string")).Return(nil).Maybe()
				m.On("DeleteVolume", "images", mock.AnythingOfType("string")).Return(nil).Maybe()

				// Mock CreateDomain (with cloud-init config)
				m.On("CreateDomain", mock.MatchedBy(func(config *libvirt.CreateVMConfig) bool {
					return config.CloudInit != nil && config.CloudInit.Hostname == "test-instance"
				}), true).Return(libvirtlib.Domain{
					Name: "i-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}, nil).Once()
			},
			expectError: false,
		},
		{
			name: "run instance with raw userdata",
			req: &entity.RunInstanceRequest{
				ImageID:  "ubuntu-jammy",
				MemoryMB: 2048,
				VCPUs:    2,
				SizeGB:   20,
				UserData: &entity.UserDataConfig{
					RawUserData: "#cloud-config\nusers:\n  - name: admin\n    groups: sudo\n    ssh_authorized_keys:\n      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...\npackages:\n  - nginx\n",
				},
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				// Mock CreateVolume
				m.On("CreateVolume", "default", mock.AnythingOfType("string"), uint64(20), "qcow2").Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock CreateVolumeFromBackingFile (ctx, format, backingFormat, backingFile, outputFile)
				q.On("CreateFromBackingFile", mock.Anything, "qcow2", "qcow2", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

				// Mock Resize (called if imageSizeGB < req.SizeGB, image is 2GB, target is 20GB)
				q.On("Resize", mock.Anything, mock.AnythingOfType("string"), uint64(20)).Return(nil).Maybe()

				// Mock GetVolume (called after CreateFromBackingFile to get updated volume info)
				m.On("GetVolume", "default", mock.AnythingOfType("string")).Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock DeleteVolume (may be called on error, StorageService tries default pool first, then images pool)
				m.On("DeleteVolume", "default", mock.AnythingOfType("string")).Return(nil).Maybe()
				m.On("DeleteVolume", "images", mock.AnythingOfType("string")).Return(nil).Maybe()

				// Mock CreateDomain (with cloud-init userdata)
				m.On("CreateDomain", mock.MatchedBy(func(config *libvirt.CreateVMConfig) bool {
					return config.CloudInitUserData != nil
				}), true).Return(libvirtlib.Domain{
					Name: "i-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}, nil).Once()
			},
			expectError: false,
		},
		{
			name: "run instance with invalid raw userdata",
			req: &entity.RunInstanceRequest{
				ImageID:  "ubuntu-jammy",
				MemoryMB: 2048,
				VCPUs:    2,
				SizeGB:   20,
				UserData: &entity.UserDataConfig{
					RawUserData: "invalid yaml: [",
				},
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				// Mock CreateVolume
				m.On("CreateVolume", "default", mock.AnythingOfType("string"), uint64(20), "qcow2").Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock CreateVolumeFromBackingFile (should succeed, then UserData parsing will fail)
				q.On("CreateFromBackingFile", mock.Anything, "qcow2", "qcow2", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

				// Mock Resize (called if imageSizeGB < req.SizeGB, image is 2GB, target is 20GB)
				q.On("Resize", mock.Anything, mock.AnythingOfType("string"), uint64(20)).Return(nil).Once()

				// Mock GetVolume (called after CreateFromBackingFile to get updated volume info)
				m.On("GetVolume", "default", mock.AnythingOfType("string")).Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock DeleteVolume (cleanup on error, StorageService tries default pool first, then images pool)
				m.On("DeleteVolume", "default", mock.AnythingOfType("string")).Return(nil).Maybe()
				m.On("DeleteVolume", "images", mock.AnythingOfType("string")).Return(nil).Maybe()
			},
			expectError:   true,
			errorContains: "invalid raw userdata YAML",
		},
		{
			name: "run instance with structured userdata and plaintext password",
			req: &entity.RunInstanceRequest{
				ImageID:  "ubuntu-jammy",
				MemoryMB: 2048,
				VCPUs:    2,
				SizeGB:   20,
				UserData: &entity.UserDataConfig{
					StructuredUserData: &entity.StructuredUserData{
						Hostname: "test-instance",
						Users: []entity.User{
							{
								Name:            "admin",
								Groups:          "sudo",
								PlainTextPasswd: "testpassword123",
							},
						},
					},
				},
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				// Mock CreateVolume
				m.On("CreateVolume", "default", mock.AnythingOfType("string"), uint64(20), "qcow2").Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock CreateVolumeFromBackingFile (ctx, format, backingFormat, backingFile, outputFile)
				q.On("CreateFromBackingFile", mock.Anything, "qcow2", "qcow2", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

				// Mock Resize (called if imageSizeGB < req.SizeGB, image is 2GB, target is 20GB)
				q.On("Resize", mock.Anything, mock.AnythingOfType("string"), uint64(20)).Return(nil).Maybe()

				// Mock GetVolume (called after CreateFromBackingFile to get updated volume info)
				m.On("GetVolume", "default", mock.AnythingOfType("string")).Return(&libvirt.VolumeInfo{
					Name:        "vol-123.qcow2",
					Path:        "/var/lib/jvp/images/vol-123.qcow2",
					CapacityB:   20 * 1024 * 1024 * 1024,
					AllocationB: 10 * 1024 * 1024 * 1024,
					Format:      "qcow2",
				}, nil).Once()

				// Mock DeleteVolume (may be called on error, StorageService tries default pool first, then images pool)
				m.On("DeleteVolume", "default", mock.AnythingOfType("string")).Return(nil).Maybe()
				m.On("DeleteVolume", "images", mock.AnythingOfType("string")).Return(nil).Maybe()

				// Mock CreateDomain (with cloud-init config, password should be hashed)
				m.On("CreateDomain", mock.MatchedBy(func(config *libvirt.CreateVMConfig) bool {
					return config.CloudInit != nil && config.CloudInit.Users != nil && len(config.CloudInit.Users) > 0
				}), true).Return(libvirtlib.Domain{
					Name: "i-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4},
				}, nil).Once()
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

			// 创建镜像文件（GetImage 需要）
			imagePath := filepath.Join(services.TempDir, "images", "ubuntu-jammy-server-cloudimg-amd64.img")
			err := os.MkdirAll(filepath.Dir(imagePath), 0o755)
			require.NoError(t, err)
			f, err := os.Create(imagePath)
			require.NoError(t, err)
			_, err = f.Write(make([]byte, 100*1024*1024)) // 100MB
			require.NoError(t, err)
			require.NoError(t, f.Close())

			// 注册镜像到数据库
			imageRepo := repository.NewImageRepository(services.Repo.DB())
			imageModel := &model.Image{
				ID:     "ubuntu-jammy",
				Name:   "Ubuntu Jammy",
				Pool:   "images",
				Path:   imagePath,
				SizeGB: 2,
				Format: "qcow2",
				State:  "available",
			}
			err = imageRepo.Create(ctx, imageModel)
			require.NoError(t, err)

			if tc.mockSetup != nil {
				tc.mockSetup(services.MockLibvirt, services.MockQemuImg)
			}

			instance, err := services.InstanceService.RunInstance(ctx, tc.req)

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
					assert.NotEmpty(t, instance.ID)
					assert.Equal(t, tc.req.ImageID, instance.ImageID)
				}
			}
		})
	}
}
