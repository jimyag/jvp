package metadata

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/rs/zerolog/log"
)

// ==================== Volume 元数据管理 ====================

// SaveVolume 保存卷元数据
func (s *LibvirtMetadataStore) SaveVolume(ctx context.Context, volume *entity.EBSVolume) error {
	log.Debug().
		Str("volume_id", volume.VolumeID).
		Str("state", volume.State).
		Uint64("size_gb", volume.SizeGB).
		Msg("Saving volume metadata")

	// 1. 转换 Tags
	tags := make(map[string]string)
	for _, tag := range volume.Tags {
		tags[tag.Key] = tag.Value
	}

	// 2. 解析 CreateTime
	createTime, _ := time.Parse(time.RFC3339, volume.CreateTime)

	// 3. 构建卷元数据
	meta := VolumeMetadata{
		Version:       "1.0",
		SchemaVersion: "1.0",
		ResourceType:  "volume",
		ID:            volume.VolumeID,
		SnapshotID:    volume.SnapshotID,
		VolumeType:    volume.VolumeType,
		Iops:          volume.Iops,
		Encrypted:     volume.Encrypted,
		KmsKeyID:      volume.KmsKeyID,
		State:         volume.State,
		CreateTime:    createTime,
		UpdateTime:    time.Now(),
		Tags:          tags,
	}

	// 2. 获取卷文件路径
	volumePath := s.getVolumePathByID(volume.VolumeID)
	if volumePath == "" {
		// 如果卷文件不存在,使用默认路径
		volumePath = filepath.Join(s.config.BasePath, "volumes", volume.VolumeID+".qcow2")
	}

	// 3. 保存边车元数据文件
	metaPath := getSidecarPath(volumePath)
	if err := saveJSONFile(metaPath, meta); err != nil {
		return fmt.Errorf("save volume metadata: %w", err)
	}

	// 4. 更新内存索引
	s.updateVolumeIndex(volume, volumePath)

	log.Info().
		Str("volume_id", volume.VolumeID).
		Str("meta_path", metaPath).
		Msg("Volume metadata saved successfully")

	return nil
}

// GetVolume 获取单个卷
func (s *LibvirtMetadataStore) GetVolume(ctx context.Context, volumeID string) (*entity.EBSVolume, error) {
	log.Debug().Str("volume_id", volumeID).Msg("Getting volume")

	// 1. 从索引获取卷路径
	volumePath := s.getVolumePathByID(volumeID)
	if volumePath == "" {
		return nil, fmt.Errorf("volume not found: %s", volumeID)
	}

	// 2. 读取元数据文件
	metaPath := getSidecarPath(volumePath)
	var meta VolumeMetadata
	if err := loadJSONFile(metaPath, &meta); err != nil {
		return nil, fmt.Errorf("load volume metadata: %w", err)
	}

	// 3. 获取卷大小
	sizeGB := uint64(0)
	if info, err := getQCOW2Info(volumePath); err == nil {
		sizeGB = info.VirtualSize / (1024 * 1024 * 1024)
	}

	// 4. 转换 Tags
	tags := make([]entity.Tag, 0, len(meta.Tags))
	for k, v := range meta.Tags {
		tags = append(tags, entity.Tag{Key: k, Value: v})
	}

	// 5. 构建卷对象
	volume := &entity.EBSVolume{
		VolumeID:    meta.ID,
		SnapshotID:  meta.SnapshotID,
		VolumeType:  meta.VolumeType,
		SizeGB:      sizeGB,
		Iops:        meta.Iops,
		Encrypted:   meta.Encrypted,
		KmsKeyID:    meta.KmsKeyID,
		State:       meta.State,
		CreateTime:  meta.CreateTime.Format(time.RFC3339),
		Tags:        tags,
		Attachments: []entity.VolumeAttachment{}, // 需要从 domain 中读取
	}

	// 6. 获取附加关系
	attachmentPtrs, err := s.GetVolumeAttachments(ctx, volumeID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get volume attachments")
	} else {
		// 转换指针数组为值数组
		attachments := make([]entity.VolumeAttachment, 0, len(attachmentPtrs))
		for _, a := range attachmentPtrs {
			if a != nil {
				attachments = append(attachments, *a)
			}
		}
		volume.Attachments = attachments
	}

	return volume, nil
}

// ListVolumes 列出所有卷
func (s *LibvirtMetadataStore) ListVolumes(ctx context.Context) ([]*entity.EBSVolume, error) {
	log.Debug().Msg("Listing all volumes")

	s.index.RLock()
	volumeIDs := make([]string, 0, len(s.index.Volumes))
	for id := range s.index.Volumes {
		volumeIDs = append(volumeIDs, id)
	}
	s.index.RUnlock()

	volumes := make([]*entity.EBSVolume, 0, len(volumeIDs))
	for _, id := range volumeIDs {
		volume, err := s.GetVolume(ctx, id)
		if err != nil {
			log.Warn().Str("volume_id", id).Err(err).Msg("Failed to get volume")
			continue
		}
		volumes = append(volumes, volume)
	}

	log.Debug().Int("count", len(volumes)).Msg("Listed volumes")
	return volumes, nil
}

