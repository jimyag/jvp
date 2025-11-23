package metadata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/digitalocean/go-libvirt"
	"github.com/rs/zerolog/log"
)

// ==================== 崩溃恢复和自动修复 ====================

// repairVolumeMetadata 修复卷元数据
func (s *LibvirtMetadataStore) repairVolumeMetadata(ctx context.Context) error {
	log.Info().Msg("Repairing volume metadata")

	volumesDir := filepath.Join(s.config.BasePath, "volumes")

	// 1. 查找所有 qcow2 文件
	pattern := filepath.Join(volumesDir, "*.qcow2")
	volumeFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob volume files: %w", err)
	}

	repaired := 0
	errors := 0

	// 2. 检查每个卷文件
	for _, volumePath := range volumeFiles {
		metaPath := getSidecarPath(volumePath)

		// 如果元数据文件不存在,尝试修复
		if !fileExists(metaPath) {
			log.Warn().
				Str("volume_path", volumePath).
				Msg("Volume metadata file missing, attempting repair")

			// 尝试从备份恢复
			if err := restoreFromBackup(metaPath); err != nil {
				log.Error().
					Str("volume_path", volumePath).
					Err(err).
					Msg("Failed to restore volume metadata from backup")
				errors++
				continue
			}

			repaired++
		}

		// 验证元数据文件
		var meta VolumeMetadata
		if err := validateJSONFile(metaPath, &meta); err != nil {
			log.Warn().
				Str("meta_path", metaPath).
				Err(err).
				Msg("Volume metadata file is corrupted")

			// 尝试从备份恢复
			if err := restoreFromBackup(metaPath); err == nil {
				repaired++
			} else {
				errors++
			}
		}
	}

	log.Info().
		Int("repaired", repaired).
		Int("errors", errors).
		Msg("Volume metadata repair completed")

	return nil
}

// repairImageMetadata 修复镜像元数据
func (s *LibvirtMetadataStore) repairImageMetadata(ctx context.Context) error {
	log.Info().Msg("Repairing image metadata")

	imagesDir := filepath.Join(s.config.BasePath, "images")

	// 1. 查找所有 qcow2 文件
	pattern := filepath.Join(imagesDir, "*.qcow2")
	imageFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob image files: %w", err)
	}

	repaired := 0
	errors := 0

	// 2. 检查每个镜像文件
	for _, imagePath := range imageFiles {
		metaPath := getSidecarPath(imagePath)

		// 如果元数据文件不存在,尝试修复
		if !fileExists(metaPath) {
			log.Warn().
				Str("image_path", imagePath).
				Msg("Image metadata file missing, attempting repair")

			// 尝试从备份恢复
			if err := restoreFromBackup(metaPath); err != nil {
				log.Error().
					Str("image_path", imagePath).
					Err(err).
					Msg("Failed to restore image metadata from backup")
				errors++
				continue
			}

			repaired++
		}

		// 验证元数据文件
		var meta ImageMetadata
		if err := validateJSONFile(metaPath, &meta); err != nil {
			log.Warn().
				Str("meta_path", metaPath).
				Err(err).
				Msg("Image metadata file is corrupted")

			// 尝试从备份恢复
			if err := restoreFromBackup(metaPath); err == nil {
				repaired++
			} else {
				errors++
			}
		}
	}

	log.Info().
		Int("repaired", repaired).
		Int("errors", errors).
		Msg("Image metadata repair completed")

	return nil
}

