package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// NewImageServiceWithQemuImg 创建带有 mock qemu-img client 的 ImageService（用于测试）
func NewImageServiceWithQemuImg(
	storageService *StorageService,
	libvirtClient libvirt.LibvirtClient,
	qemuImgClient qemuimg.QemuImgClient,
	repo *repository.Repository,
) *ImageService {
	return &ImageService{
		storageService: storageService,
		libvirtClient:  libvirtClient,
		qemuImgClient:  qemuImgClient,
		idGen:          idgen.New(),
		httpClient:     &http.Client{Timeout: 30 * time.Minute},
		imageRepo:      repository.NewImageRepository(repo.DB()),
		imagesPoolName: "images",
		imagesPoolPath: "/var/lib/jvp/images/images",
	}
}

func setupTestImageService(t *testing.T) (*ImageService, *repository.Repository, *libvirt.MockClient, *qemuimg.MockClient) {
	t.Helper()

	// 创建测试数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.New(dbPath)
	require.NoError(t, err)

	// 创建测试镜像目录
	imagesDir := filepath.Join(tmpDir, "images")
	err = os.MkdirAll(imagesDir, 0o755)
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
	// 设置测试镜像路径
	imageService.imagesPoolPath = imagesDir

	return imageService, repo, mockLibvirtClient, mockQemuImgClient
}

func TestImageService_RegisterImage(t *testing.T) {
	t.Parallel()

	imageService, repo, mockClient, _ := setupTestImageService(t)
	_ = mockClient
	ctx := context.Background()

	testcases := []struct {
		name          string
		setupFile     func() string
		req           *entity.RegisterImageRequest
		expectError   bool
		errorContains string
	}{
		{
			name: "successful register",
			setupFile: func() string {
				tmpDir := t.TempDir()
				imagePath := filepath.Join(tmpDir, "test-image.qcow2")
				// 创建一个测试文件
				f, err := os.Create(imagePath)
				require.NoError(t, err)
				// 写入一些数据使其大小 > 0（100MB 足够测试）
				_, err = f.Write(make([]byte, 100*1024*1024)) // 100MB
				require.NoError(t, err)
				require.NoError(t, f.Close())
				return imagePath
			},
			req: &entity.RegisterImageRequest{
				Name:        "test-image",
				Description: "Test image",
				Path:        "",
				Pool:        "images",
			},
			expectError: false,
		},
		{
			name: "file not found",
			setupFile: func() string {
				return "/nonexistent/path/image.qcow2"
			},
			req: &entity.RegisterImageRequest{
				Name: "test-image",
				Path: "/nonexistent/path/image.qcow2",
			},
			expectError:   true,
			errorContains: "not found",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.setupFile != nil {
				imagePath := tc.setupFile()
				tc.req.Path = imagePath
			}

			image, err := imageService.RegisterImage(ctx, tc.req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, image)
			} else {
				assert.NoError(t, err)
				if image != nil {
					assert.NotEmpty(t, image.ID)
					assert.Equal(t, tc.req.Name, image.Name)
					// 验证已保存到数据库
					imageRepo := repository.NewImageRepository(repo.DB())
					imageModel, err := imageRepo.GetByID(ctx, image.ID)
					assert.NoError(t, err)
					assert.NotNil(t, imageModel)
				}
			}
		})
	}
}

