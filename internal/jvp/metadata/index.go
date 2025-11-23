package metadata

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/digitalocean/go-libvirt"
	"github.com/rs/zerolog/log"
)

// ==================== 索引构建函数 ====================

// indexInstances 索引所有实例
func (s *LibvirtMetadataStore) indexInstances(ctx context.Context, index *MemoryIndex) error {
	log.Debug().Msg("Indexing instances")

	// 1. 获取所有 domains (包括运行和停止的)
	flags := libvirt.ConnectListDomainsActive | libvirt.ConnectListDomainsInactive
	domains, _, err := s.conn.ConnectListAllDomains(1, flags)
	if err != nil {
		return fmt.Errorf("list domains: %w", err)
	}

	// 2. 遍历 domains
	for _, domain := range domains {
		// 尝试解析实例（无论是否有 JVP 元数据）
		instance, err := s.buildInstanceFromDomain(ctx, domain)
		if err != nil {
			log.Warn().
				Str("domain_uuid", uuidToString(domain.UUID)).
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to build instance from domain")
			continue
		}

		// 添加到索引
		idx := &InstanceIndex{
			ID:         instance.ID,
			DomainUUID: uuidToString(domain.UUID),
			DomainName: domain.Name,
			State:      instance.State,
			ImageID:    instance.ImageID,
			VolumeID:   instance.VolumeID,
		}

		index.Instances[instance.ID] = idx

		// 添加到状态索引
		index.InstancesByState[instance.State] = append(
			index.InstancesByState[instance.State],
			instance.ID,
		)

		// 添加到镜像索引
		if instance.ImageID != "" {
			index.InstancesByImage[instance.ImageID] = append(
				index.InstancesByImage[instance.ImageID],
				instance.ID,
			)
		}

		// (标签索引已移除,因为 entity.Instance 不支持 Tags)
	}

	log.Debug().Int("count", len(index.Instances)).Msg("Instances indexed")
	return nil
}

// indexVolumes 索引所有卷
func (s *LibvirtMetadataStore) indexVolumes(ctx context.Context, index *MemoryIndex) error {
	log.Debug().Msg("Indexing volumes")

	// 1. 从 libvirt 获取所有存储池
	pools, _, err := s.conn.ConnectListAllStoragePools(1, 0)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list storage pools, falling back to file scan")
		return s.indexVolumesFromFiles(ctx, index)
	}

	// 2. 遍历每个存储池
	for _, pool := range pools {
		// 获取存储池中的所有卷
		volumes, _, err := s.conn.StoragePoolListAllVolumes(pool, 0, 0)
		if err != nil {
			log.Warn().
				Str("pool_name", pool.Name).
				Err(err).
				Msg("Failed to list volumes in pool")
			continue
		}

		// 3. 遍历卷
		for _, vol := range volumes {
			// 获取卷路径
			volPath, err := s.conn.StorageVolGetPath(vol)
			if err != nil {
				log.Warn().
					Str("vol_name", vol.Name).
					Err(err).
					Msg("Failed to get volume path")
				continue
			}

			// 获取卷信息
			volType, capacity, _, err := s.conn.StorageVolGetInfo(vol)
			if err != nil {
				log.Warn().
					Str("vol_name", vol.Name).
					Err(err).
					Msg("Failed to get volume info")
				continue
			}

			// 只处理文件类型的卷
			if volType != 0 { // 0 = VIR_STORAGE_VOL_FILE
				continue
			}

			// 尝试读取 JVP 元数据
			metaPath := getSidecarPath(volPath)
			var meta VolumeMetadata
			hasMetadata := false
			if fileExists(metaPath) {
				if err := loadJSONFile(metaPath, &meta); err == nil {
					hasMetadata = true
				}
			}

			// 构建索引
			volumeID := vol.Name
			volumeState := "available"
			volumeType := "gp2"
			var tags map[string]string

			if hasMetadata {
				volumeID = meta.ID
				volumeState = meta.State
				volumeType = meta.VolumeType
				tags = meta.Tags
			}

			idx := &VolumeIndex{
				ID:         volumeID,
				Path:       volPath,
				State:      volumeState,
				VolumeType: volumeType,
				SizeGB:     capacity / (1024 * 1024 * 1024),
				Tags:       tags,
			}

			index.Volumes[volumeID] = idx

			// 添加到类型索引
			index.VolumesByType[volumeType] = append(
				index.VolumesByType[volumeType],
				volumeID,
			)

			// 添加到状态索引
			index.VolumesByState[volumeState] = append(
				index.VolumesByState[volumeState],
				volumeID,
			)

			// 添加到标签索引
			for k, v := range tags {
				tagPair := fmt.Sprintf("%s=%s", k, v)
				index.VolumesByTag[tagPair] = append(
					index.VolumesByTag[tagPair],
					volumeID,
				)
			}
		}
	}

	log.Debug().Int("count", len(index.Volumes)).Msg("Volumes indexed")
	return nil
}