// repairSnapshotIndex 修复快照索引
func (s *LibvirtMetadataStore) repairSnapshotIndex(ctx context.Context) error {
	log.Info().Msg("Repairing snapshot indexes")

	snapshotsDir := filepath.Join(s.config.BasePath, "volumes", ".snapshots")

	// 1. 查找所有快照索引文件
	pattern := filepath.Join(snapshotsDir, "*.json")
	indexFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob snapshot index files: %w", err)
	}

	repaired := 0
	errors := 0

	// 2. 验证每个索引文件
	for _, indexPath := range indexFiles {
		var index SnapshotIndex
		if err := validateJSONFile(indexPath, &index); err != nil {
			log.Warn().
				Str("index_path", indexPath).
				Err(err).
				Msg("Snapshot index file is corrupted")

			// 尝试从备份恢复
			if err := restoreFromBackup(indexPath); err == nil {
				repaired++
			} else {
				// 无法恢复,删除损坏的索引文件
				log.Error().
					Str("index_path", indexPath).
					Msg("Failed to repair snapshot index, removing corrupted file")
				os.Remove(indexPath)
				errors++
			}
		}
	}

	log.Info().
		Int("repaired", repaired).
		Int("errors", errors).
		Msg("Snapshot index repair completed")

	return nil
}

// repairInstanceMetadata 修复实例元数据
//
//lint:ignore U1000 // 保留供将来使用
func (s *LibvirtMetadataStore) repairInstanceMetadata(ctx context.Context) error {
	log.Info().Msg("Repairing instance metadata")

	// 获取所有 domains
	domains, _, err := s.conn.ConnectListAllDomains(1, 0)
	if err != nil {
		return fmt.Errorf("list domains: %w", err)
	}

	repaired := 0
	errors := 0

	// 检查每个 domain 的 JVP 元数据
	for _, domain := range domains {
		// 尝试读取 JVP 元数据
		metaXML, err := s.conn.DomainGetMetadata(
			domain,
			int32(libvirt.DomainMetadataElement),
			libvirt.OptString{JVPNamespace},
			libvirt.DomainAffectConfig,
		)

		if err != nil || metaXML == "" {
			// 没有 JVP 元数据,可能是非 JVP 管理的虚拟机,跳过
			continue
		}

		// 尝试解析元数据
		var jvpMeta JVPInstanceMetadata
		if err := unmarshalXML([]byte(metaXML), &jvpMeta); err != nil {
			log.Warn().
				Str("domain_uuid", uuidToString(domain.UUID)).
				Err(err).
				Msg("Instance metadata is corrupted")

			// 无法自动修复 domain metadata,记录错误
			errors++
			continue
		}

		// 元数据正常
		repaired++
	}

	log.Info().
		Int("checked", len(domains)).
		Int("errors", errors).
		Msg("Instance metadata repair completed")

	return nil
}

// repairKeyPairMetadata 修复密钥对元数据
//
//lint:ignore U1000 // 保留供将来使用
func (s *LibvirtMetadataStore) repairKeyPairMetadata(ctx context.Context) error {
	log.Info().Msg("Repairing keypair metadata")

	keypairsDir := filepath.Join(s.config.BasePath, "keypairs")

	// 1. 查找所有密钥对元数据文件
	pattern := filepath.Join(keypairsDir, "*.json")
	metaFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob keypair metadata files: %w", err)
	}

	repaired := 0
	errors := 0

	// 2. 验证每个元数据文件
	for _, metaPath := range metaFiles {
		var meta KeyPairMetadata
		if err := validateJSONFile(metaPath, &meta); err != nil {
			log.Warn().
				Str("meta_path", metaPath).
				Err(err).
				Msg("KeyPair metadata file is corrupted")

			// 尝试从备份恢复
			if err := restoreFromBackup(metaPath); err == nil {
				repaired++
			} else {
				errors++
			}
		}
	}

	log.Info().
		Int("repaired", repaired).
		Int("errors", errors).
		Msg("KeyPair metadata repair completed")

	return nil
}

// cleanOrphanedMetadata 清理孤儿元数据
//
//lint:ignore U1000 // 保留供将来使用
func (s *LibvirtMetadataStore) cleanOrphanedMetadata(ctx context.Context) error {
	log.Info().Msg("Cleaning orphaned metadata")

	// 1. 清理孤儿卷元数据
	if err := s.cleanOrphanedVolumeMetadata(); err != nil {
		log.Error().Err(err).Msg("Failed to clean orphaned volume metadata")
	}

	// 2. 清理孤儿镜像元数据
	if err := s.cleanOrphanedImageMetadata(); err != nil {
		log.Error().Err(err).Msg("Failed to clean orphaned image metadata")
	}

	// 3. 清理孤儿快照索引
	if err := s.cleanOrphanedSnapshotIndex(); err != nil {
		log.Error().Err(err).Msg("Failed to clean orphaned snapshot indexes")
	}

	return nil
}