func TestImageService_GetImage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		setupImage    func(*repository.Repository, string) string
		imageID       string
		expectError   bool
		errorContains string
	}{
		{
			name: "get image from database",
			setupImage: func(repo *repository.Repository, imagesDir string) string {
				imageRepo := repository.NewImageRepository(repo.DB())
				image := &model.Image{
					ID:          "ami-test-123",
					Name:        "Test Image",
					Description: "Test description",
					Pool:        "images",
					Path:        "/var/lib/jvp/images/images/test.qcow2",
					SizeGB:      20,
					Format:      "qcow2",
					State:       "available",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err := imageRepo.Create(ctx, image)
				require.NoError(t, err)
				return image.ID
			},
			imageID:     "ami-test-123",
			expectError: false,
		},
		{
			name: "get default image",
			setupImage: func(*repository.Repository, string) string {
				// 这个测试用例会在子测试中创建文件
				return "ubuntu-jammy"
			},
			imageID:     "ubuntu-jammy",
			expectError: false,
		},
		{
			name: "image not found",
			setupImage: func(*repository.Repository, string) string {
				return ""
			},
			imageID:       "ami-not-found",
			expectError:   true,
			errorContains: "not found",
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

			// 创建独立的 mock client
			mockClientForTest := libvirt.NewMockClient()
			mockClientForTest.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
			mockClientForTest.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

			// 创建独立的 StorageService
			volumeRepo := repository.NewVolumeRepository(repo.DB())
			storageService, err := NewStorageService(mockClientForTest, volumeRepo)
			require.NoError(t, err)

			// 创建独立的 mock qemu-img client
			mockQemuImgClientForTest := qemuimg.NewMockClient()

			// 创建独立的 ImageService
			imageServiceForTest := NewImageServiceWithQemuImg(storageService, mockClientForTest, mockQemuImgClientForTest, repo)

			// 创建测试镜像目录
			imagesDir := filepath.Join(tmpDir, "images")
			err = os.MkdirAll(imagesDir, 0o755)
			require.NoError(t, err)
			imageServiceForTest.imagesPoolPath = imagesDir

			// 对于 "get default image" 测试用例，需要创建镜像文件以避免下载
			if tc.imageID == "ubuntu-jammy" {
				imagePath := filepath.Join(imagesDir, "ubuntu-jammy-server-cloudimg-amd64.img")
				f, err := os.Create(imagePath)
				require.NoError(t, err)
				_, err = f.Write(make([]byte, 100*1024*1024)) // 100MB，足够测试
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			if tc.setupImage != nil {
				tc.setupImage(repo, imagesDir)
			}

			image, err := imageServiceForTest.GetImage(ctx, tc.imageID)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, image)
			} else {
				assert.NoError(t, err)
				if image != nil {
					assert.Equal(t, tc.imageID, image.ID)
				}
			}
		})
	}
}

func TestImageService_ListImages(t *testing.T) {
	t.Parallel()

	imageService, repo, mockClient, _ := setupTestImageService(t)
	_ = mockClient
	ctx := context.Background()

	// 创建测试数据
	imageRepo := repository.NewImageRepository(repo.DB())
	images := []*model.Image{
		{ID: "ami-1", Name: "Image 1", Pool: "images", Path: "/path/1.qcow2", SizeGB: 10, Format: "qcow2", State: "available", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ami-2", Name: "Image 2", Pool: "images", Path: "/path/2.qcow2", SizeGB: 20, Format: "qcow2", State: "available", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, img := range images {
		err := imageRepo.Create(ctx, img)
		require.NoError(t, err)
	}

	imageList, err := imageService.ListImages(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(imageList), 2) // 至少包含数据库中的镜像
}

func TestImageService_DescribeImages(t *testing.T) {
	t.Parallel()

	imageService, repo, mockClient, _ := setupTestImageService(t)
	_ = mockClient
	ctx := context.Background()

	// 创建测试数据
	imageRepo := repository.NewImageRepository(repo.DB())
	images := []*model.Image{
		{ID: "ami-desc-1", Name: "Image 1", Pool: "images", Path: "/path/1.qcow2", SizeGB: 10, Format: "qcow2", State: "available", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ami-desc-2", Name: "Image 2", Pool: "images", Path: "/path/2.qcow2", SizeGB: 20, Format: "qcow2", State: "available", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, img := range images {
		err := imageRepo.Create(ctx, img)
		require.NoError(t, err)
	}

	testcases := []struct {
		name        string
		req         *entity.DescribeImagesRequest
		expectCount int
	}{
		{
			name: "describe all images",
			req: &entity.DescribeImagesRequest{
				ImageIDs: []string{},
			},
			expectCount: 2,
		},
		{
			name: "describe by IDs",
			req: &entity.DescribeImagesRequest{
				ImageIDs: []string{"ami-desc-1", "ami-desc-2"},
			},
			expectCount: 2,
		},
		{
			name: "describe by single ID",
			req: &entity.DescribeImagesRequest{
				ImageIDs: []string{"ami-desc-1"},
			},
			expectCount: 1,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			images, err := imageService.DescribeImages(ctx, tc.req)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(images), tc.expectCount)
		})
	}
}

func TestImageService_DeleteImage(t *testing.T) {
	t.Parallel()

	imageService, repo, mockClient, _ := setupTestImageService(t)
	_ = mockClient
	ctx := context.Background()

	testcases := []struct {
		name          string
		setupImage    func() string
		imageID       string
		expectError   bool
		errorContains string
	}{
		{
			name: "successful delete",
			setupImage: func() string {
				imageRepo := repository.NewImageRepository(repo.DB())
				image := &model.Image{
					ID:        "ami-delete-123",
					Name:      "Test Image",
					Pool:      "images",
					Path:      "/tmp/test-image.qcow2",
					SizeGB:    20,
					Format:    "qcow2",
					State:     "available",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := imageRepo.Create(ctx, image)
				require.NoError(t, err)
				return image.ID
			},
			imageID:     "ami-delete-123",
			expectError: false,
		},
		{
			name: "image not found",
			setupImage: func() string {
				return ""
			},
			imageID:       "ami-not-found",
			expectError:   true,
			errorContains: "not found",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.setupImage != nil {
				tc.setupImage()
			}

			err := imageService.DeleteImage(ctx, tc.imageID)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// 验证已从数据库删除
				imageRepo := repository.NewImageRepository(repo.DB())
				_, err := imageRepo.GetByID(ctx, tc.imageID)
				assert.Error(t, err) // 应该查询不到
			}
		})
	}
}