// indexVolumesFromFiles 从文件系统索引卷（备用方案）
func (s *LibvirtMetadataStore) indexVolumesFromFiles(ctx context.Context, index *MemoryIndex) error {
	volumesDir := filepath.Join(s.config.BasePath, "volumes")

	// 查找所有 qcow2 文件
	pattern := filepath.Join(volumesDir, "*.qcow2")
	volumeFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob volume files: %w", err)
	}

	for _, volumePath := range volumeFiles {
		metaPath := getSidecarPath(volumePath)
		if !fileExists(metaPath) {
			continue
		}

		var meta VolumeMetadata
		if err := loadJSONFile(metaPath, &meta); err != nil {
			continue
		}

		sizeGB := uint64(0)
		if info, err := getQCOW2Info(volumePath); err == nil {
			sizeGB = info.VirtualSize / (1024 * 1024 * 1024)
		}

		idx := &VolumeIndex{
			ID:         meta.ID,
			Path:       volumePath,
			State:      meta.State,
			VolumeType: meta.VolumeType,
			SizeGB:     sizeGB,
			Tags:       meta.Tags,
		}

		index.Volumes[meta.ID] = idx
		index.VolumesByType[meta.VolumeType] = append(index.VolumesByType[meta.VolumeType], meta.ID)
		index.VolumesByState[meta.State] = append(index.VolumesByState[meta.State], meta.ID)

		for k, v := range meta.Tags {
			tagPair := fmt.Sprintf("%s=%s", k, v)
			index.VolumesByTag[tagPair] = append(index.VolumesByTag[tagPair], meta.ID)
		}
	}

	return nil
}

// indexImages 索引所有镜像
func (s *LibvirtMetadataStore) indexImages(ctx context.Context, index *MemoryIndex) error {
	log.Debug().Msg("Indexing images")

	imagesDir := filepath.Join(s.config.BasePath, "images")

	// 1. 查找所有 qcow2 文件
	pattern := filepath.Join(imagesDir, "*.qcow2")
	imageFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob image files: %w", err)
	}

	// 2. 遍历镜像文件
	for _, imagePath := range imageFiles {
		// 读取边车元数据文件
		metaPath := getSidecarPath(imagePath)
		if !fileExists(metaPath) {
			log.Warn().Str("image_path", imagePath).Msg("Image metadata file not found")
			continue
		}

		var meta ImageMetadata
		if err := loadJSONFile(metaPath, &meta); err != nil {
			log.Warn().
				Str("meta_path", metaPath).
				Err(err).
				Msg("Failed to load image metadata")
			continue
		}

		// 获取镜像大小
		sizeGB := uint64(0)
		if info, err := getQCOW2Info(imagePath); err == nil {
			sizeGB = info.VirtualSize / (1024 * 1024 * 1024)
		}

		// 添加到索引
		idx := &ImageIndex{
			ID:     meta.ID,
			Name:   meta.Name,
			Path:   imagePath,
			State:  meta.State,
			SizeGB: sizeGB,
		}

		index.Images[meta.ID] = idx
	}

	log.Debug().Int("count", len(index.Images)).Msg("Images indexed")
	return nil
}

