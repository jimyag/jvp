package service

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestServices 包含测试所需的所有服务和依赖
type TestServices struct {
	Repo            *repository.Repository
	MockLibvirt     *libvirt.MockClient
	MockQemuImg     *qemuimg.MockClient
	StorageService  *StorageService
	ImageService    *ImageService
	InstanceService *InstanceService
	VolumeService   *VolumeService
	SnapshotService *SnapshotService
	KeyPairService  *KeyPairService
	TempDir         string
}

// setupTestServices 为每个测试用例创建独立的测试环境
// 每个测试用例都会获得自己的数据库、mock clients 和 service 实例
func setupTestServices(t *testing.T) *TestServices {
	t.Helper()

	// 创建临时目录和数据库（每个测试用例都有独立的数据库文件）
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.New(dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Close()
		_ = os.RemoveAll(tmpDir)
	})

	// 创建 mock libvirt client
	mockLibvirt := libvirt.NewMockClient()
	mockLibvirt.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
	mockLibvirt.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

	// 创建 StorageService
	storageService, err := NewStorageService(mockLibvirt)
	require.NoError(t, err)

	// 创建 mock qemu-img client
	mockQemuImg := qemuimg.NewMockClient()

	// 替换 StorageService 中的真实 qemu-img client 为 mock
	storageService.qemuImgClient = mockQemuImg

	// 创建 ImageService
	imageService := &ImageService{
		storageService: storageService,
		libvirtClient:  mockLibvirt,
		qemuImgClient:  mockQemuImg,
		idGen:          idgen.New(),
		httpClient:     &http.Client{Timeout: 30 * time.Minute},
		imageRepo:      repository.NewImageRepository(repo.DB()),
		imagesPoolName: "images",
		imagesPoolPath: "/var/lib/jvp/images/images",
	}
	// 设置测试镜像路径
	imagesDir := filepath.Join(tmpDir, "images")
	err = os.MkdirAll(imagesDir, 0o755)
	require.NoError(t, err)
	imageService.imagesPoolPath = imagesDir

	// 创建 virt-customize 客户端（mock，测试中会替换）
	virtCustomizeClient, _ := virtcustomize.NewClient()
	if virtCustomizeClient == nil {
		// 如果 virt-customize 不存在，使用 mock path
		virtCustomizeClient = virtcustomize.NewClientWithPath("/usr/bin/virt-customize")
	}

	// 创建 InstanceService
	instanceService := &InstanceService{
		storageService:      storageService,
		imageService:        imageService,
		libvirtClient:       mockLibvirt,
		virtCustomizeClient: virtCustomizeClient,
		idGen:               idgen.New(),
		instanceRepo:        repository.NewInstanceRepository(repo.DB()),
	}

	// 创建 VolumeService
	volumeService := &VolumeService{
		storageService:  storageService,
		instanceService: instanceService,
		libvirtClient:   mockLibvirt,
		qemuImgClient:   mockQemuImg,
		idGen:           idgen.New(),
		volumeRepo:      repository.NewVolumeRepository(repo.DB()),
		snapshotRepo:    repository.NewSnapshotRepository(repo.DB()),
	}

	// 创建 SnapshotService
	snapshotService := &SnapshotService{
		storageService: storageService,
		libvirtClient:  mockLibvirt,
		qemuImgClient:  mockQemuImg,
		idGen:          idgen.New(),
		snapshotRepo:   repository.NewSnapshotRepository(repo.DB()),
	}

	// 创建 KeyPairService
	keyPairService := NewKeyPairService(repo)

	// 更新 InstanceService 以包含 KeyPairService
	instanceService.keyPairService = keyPairService

	return &TestServices{
		Repo:            repo,
		MockLibvirt:     mockLibvirt,
		MockQemuImg:     mockQemuImg,
		StorageService:  storageService,
		ImageService:    imageService,
		InstanceService: instanceService,
		VolumeService:   volumeService,
		SnapshotService: snapshotService,
		KeyPairService:  keyPairService,
		TempDir:         tmpDir,
	}
}