func TestImageService_GetDefaultImageByName(t *testing.T) {
	t.Parallel()

	imageService, _, _, _ := setupTestImageService(t)

	testcases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "find ubuntu-jammy",
			input:    "ubuntu-jammy",
			expected: true,
		},
		{
			name:     "find ubuntu-focal",
			input:    "ubuntu-focal",
			expected: true,
		},
		{
			name:     "not found",
			input:    "nonexistent",
			expected: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			img := imageService.GetDefaultImageByName(tc.input)
			if tc.expected {
				assert.NotNil(t, img)
				assert.Equal(t, tc.input, img.Name)
			} else {
				assert.Nil(t, img)
			}
		})
	}
}

func TestImageService_ListDefaultImages(t *testing.T) {
	t.Parallel()

	imageService, _, _, _ := setupTestImageService(t)

	images := imageService.ListDefaultImages()
	assert.NotEmpty(t, images)
	assert.GreaterOrEqual(t, len(images), 3) // 至少包含 3 个默认镜像
}

func TestNewImageService(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.New(dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Close()
		_ = os.RemoveAll(tmpDir)
	})

	mockLibvirtClient := libvirt.NewMockClient()
	mockLibvirtClient.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
	mockLibvirtClient.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

	volumeRepo := repository.NewVolumeRepository(repo.DB())
	storageService, err := NewStorageService(mockLibvirtClient, volumeRepo)
	require.NoError(t, err)

	imageService, err := NewImageService(storageService, mockLibvirtClient, repo)
	assert.NoError(t, err)
	assert.NotNil(t, imageService)
}

func TestImageService_CreateImageFromInstance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		req           *entity.CreateImageFromInstanceRequest
		mockSetup     func(*libvirt.MockClient, *qemuimg.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful create from instance",
			req: &entity.CreateImageFromInstanceRequest{
				InstanceID:  "i-create-img-123",
				ImageName:   "Test Image",
				Description: "Created from instance",
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				m.On("GetDomainDisks", "i-create-img-123").Return([]libvirt.DomainDisk{
					{
						Type:   "file",
						Device: "disk",
						Source: libvirt.DomainDiskSource{File: "/var/lib/jvp/images/vol-123.qcow2"},
					},
				}, nil)
				m.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)
				// Mock Convert 调用
				q.On("Convert", mock.Anything, "qcow2", "qcow2", "/var/lib/jvp/images/vol-123.qcow2", mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
					// 在 Convert 调用后创建文件，模拟转换成功
					imagePath := args.Get(4).(string)
					dir := filepath.Dir(imagePath)
					_ = os.MkdirAll(dir, 0o755)
					_ = os.WriteFile(imagePath, []byte("test image"), 0o644)
				})
			},
			expectError: false,
		},
		{
			name: "instance not found",
			req: &entity.CreateImageFromInstanceRequest{
				InstanceID: "i-not-found",
				ImageName:  "Test Image",
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				m.On("GetDomainDisks", "i-not-found").Return(nil, fmt.Errorf("domain not found"))
			},
			expectError:   true,
			errorContains: "get instance disks",
		},
		{
			name: "instance has no disk",
			req: &entity.CreateImageFromInstanceRequest{
				InstanceID: "i-no-disk",
				ImageName:  "Test Image",
			},
			mockSetup: func(m *libvirt.MockClient, q *qemuimg.MockClient) {
				m.On("GetDomainDisks", "i-no-disk").Return([]libvirt.DomainDisk{}, nil)
			},
			expectError:   true,
			errorContains: "has no disk",
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

			// 创建独立的 mock client
			mockClientForTest := libvirt.NewMockClient()
			mockClientForTest.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
			mockClientForTest.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

			// 创建独立的 StorageService
			volumeRepo := repository.NewVolumeRepository(repo.DB())
			storageService, err := NewStorageService(mockClientForTest, volumeRepo)
			require.NoError(t, err)

			// 创建独立的 mock qemu-img client
			mockQemuImgClientForTest := qemuimg.NewMockClient()

			// 创建独立的 ImageService
			imageServiceForTest := NewImageServiceWithQemuImg(storageService, mockClientForTest, mockQemuImgClientForTest, repo)

			// 创建测试镜像目录
			imagesDir := filepath.Join(tmpDir, "images")
			err = os.MkdirAll(imagesDir, 0o755)
			require.NoError(t, err)
			imageServiceForTest.imagesPoolPath = imagesDir

			if tc.mockSetup != nil {
				tc.mockSetup(mockClientForTest, mockQemuImgClientForTest)
			}

			image, err := imageServiceForTest.CreateImageFromInstance(ctx, tc.req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, image)
			} else {
				assert.NoError(t, err)
				if image != nil {
					assert.NotEmpty(t, image.ID)
					assert.Equal(t, tc.req.ImageName, image.Name)
				}
			}
		})
	}
}

