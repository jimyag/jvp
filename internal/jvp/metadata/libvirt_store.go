package metadata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/rs/zerolog/log"
)

// LibvirtMetadataStore 基于 Libvirt 的元数据存储实现
type LibvirtMetadataStore struct {
	config *StoreConfig
	conn   *libvirt.Libvirt

	// 内存索引
	index *MemoryIndex

	// 子管理器
	snapshotMgr *SnapshotManager
	keypairMgr  *KeyPairManager

	// 停止信号
	stopCh chan struct{}
}

// NewLibvirtMetadataStore 创建新的 LibvirtMetadataStore
func NewLibvirtMetadataStore(config *StoreConfig) (*LibvirtMetadataStore, error) {
	if config == nil {
		config = DefaultStoreConfig()
	}

	// 连接 libvirt
	// Note: go-libvirt requires a network connection, not a URI
	// For local connections, use unix socket: /var/run/libvirt/libvirt-sock
	// This is a simplified implementation - production code should handle URI parsing
	//lint:ignore SA1019 待实现:需要迁移到 NewWithDialer
	conn := libvirt.New(nil) // This needs to be properly connected via net.Conn
	// TODO: Implement proper libvirt connection via unix socket or TCP

	store := &LibvirtMetadataStore{
		config: config,
		conn:   conn,
		index:  NewMemoryIndex(),
		stopCh: make(chan struct{}),
	}

	// 初始化子管理器
	store.snapshotMgr = NewSnapshotManager(
		filepath.Join(config.BasePath, "volumes", ".snapshots"),
	)

	store.keypairMgr = NewKeyPairManager(
		filepath.Join(config.BasePath, "keypairs"),
	)

	return store, nil
}

// Initialize 初始化存储
func (s *LibvirtMetadataStore) Initialize(ctx context.Context) error {
	log.Info().Msg("Initializing LibvirtMetadataStore")

	// 1. 确保必要的目录存在
	dirs := []string{
		filepath.Join(s.config.BasePath, "volumes"),
		filepath.Join(s.config.BasePath, "volumes", ".snapshots"),
		filepath.Join(s.config.BasePath, "images"),
		filepath.Join(s.config.BasePath, "keypairs"),
		filepath.Join(s.config.BasePath, "locks"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// 2. 清理陈旧的锁文件
	s.cleanupStaleLocks()

	// 3. 验证和修复元数据
	if err := s.repairMetadata(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to repair metadata")
	}

	// 4. 构建内存索引
	if err := s.rebuildIndex(ctx); err != nil {
		return fmt.Errorf("rebuild index: %w", err)
	}

	// 5. 启动后台索引刷新
	if s.config.EnableIndexCache {
		go s.indexRefreshLoop()
	}

	log.Info().Msg("LibvirtMetadataStore initialized successfully")
	return nil
}

// Close 关闭存储
func (s *LibvirtMetadataStore) Close() error {
	close(s.stopCh)

	if s.conn != nil {
		return s.conn.Disconnect()
	}

	return nil
}

// cleanupStaleLocks 清理陈旧的锁文件
func (s *LibvirtMetadataStore) cleanupStaleLocks() {
	lockDir := filepath.Join(s.config.BasePath, "locks")
	lockFiles, _ := filepath.Glob(filepath.Join(lockDir, "*.lock"))

	for _, lockFile := range lockFiles {
		info, err := os.Stat(lockFile)
		if err != nil {
			continue
		}

		// 超过 5 分钟的锁文件认为是陈旧的
		if time.Since(info.ModTime()) > 5*time.Minute {
			log.Warn().Str("lock_file", lockFile).Msg("Removing stale lock file")
			os.Remove(lockFile)
		}
	}
}

// repairMetadata 修复元数据
func (s *LibvirtMetadataStore) repairMetadata(ctx context.Context) error {
	log.Info().Msg("Starting metadata repair")

	// 1. 修复卷元数据
	if err := s.repairVolumeMetadata(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to repair volume metadata")
	}

	// 2. 修复镜像元数据
	if err := s.repairImageMetadata(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to repair image metadata")
	}

	// 3. 修复快照索引
	if err := s.repairSnapshotIndex(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to repair snapshot index")
	}

	log.Info().Msg("Metadata repair completed")
	return nil
}

// indexRefreshLoop 后台索引刷新循环
func (s *LibvirtMetadataStore) indexRefreshLoop() {
	ticker := time.NewTicker(s.config.IndexRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := s.rebuildIndex(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to rebuild index")
			} else {
				log.Debug().Msg("Index rebuilt successfully")
			}

		case <-s.stopCh:
			return
		}
	}
}

// rebuildIndex 重建内存索引
func (s *LibvirtMetadataStore) rebuildIndex(ctx context.Context) error {
	log.Debug().Msg("Rebuilding memory index")

	newIndex := NewMemoryIndex()

	// 1. 索引实例
	if err := s.indexInstances(ctx, newIndex); err != nil {
		return fmt.Errorf("index instances: %w", err)
	}

	// 2. 索引卷
	if err := s.indexVolumes(ctx, newIndex); err != nil {
		return fmt.Errorf("index volumes: %w", err)
	}

	// 3. 索引镜像
	if err := s.indexImages(ctx, newIndex); err != nil {
		return fmt.Errorf("index images: %w", err)
	}

	// 4. 索引快照
	if err := s.indexSnapshots(ctx, newIndex); err != nil {
		return fmt.Errorf("index snapshots: %w", err)
	}

	// 5. 索引密钥对
	if err := s.indexKeyPairs(ctx, newIndex); err != nil {
		return fmt.Errorf("index keypairs: %w", err)
	}

	newIndex.LastSync = time.Now()

	// 6. 替换旧索引 (逐字段复制以避免复制mutex)
	s.index.Lock()
	s.index.Instances = newIndex.Instances
	s.index.InstancesByState = newIndex.InstancesByState
	s.index.InstancesByImage = newIndex.InstancesByImage
	s.index.InstancesByTag = newIndex.InstancesByTag
	s.index.Volumes = newIndex.Volumes
	s.index.VolumesByType = newIndex.VolumesByType
	s.index.VolumesByState = newIndex.VolumesByState
	s.index.VolumesByTag = newIndex.VolumesByTag
	s.index.Images = newIndex.Images
	s.index.ImagesByTag = newIndex.ImagesByTag
	s.index.Snapshots = newIndex.Snapshots
	s.index.SnapshotsByVolume = newIndex.SnapshotsByVolume
	s.index.KeyPairs = newIndex.KeyPairs
	s.index.LastSync = newIndex.LastSync
	s.index.Unlock()

	log.Debug().
		Int("instances", len(newIndex.Instances)).
		Int("volumes", len(newIndex.Volumes)).
		Int("images", len(newIndex.Images)).
		Int("snapshots", len(newIndex.Snapshots)).
		Int("keypairs", len(newIndex.KeyPairs)).
		Msg("Index rebuilt")

	return nil
}

// ==================== 辅助函数 ====================

// getDomainByInstanceID 根据实例 ID 获取 libvirt domain
func (s *LibvirtMetadataStore) getDomainByInstanceID(ctx context.Context, instanceID string) (libvirt.Domain, error) {
	s.index.RLock()
	idx, exists := s.index.Instances[instanceID]
	s.index.RUnlock()

	if !exists {
		return libvirt.Domain{}, fmt.Errorf("instance not found: %s", instanceID)
	}

	// Convert UUID string to libvirt.UUID
	uuid, err := stringToUUID(idx.DomainUUID)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("parse UUID: %w", err)
	}

	domain, err := s.conn.DomainLookupByUUID(uuid)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("lookup domain by UUID: %w", err)
	}

	return domain, nil
}