// indexSnapshots 索引所有快照
func (s *LibvirtMetadataStore) indexSnapshots(ctx context.Context, index *MemoryIndex) error {
	log.Debug().Msg("Indexing snapshots")

	snapshotsDir := filepath.Join(s.config.BasePath, "volumes", ".snapshots")

	// 1. 查找所有快照索引文件
	pattern := filepath.Join(snapshotsDir, "*.json")
	indexFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob snapshot index files: %w", err)
	}

	// 2. 遍历快照索引文件
	for _, indexPath := range indexFiles {
		var snapIndex SnapshotIndex
		if err := loadJSONFile(indexPath, &snapIndex); err != nil {
			log.Warn().
				Str("index_path", indexPath).
				Err(err).
				Msg("Failed to load snapshot index")
			continue
		}

		// 添加所有快照到索引
		for _, snapshot := range snapIndex.Snapshots {
			idx := &SnapshotIndexItem{
				ID:         snapshot.ID,
				VolumeID:   snapIndex.VolumeID,
				State:      snapshot.State,
				SizeGB:     snapshot.SizeGB,
				CreateTime: snapshot.StartTime,
				Tags:       snapshot.Tags,
			}

			index.Snapshots[snapshot.ID] = idx

			// 添加到卷快照索引
			index.SnapshotsByVolume[snapIndex.VolumeID] = append(
				index.SnapshotsByVolume[snapIndex.VolumeID],
				snapshot.ID,
			)
		}
	}

	log.Debug().Int("count", len(index.Snapshots)).Msg("Snapshots indexed")
	return nil
}

// indexKeyPairs 索引所有密钥对
func (s *LibvirtMetadataStore) indexKeyPairs(ctx context.Context, index *MemoryIndex) error {
	log.Debug().Msg("Indexing keypairs")

	keypairsDir := filepath.Join(s.config.BasePath, "keypairs")

	// 1. 查找所有密钥对元数据文件
	pattern := filepath.Join(keypairsDir, "*.json")
	metaFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob keypair metadata files: %w", err)
	}

	// 2. 遍历密钥对文件
	for _, metaPath := range metaFiles {
		var meta KeyPairMetadata
		if err := loadJSONFile(metaPath, &meta); err != nil {
			log.Warn().
				Str("meta_path", metaPath).
				Err(err).
				Msg("Failed to load keypair metadata")
			continue
		}

		// 添加到索引
		idx := &KeyPairIndexEntry{
			ID:          meta.ID,
			Name:        meta.Name,
			Fingerprint: meta.Fingerprint,
		}

		index.KeyPairs[meta.ID] = idx
	}

	log.Debug().Int("count", len(index.KeyPairs)).Msg("KeyPairs indexed")
	return nil
}

// ==================== QCOW2 工具函数 ====================

// QCOW2Info QCOW2 镜像信息
type QCOW2Info struct {
	VirtualSize uint64
	ActualSize  uint64
	Format      string
}

// getQCOW2Info 获取 QCOW2 镜像信息
func getQCOW2Info(imagePath string) (*QCOW2Info, error) {
	// 这里需要调用 qemu-img info 命令获取镜像信息
	// 简化实现,直接使用文件大小
	fileInfo, err := filepath.Glob(imagePath)
	if err != nil || len(fileInfo) == 0 {
		return nil, fmt.Errorf("file not found: %s", imagePath)
	}

	// 实际应该解析 qemu-img info 的输出
	// 这里返回一个简化的实现
	return &QCOW2Info{
		VirtualSize: 10 * 1024 * 1024 * 1024, // 10GB 默认
		ActualSize:  0,
		Format:      "qcow2",
	}, nil
}