func TestImageService_DownloadImage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testcases := []struct {
		name          string
		url           string
		filename      string
		setupFile     func(string) string // 返回文件路径，如果文件已存在
		expectError   bool
		errorContains string
	}{
		{
			name:     "file already exists",
			url:      "http://example.com/image.qcow2",
			filename: "existing.qcow2",
			setupFile: func(imagesDir string) string {
				filePath := filepath.Join(imagesDir, "existing.qcow2")
				err := os.WriteFile(filePath, []byte("test"), 0o644)
				require.NoError(t, err)
				return filePath
			},
			expectError: false,
		},
		{
			name:     "invalid URL",
			url:      "invalid-url",
			filename: "test.qcow2",
			setupFile: func(string) string {
				return ""
			},
			expectError:   true,
			errorContains: "download",
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

			// 创建独立的 mock client
			mockClientForTest := libvirt.NewMockClient()
			mockClientForTest.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
			mockClientForTest.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

			// 创建独立的 StorageService
			volumeRepo := repository.NewVolumeRepository(repo.DB())
			storageService, err := NewStorageService(mockClientForTest, volumeRepo)
			require.NoError(t, err)

			// 创建独立的 mock qemu-img client
			mockQemuImgClientForTest := qemuimg.NewMockClient()

			// 创建独立的 ImageService
			imageServiceForTest := NewImageServiceWithQemuImg(storageService, mockClientForTest, mockQemuImgClientForTest, repo)

			// 创建测试镜像目录
			imagesDir := filepath.Join(tmpDir, "images")
			err = os.MkdirAll(imagesDir, 0o755)
			require.NoError(t, err)
			imageServiceForTest.imagesPoolPath = imagesDir

			if tc.setupFile != nil {
				tc.setupFile(imagesDir)
			}

			path, err := imageServiceForTest.DownloadImage(ctx, tc.url, tc.filename)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Empty(t, path)
			} else {
				// 对于已存在的文件，应该返回路径
				if tc.setupFile != nil {
					filePath := tc.setupFile(imagesDir)
					if filePath != "" {
						assert.NoError(t, err)
						assert.NotEmpty(t, path)
					}
				}
			}
		})
	}
}

func TestImageService_EnsureDefaultImages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// 为每个测试用例创建独立的数据库和 repository
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.New(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close()
		_ = os.RemoveAll(tmpDir)
	})

	// 创建独立的 mock client
	mockClientForTest := libvirt.NewMockClient()
	mockClientForTest.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
	mockClientForTest.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

	// 创建独立的 StorageService
	volumeRepo := repository.NewVolumeRepository(repo.DB())
	storageService, err := NewStorageService(mockClientForTest, volumeRepo)
	require.NoError(t, err)

	// 创建独立的 mock qemu-img client
	mockQemuImgClientForTest := qemuimg.NewMockClient()

	// 创建独立的 ImageService
	imageServiceForTest := NewImageServiceWithQemuImg(storageService, mockClientForTest, mockQemuImgClientForTest, repo)

	// 创建测试镜像目录
	imagesDir := filepath.Join(tmpDir, "images")
	err = os.MkdirAll(imagesDir, 0o755)
	require.NoError(t, err)
	imageServiceForTest.imagesPoolPath = imagesDir

	// 预先创建所有默认镜像文件，避免实际下载
	for _, defaultImg := range DefaultImages {
		imagePath := filepath.Join(imagesDir, defaultImg.Filename)
		f, err := os.Create(imagePath)
		require.NoError(t, err)
		_, err = f.Write(make([]byte, 100*1024*1024)) // 100MB，足够测试
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	// 测试 EnsureDefaultImages，应该检测到文件已存在，不会下载
	err = imageServiceForTest.EnsureDefaultImages(ctx)
	// 由于文件已存在，应该成功且不会尝试下载
	assert.NoError(t, err)
}