// DescribeVolumes 查询卷(支持过滤)
func (s *LibvirtMetadataStore) DescribeVolumes(ctx context.Context, req *entity.DescribeVolumesRequest) ([]*entity.EBSVolume, error) {
	log.Debug().
		Strs("volume_ids", req.VolumeIDs).
		Interface("filters", req.Filters).
		Msg("Describing volumes")

	// 如果指定了卷 ID,直接查询
	if len(req.VolumeIDs) > 0 {
		return s.getVolumesByIDs(ctx, req.VolumeIDs)
	}

	// 否则,使用索引进行过滤查询
	candidateIDs := s.filterVolumesByIndex(req)

	// 获取卷详情
	volumes := make([]*entity.EBSVolume, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		volume, err := s.GetVolume(ctx, id)
		if err != nil {
			log.Warn().Str("volume_id", id).Err(err).Msg("Failed to get volume")
			continue
		}

		// 应用额外的过滤条件
		if s.matchesVolumeFilters(volume, req.Filters) {
			volumes = append(volumes, volume)
		}
	}

	log.Debug().Int("count", len(volumes)).Msg("Described volumes")
	return volumes, nil
}

// DeleteVolume 删除卷元数据
func (s *LibvirtMetadataStore) DeleteVolume(ctx context.Context, volumeID string) error {
	log.Debug().Str("volume_id", volumeID).Msg("Deleting volume metadata")

	// 1. 获取卷路径
	volumePath := s.getVolumePathByID(volumeID)
	if volumePath == "" {
		// 卷不存在,认为已删除
		return nil
	}

	// 2. 删除边车元数据文件
	metaPath := getSidecarPath(volumePath)
	if err := deleteJSONFile(metaPath); err != nil {
		log.Warn().Err(err).Msg("Failed to delete volume metadata file")
	}

	// 3. 从索引中删除
	s.removeVolumeFromIndex(volumeID)

	log.Info().Str("volume_id", volumeID).Msg("Volume metadata deleted")
	return nil
}

// UpdateVolumeState 更新卷状态
func (s *LibvirtMetadataStore) UpdateVolumeState(ctx context.Context, volumeID string, state string) error {
	log.Debug().
		Str("volume_id", volumeID).
		Str("state", state).
		Msg("Updating volume state")

	// 1. 获取卷
	volume, err := s.GetVolume(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("get volume: %w", err)
	}

	// 2. 更新状态
	volume.State = state

	// 3. 保存
	return s.SaveVolume(ctx, volume)
}

// GetVolumeAttachments 获取卷的附加关系
func (s *LibvirtMetadataStore) GetVolumeAttachments(ctx context.Context, volumeID string) ([]*entity.VolumeAttachment, error) {
	log.Debug().Str("volume_id", volumeID).Msg("Getting volume attachments")

	attachments := make([]*entity.VolumeAttachment, 0)

	// 遍历所有实例,查找使用该卷的实例
	s.index.RLock()
	instanceIDs := make([]string, 0, len(s.index.Instances))
	for id, idx := range s.index.Instances {
		if idx.VolumeID == volumeID {
			instanceIDs = append(instanceIDs, id)
		}
	}
	s.index.RUnlock()

	// 构建附加关系
	for _, instanceID := range instanceIDs {
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			log.Warn().Str("instance_id", instanceID).Err(err).Msg("Failed to get instance")
			continue
		}

		attachment := &entity.VolumeAttachment{
			VolumeID:   volumeID,
			InstanceID: instanceID,
			Device:     "/dev/vda", // 根卷默认设备
			State:      "attached",
			AttachTime: instance.CreatedAt,
		}

		// 如果实例停止,附加状态也是 detached
		if instance.State == "stopped" || instance.State == "terminated" {
			attachment.State = "detached"
		}

		attachments = append(attachments, attachment)
	}

	log.Debug().Int("count", len(attachments)).Msg("Got volume attachments")
	return attachments, nil
}

// ==================== 辅助函数 ====================

