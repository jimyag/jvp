package metadata

import (
	"context"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
)

// ==================== 基础接口 ====================

// Store 定义元数据存储的基础接口
type Store interface {
	// Initialize 初始化存储,包括目录创建、索引构建、数据修复等
	Initialize(ctx context.Context) error

	// Close 关闭存储,释放资源
	Close() error
}

// ==================== 实例存储接口 ====================

// InstanceStore 实例元数据存储接口
type InstanceStore interface {
	// SaveInstance 保存实例元数据
	SaveInstance(ctx context.Context, instance *entity.Instance) error

	// GetInstance 获取单个实例
	GetInstance(ctx context.Context, instanceID string) (*entity.Instance, error)

	// ListInstances 列出所有实例
	ListInstances(ctx context.Context) ([]*entity.Instance, error)

	// DescribeInstances 查询实例(支持过滤)
	DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]*entity.Instance, error)

	// DeleteInstance 删除实例元数据
	DeleteInstance(ctx context.Context, instanceID string) error

	// UpdateInstanceState 更新实例状态
	UpdateInstanceState(ctx context.Context, instanceID string, state string) error
}

// ==================== 卷存储接口 ====================

// VolumeStore 卷元数据存储接口
type VolumeStore interface {
	// SaveVolume 保存卷元数据
	SaveVolume(ctx context.Context, volume *entity.EBSVolume) error

	// GetVolume 获取单个卷
	GetVolume(ctx context.Context, volumeID string) (*entity.EBSVolume, error)

	// ListVolumes 列出所有卷
	ListVolumes(ctx context.Context) ([]*entity.EBSVolume, error)

	// DescribeVolumes 查询卷(支持过滤)
	DescribeVolumes(ctx context.Context, req *entity.DescribeVolumesRequest) ([]*entity.EBSVolume, error)

	// DeleteVolume 删除卷元数据
	DeleteVolume(ctx context.Context, volumeID string) error

	// UpdateVolumeState 更新卷状态
	UpdateVolumeState(ctx context.Context, volumeID string, state string) error

	// GetVolumeAttachments 获取卷的附加关系
	GetVolumeAttachments(ctx context.Context, volumeID string) ([]*entity.VolumeAttachment, error)
}

// ==================== 镜像存储接口 ====================

// ImageStore 镜像元数据存储接口
type ImageStore interface {
	// SaveImage 保存镜像元数据
	SaveImage(ctx context.Context, image *entity.Image) error

	// GetImage 获取单个镜像
	GetImage(ctx context.Context, imageID string) (*entity.Image, error)

	// ListImages 列出所有镜像
	ListImages(ctx context.Context) ([]*entity.Image, error)

	// DescribeImages 查询镜像(支持过滤)
	DescribeImages(ctx context.Context, req *entity.DescribeImagesRequest) ([]*entity.Image, error)

	// DeleteImage 删除镜像元数据
	DeleteImage(ctx context.Context, imageID string) error
}

// ==================== 快照存储接口 ====================

// SnapshotStore 快照元数据存储接口
type SnapshotStore interface {
	// SaveSnapshot 保存快照元数据
	SaveSnapshot(ctx context.Context, snapshot *entity.EBSSnapshot) error

	// GetSnapshot 获取单个快照
	GetSnapshot(ctx context.Context, snapshotID string) (*entity.EBSSnapshot, error)

	// ListSnapshots 列出卷的所有快照
	ListSnapshots(ctx context.Context, volumeID string) ([]*entity.EBSSnapshot, error)

	// DescribeSnapshots 查询快照(支持过滤)
	DescribeSnapshots(ctx context.Context, req *entity.DescribeSnapshotsRequest) ([]*entity.EBSSnapshot, error)

	// DeleteSnapshot 删除快照元数据
	DeleteSnapshot(ctx context.Context, snapshotID string) error
}

// ==================== 密钥对存储接口 ====================

// KeyPairStore 密钥对元数据存储接口
type KeyPairStore interface {
	// SaveKeyPair 保存密钥对元数据
	SaveKeyPair(ctx context.Context, keyPair *entity.KeyPair) error

	// GetKeyPair 获取单个密钥对
	GetKeyPair(ctx context.Context, keyPairID string) (*entity.KeyPair, error)

	// ListKeyPairs 列出所有密钥对
	ListKeyPairs(ctx context.Context) ([]*entity.KeyPair, error)

	// DescribeKeyPairs 查询密钥对(支持过滤)
	DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]*entity.KeyPair, error)

	// DeleteKeyPair 删除密钥对元数据
	DeleteKeyPair(ctx context.Context, keyPairID string) error

	// GetKeyPairPublicKey 获取密钥对的公钥(用于注入实例)
	GetKeyPairPublicKey(ctx context.Context, keyPairID string) (string, error)
}

// ==================== 组合接口 ====================

// MetadataStore 元数据存储的完整接口 组合所有子接口
type MetadataStore interface {
	Store
	InstanceStore
	VolumeStore
	ImageStore
	SnapshotStore
	KeyPairStore
}

// ==================== 配置 ====================

// StoreConfig 存储配置
type StoreConfig struct {
	// BasePath 基础路径
	BasePath string

	// LibvirtURI Libvirt 连接 URI
	LibvirtURI string

	// EnableIndexCache 是否启用索引缓存
	EnableIndexCache bool

	// IndexRefreshInterval 索引刷新间隔
	IndexRefreshInterval time.Duration

	// LockTimeout 锁超时时间
	LockTimeout time.Duration
}

// DefaultStoreConfig 返回默认配置
func DefaultStoreConfig() *StoreConfig {
	return &StoreConfig{
		BasePath:             "/var/lib/jvp",
		LibvirtURI:           "qemu:///system",
		EnableIndexCache:     true,
		IndexRefreshInterval: 5 * time.Minute,
		LockTimeout:          30 * time.Second,
	}
}
