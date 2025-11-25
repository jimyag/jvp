package service

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestServices 包含测试所需的所有服务和依赖
type TestServices struct {
	MockLibvirt     *libvirt.MockClient
	MockQemuImg     *qemuimg.MockClient
	StorageService  *StorageService
	ImageService    *ImageService
	InstanceService *InstanceService
	VolumeService   *VolumeService
	KeyPairService  *KeyPairService
	TempDir         string
}

// setupTestServices 为每个测试用例创建独立的测试环境
// 每个测试用例都会获得自己的 mock clients 和 service 实例
func setupTestServices(t *testing.T) *TestServices {
	t.Helper()

	// 创建临时目录
	tmpDir := t.TempDir()

	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	// 创建 mock libvirt client
	mockLibvirt := libvirt.NewMockClient()
	mockLibvirt.On("EnsureStoragePool", "default", "dir", mock.AnythingOfType("string")).Return(nil)
	mockLibvirt.On("EnsureStoragePool", "images", "dir", mock.AnythingOfType("string")).Return(nil)

	// 创建 StorageService，使用临时目录
	storageService, err := NewStorageService(mockLibvirt, tmpDir)
	require.NoError(t, err)

	// 创建 mock qemu-img client
	mockQemuImg := qemuimg.NewMockClient()

	// 替换 StorageService 中的真实 qemu-img client 为 mock
	storageService.qemuImgClient = mockQemuImg

	// 创建 ImageService（不再使用 metadataStore）
	imageService := &ImageService{
		storageService: storageService,
		libvirtClient:  mockLibvirt,
		qemuImgClient:  mockQemuImg,
		idGen:          idgen.New(),
		httpClient:     &http.Client{Timeout: 30 * time.Minute},
		imagesPoolName: storageService.imagesPoolName,
		imagesPoolPath: storageService.imagesPoolPath,
	}
	// 确保测试镜像目录存在
	err = os.MkdirAll(imageService.imagesPoolPath, 0o755)
	require.NoError(t, err)

	// 创建 virt-customize 客户端（mock，测试中会替换）
	virtCustomizeClient, _ := virtcustomize.NewClient()
	if virtCustomizeClient == nil {
		// 如果 virt-customize 不存在，使用 mock path
		virtCustomizeClient = virtcustomize.NewClientWithPath("/usr/bin/virt-customize")
	}

	// 创建 KeyPairService（使用文件存储）
	keypairDir := filepath.Join(tmpDir, "keypairs")
	err = os.MkdirAll(keypairDir, 0700)
	require.NoError(t, err)
	keyPairService := &KeyPairService{
		idGen:      idgen.New(),
		storageDir: keypairDir,
	}

	// 创建 InstanceService（不再使用 metadataStore）
	instanceService := &InstanceService{
		storageService:      storageService,
		imageService:        imageService,
		keyPairService:      keyPairService,
		libvirtClient:       mockLibvirt,
		virtCustomizeClient: virtCustomizeClient,
		idGen:               idgen.New(),
	}

	// 创建 NodeStorage (测试用)
	nodeStorage, _ := NewNodeStorage("/tmp/jvp-test")
	nodeService, _ := NewNodeService(nodeStorage)
	storagePoolService := NewStoragePoolService(nodeStorage)

	// 创建 VolumeService（使用新的依赖）
	volumeService := NewVolumeService(nodeService, storagePoolService)
	volumeService.qemuImgClient = mockQemuImg  // 注入 mock 客户端

	return &TestServices{
		MockLibvirt:     mockLibvirt,
		MockQemuImg:     mockQemuImg,
		StorageService:  storageService,
		ImageService:    imageService,
		InstanceService: instanceService,
		VolumeService:   volumeService,
		KeyPairService:  keyPairService,
		TempDir:         tmpDir,
	}
}