// getVolumePathByID 根据卷 ID 获取卷文件路径
func (s *LibvirtMetadataStore) getVolumePathByID(volumeID string) string {
	s.index.RLock()
	defer s.index.RUnlock()

	if idx, exists := s.index.Volumes[volumeID]; exists {
		return idx.Path
	}

	// 如果索引中没有,尝试直接查找文件
	volumesDir := filepath.Join(s.config.BasePath, "volumes")
	pattern := filepath.Join(volumesDir, "*.qcow2")
	files, _ := filepath.Glob(pattern)

	for _, file := range files {
		metaPath := file + ".jvp.json"
		if fileExists(metaPath) {
			var meta VolumeMetadata
			if loadJSONFile(metaPath, &meta) == nil && meta.ID == volumeID {
				return file
			}
		}
	}

	return ""
}

// getImagePathByID 根据镜像 ID 获取镜像文件路径
func (s *LibvirtMetadataStore) getImagePathByID(imageID string) string {
	s.index.RLock()
	defer s.index.RUnlock()

	if idx, exists := s.index.Images[imageID]; exists {
		return idx.Path
	}

	// 如果索引中没有,尝试直接查找文件
	imagesDir := filepath.Join(s.config.BasePath, "images")
	pattern := filepath.Join(imagesDir, "*.qcow2")
	files, _ := filepath.Glob(pattern)

	for _, file := range files {
		metaPath := file + ".jvp.json"
		if fileExists(metaPath) {
			var meta ImageMetadata
			if loadJSONFile(metaPath, &meta) == nil && meta.ID == imageID {
				return file
			}
		}
	}

	return ""
}

// uuidToString 将 libvirt UUID 转换为字符串
func uuidToString(uuid libvirt.UUID) string {
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		uuid[0], uuid[1], uuid[2], uuid[3],
		uuid[4], uuid[5],
		uuid[6], uuid[7],
		uuid[8], uuid[9],
		uuid[10], uuid[11], uuid[12], uuid[13], uuid[14], uuid[15])
}

// stringToUUID 将字符串转换为 libvirt UUID
func stringToUUID(s string) (libvirt.UUID, error) {
	var uuid libvirt.UUID
	// Remove dashes
	s = fmt.Sprintf("%s%s%s%s%s", s[0:8], s[9:13], s[14:18], s[19:23], s[24:36])

	// Parse hex string to bytes
	for i := 0; i < 16; i++ {
		_, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &uuid[i])
		if err != nil {
			return uuid, fmt.Errorf("invalid UUID format: %w", err)
		}
	}

	return uuid, nil
}