// getVolumesByIDs 根据卷 ID 列表获取卷
func (s *LibvirtMetadataStore) getVolumesByIDs(ctx context.Context, volumeIDs []string) ([]*entity.EBSVolume, error) {
	volumes := make([]*entity.EBSVolume, 0, len(volumeIDs))

	for _, id := range volumeIDs {
		volume, err := s.GetVolume(ctx, id)
		if err != nil {
			log.Warn().Str("volume_id", id).Err(err).Msg("Failed to get volume")
			continue
		}
		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// filterVolumesByIndex 使用内存索引过滤卷
func (s *LibvirtMetadataStore) filterVolumesByIndex(req *entity.DescribeVolumesRequest) []string {
	s.index.RLock()
	defer s.index.RUnlock()

	// 如果没有过滤条件,返回所有卷 ID
	if len(req.Filters) == 0 {
		ids := make([]string, 0, len(s.index.Volumes))
		for id := range s.index.Volumes {
			ids = append(ids, id)
		}
		return ids
	}

	// 使用过滤条件
	candidateSet := make(map[string]bool)
	firstFilter := true

	for _, filter := range req.Filters {
		var matchedIDs []string

		switch filter.Name {
		case "volume-type":
			for _, value := range filter.Values {
				if ids, exists := s.index.VolumesByType[value]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		case "status":
			for _, value := range filter.Values {
				if ids, exists := s.index.VolumesByState[value]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		case "tag":
			for _, tagPair := range filter.Values {
				if ids, exists := s.index.VolumesByTag[tagPair]; exists {
					matchedIDs = append(matchedIDs, ids...)
				}
			}

		default:
			continue
		}

		// 合并结果(交集)
		if firstFilter {
			for _, id := range matchedIDs {
				candidateSet[id] = true
			}
			firstFilter = false
		} else {
			newSet := make(map[string]bool)
			for _, id := range matchedIDs {
				if candidateSet[id] {
					newSet[id] = true
				}
			}
			candidateSet = newSet
		}
	}

	// 转换为切片
	result := make([]string, 0, len(candidateSet))
	for id := range candidateSet {
		result = append(result, id)
	}

	return result
}

// matchesVolumeFilters 检查卷是否匹配过滤条件
func (s *LibvirtMetadataStore) matchesVolumeFilters(volume *entity.EBSVolume, filters []entity.Filter) bool {
	for _, filter := range filters {
		switch filter.Name {
		case "volume-id":
			matched := false
			for _, value := range filter.Values {
				if volume.VolumeID == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "size":
			matched := false
			for _, value := range filter.Values {
				// 简化处理,直接字符串比较
				if fmt.Sprintf("%d", volume.SizeGB) == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	return true
}

// updateVolumeIndex 更新卷索引
func (s *LibvirtMetadataStore) updateVolumeIndex(volume *entity.EBSVolume, volumePath string) {
	s.index.Lock()
	defer s.index.Unlock()

	// 转换 Tags
	tags := make(map[string]string)
	for _, tag := range volume.Tags {
		tags[tag.Key] = tag.Value
	}

	idx := &VolumeIndex{
		ID:         volume.VolumeID,
		Path:       volumePath,
		State:      volume.State,
		VolumeType: volume.VolumeType,
		SizeGB:     volume.SizeGB,
		Tags:       tags,
	}

	s.index.Volumes[volume.VolumeID] = idx

	// 更新类型索引
	if _, exists := s.index.VolumesByType[volume.VolumeType]; !exists {
		s.index.VolumesByType[volume.VolumeType] = []string{}
	}
	s.index.VolumesByType[volume.VolumeType] = append(
		s.index.VolumesByType[volume.VolumeType],
		volume.VolumeID,
	)

	// 更新状态索引
	if _, exists := s.index.VolumesByState[volume.State]; !exists {
		s.index.VolumesByState[volume.State] = []string{}
	}
	s.index.VolumesByState[volume.State] = append(
		s.index.VolumesByState[volume.State],
		volume.VolumeID,
	)

	// 更新标签索引
	for _, tag := range volume.Tags {
		tagPair := fmt.Sprintf("%s=%s", tag.Key, tag.Value)
		if _, exists := s.index.VolumesByTag[tagPair]; !exists {
			s.index.VolumesByTag[tagPair] = []string{}
		}
		s.index.VolumesByTag[tagPair] = append(
			s.index.VolumesByTag[tagPair],
			volume.VolumeID,
		)
	}
}

// removeVolumeFromIndex 从索引中删除卷
func (s *LibvirtMetadataStore) removeVolumeFromIndex(volumeID string) {
	s.index.Lock()
	defer s.index.Unlock()

	idx, exists := s.index.Volumes[volumeID]
	if !exists {
		return
	}

	delete(s.index.Volumes, volumeID)

	// 从类型索引删除
	if ids, exists := s.index.VolumesByType[idx.VolumeType]; exists {
		s.index.VolumesByType[idx.VolumeType] = removeFromSlice(ids, volumeID)
	}

	// 从状态索引删除
	if ids, exists := s.index.VolumesByState[idx.State]; exists {
		s.index.VolumesByState[idx.State] = removeFromSlice(ids, volumeID)
	}

	// 从标签索引删除
	for k, v := range idx.Tags {
		tagPair := fmt.Sprintf("%s=%s", k, v)
		if ids, exists := s.index.VolumesByTag[tagPair]; exists {
			s.index.VolumesByTag[tagPair] = removeFromSlice(ids, volumeID)
		}
	}
}