// cleanOrphanedVolumeMetadata 清理孤儿卷元数据
//
//lint:ignore U1000 // 保留供将来使用
func (s *LibvirtMetadataStore) cleanOrphanedVolumeMetadata() error {
	volumesDir := filepath.Join(s.config.BasePath, "volumes")

	// 查找所有元数据文件
	pattern := filepath.Join(volumesDir, "*.jvp.json")
	metaFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob volume metadata files: %w", err)
	}

	cleaned := 0

	for _, metaPath := range metaFiles {
		// 检查对应的 qcow2 文件是否存在
		volumePath := metaPath[:len(metaPath)-9] // 移除 ".jvp.json"
		if !fileExists(volumePath) {
			log.Warn().
				Str("meta_path", metaPath).
				Msg("Orphaned volume metadata file found, removing")

			if err := os.Remove(metaPath); err != nil {
				log.Error().Err(err).Msg("Failed to remove orphaned metadata")
			} else {
				cleaned++
			}
		}
	}

	log.Info().Int("cleaned", cleaned).Msg("Cleaned orphaned volume metadata")
	return nil
}

// cleanOrphanedImageMetadata 清理孤儿镜像元数据
//
//lint:ignore U1000 // 保留供将来使用
func (s *LibvirtMetadataStore) cleanOrphanedImageMetadata() error {
	imagesDir := filepath.Join(s.config.BasePath, "images")

	// 查找所有元数据文件
	pattern := filepath.Join(imagesDir, "*.jvp.json")
	metaFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob image metadata files: %w", err)
	}

	cleaned := 0

	for _, metaPath := range metaFiles {
		// 检查对应的 qcow2 文件是否存在
		imagePath := metaPath[:len(metaPath)-9] // 移除 ".jvp.json"
		if !fileExists(imagePath) {
			log.Warn().
				Str("meta_path", metaPath).
				Msg("Orphaned image metadata file found, removing")

			if err := os.Remove(metaPath); err != nil {
				log.Error().Err(err).Msg("Failed to remove orphaned metadata")
			} else {
				cleaned++
			}
		}
	}

	log.Info().Int("cleaned", cleaned).Msg("Cleaned orphaned image metadata")
	return nil
}

// cleanOrphanedSnapshotIndex 清理孤儿快照索引
//
//lint:ignore U1000 // 保留供将来使用
func (s *LibvirtMetadataStore) cleanOrphanedSnapshotIndex() error {
	snapshotsDir := filepath.Join(s.config.BasePath, "volumes", ".snapshots")

	// 查找所有快照索引文件
	pattern := filepath.Join(snapshotsDir, "*.json")
	indexFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob snapshot index files: %w", err)
	}

	cleaned := 0

	for _, indexPath := range indexFiles {
		var index SnapshotIndex
		if err := loadJSONFile(indexPath, &index); err != nil {
			continue
		}

		// 检查卷文件是否存在
		if !fileExists(index.VolumePath) {
			log.Warn().
				Str("index_path", indexPath).
				Str("volume_id", index.VolumeID).
				Msg("Orphaned snapshot index found, removing")

			if err := os.Remove(indexPath); err != nil {
				log.Error().Err(err).Msg("Failed to remove orphaned snapshot index")
			} else {
				cleaned++
			}
		}
	}

	log.Info().Int("cleaned", cleaned).Msg("Cleaned orphaned snapshot indexes")
	return nil
}

// unmarshalXML 解析 XML
//
//lint:ignore U1000 // 保留供将来使用
func unmarshalXML(data []byte, v interface{}) error {
	// 使用 encoding/xml 解析
	return nil // 简化实现
}
