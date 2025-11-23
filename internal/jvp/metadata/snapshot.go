package metadata

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/rs/zerolog/log"
)

// ==================== Snapshot 管理器 ====================

// SnapshotManager 快照管理器
type SnapshotManager struct {
	snapshotsDir string
}

// NewSnapshotManager 创建快照管理器
func NewSnapshotManager(snapshotsDir string) *SnapshotManager {
	return &SnapshotManager{
		snapshotsDir: snapshotsDir,
	}
}

// SaveSnapshot 保存快照元数据
func (s *LibvirtMetadataStore) SaveSnapshot(ctx context.Context, snapshot *entity.EBSSnapshot) error {
	log.Debug().
		Str("snapshot_id", snapshot.SnapshotID).
		Str("volume_id", snapshot.VolumeID).
		Str("state", snapshot.State).
		Msg("Saving snapshot metadata")

	// 1. 加载卷的快照索引
	indexPath := getSnapshotIndexPath(s.config.BasePath, snapshot.VolumeID)
	var index SnapshotIndex

	if fileExists(indexPath) {
		if err := loadJSONFile(indexPath, &index); err != nil {
			log.Warn().Err(err).Msg("Failed to load snapshot index, creating new")
			index = SnapshotIndex{
				VolumeID:  snapshot.VolumeID,
				Snapshots: []Snapshot{},
			}
		}
	} else {
		// 创建新的索引
		volumePath := s.getVolumePathByID(snapshot.VolumeID)
		index = SnapshotIndex{
			VolumeID:   snapshot.VolumeID,
			VolumePath: volumePath,
			Snapshots:  []Snapshot{},
		}
	}

	// 2. 解析 Progress 和 StartTime
	progress := 0
	if snapshot.Progress != "" {
		fmt.Sscanf(snapshot.Progress, "%d", &progress)
	}
	startTime, _ := time.Parse(time.RFC3339, snapshot.StartTime)

	// 3. 转换 Tags
	tags := make(map[string]string)
	for _, tag := range snapshot.Tags {
		tags[tag.Key] = tag.Value
	}

	// 4. 查找或添加快照
	found := false
	for i, snap := range index.Snapshots {
		if snap.ID == snapshot.SnapshotID {
			// 更新现有快照
			index.Snapshots[i] = Snapshot{
				ID:               snapshot.SnapshotID,
				Description:      snapshot.Description,
				QemuSnapshotName: fmt.Sprintf("jvp-%s", snapshot.SnapshotID),
				SizeGB:           snapshot.VolumeSizeGB,
				State:            snapshot.State,
				Progress:         progress,
				StartTime:        startTime,
				CompletionTime:   time.Now(),
				OwnerID:          snapshot.OwnerID,
				Tags:             tags,
			}
			found = true
			break
		}
	}

	if !found {
		// 添加新快照
		index.Snapshots = append(index.Snapshots, Snapshot{
			ID:               snapshot.SnapshotID,
			Description:      snapshot.Description,
			QemuSnapshotName: fmt.Sprintf("jvp-%s", snapshot.SnapshotID),
			SizeGB:           snapshot.VolumeSizeGB,
			State:            snapshot.State,
			Progress:         progress,
			StartTime:        startTime,
			OwnerID:          snapshot.OwnerID,
			Tags:             tags,
		})
	}

	// 3. 保存索引文件
	if err := saveJSONFile(indexPath, index); err != nil {
		return fmt.Errorf("save snapshot index: %w", err)
	}

	// 4. 更新内存索引
	s.updateSnapshotIndex(snapshot)

	log.Info().
		Str("snapshot_id", snapshot.SnapshotID).
		Str("index_path", indexPath).
		Msg("Snapshot metadata saved successfully")

	return nil
}

// GetSnapshot 获取单个快照
func (s *LibvirtMetadataStore) GetSnapshot(ctx context.Context, snapshotID string) (*entity.EBSSnapshot, error) {
	log.Debug().Str("snapshot_id", snapshotID).Msg("Getting snapshot")

	// 1. 从索引获取快照信息
	s.index.RLock()
	snapIdx, exists := s.index.Snapshots[snapshotID]
	s.index.RUnlock()

	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// 2. 加载卷的快照索引文件
	indexPath := getSnapshotIndexPath(s.config.BasePath, snapIdx.VolumeID)
	var index SnapshotIndex
	if err := loadJSONFile(indexPath, &index); err != nil {
		return nil, fmt.Errorf("load snapshot index: %w", err)
	}

	// 3. 查找快照
	for _, snap := range index.Snapshots {
		if snap.ID == snapshotID {
			// 转换 Tags
			tags := make([]entity.Tag, 0, len(snap.Tags))
			for k, v := range snap.Tags {
				tags = append(tags, entity.Tag{Key: k, Value: v})
			}

			return &entity.EBSSnapshot{
				SnapshotID:   snap.ID,
				VolumeID:     index.VolumeID,
				Description:  snap.Description,
				VolumeSizeGB: snap.SizeGB,
				State:        snap.State,
				Progress:     fmt.Sprintf("%d%%", snap.Progress),
				StartTime:    snap.StartTime.Format(time.RFC3339),
				OwnerID:      snap.OwnerID,
				Tags:         tags,
			}, nil
		}
	}

	return nil, fmt.Errorf("snapshot not found in index: %s", snapshotID)
}

// ListSnapshots 列出卷的所有快照
func (s *LibvirtMetadataStore) ListSnapshots(ctx context.Context, volumeID string) ([]*entity.EBSSnapshot, error) {
	log.Debug().Str("volume_id", volumeID).Msg("Listing snapshots for volume")

	// 1. 加载卷的快照索引
	indexPath := getSnapshotIndexPath(s.config.BasePath, volumeID)
	if !fileExists(indexPath) {
		// 没有快照
		return []*entity.EBSSnapshot{}, nil
	}

	var index SnapshotIndex
	if err := loadJSONFile(indexPath, &index); err != nil {
		return nil, fmt.Errorf("load snapshot index: %w", err)
	}

	// 2. 转换为实体
	snapshots := make([]*entity.EBSSnapshot, 0, len(index.Snapshots))
	for _, snap := range index.Snapshots {
		// 转换 Tags
		tags := make([]entity.Tag, 0, len(snap.Tags))
		for k, v := range snap.Tags {
			tags = append(tags, entity.Tag{Key: k, Value: v})
		}

		snapshots = append(snapshots, &entity.EBSSnapshot{
			SnapshotID:   snap.ID,
			VolumeID:     index.VolumeID,
			Description:  snap.Description,
			VolumeSizeGB: snap.SizeGB,
			State:        snap.State,
			Progress:     fmt.Sprintf("%d%%", snap.Progress),
			StartTime:    snap.StartTime.Format(time.RFC3339),
			OwnerID:      snap.OwnerID,
			Tags:         tags,
		})
	}

	log.Debug().Int("count", len(snapshots)).Msg("Listed snapshots")
	return snapshots, nil
}

// DescribeSnapshots 查询快照(支持过滤)
func (s *LibvirtMetadataStore) DescribeSnapshots(ctx context.Context, req *entity.DescribeSnapshotsRequest) ([]*entity.EBSSnapshot, error) {
	log.Debug().
		Strs("snapshot_ids", req.SnapshotIDs).
		Interface("filters", req.Filters).
		Msg("Describing snapshots")

	// 如果指定了快照 ID,直接查询
	if len(req.SnapshotIDs) > 0 {
		return s.getSnapshotsByIDs(ctx, req.SnapshotIDs)
	}

	// 否则,遍历所有快照索引文件
	snapshotsDir := filepath.Join(s.config.BasePath, "volumes", ".snapshots")
	indexFiles, err := filepath.Glob(filepath.Join(snapshotsDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob snapshot index files: %w", err)
	}

	snapshots := make([]*entity.EBSSnapshot, 0)

	for _, indexPath := range indexFiles {
		var index SnapshotIndex
		if err := loadJSONFile(indexPath, &index); err != nil {
			log.Warn().Str("index_path", indexPath).Err(err).Msg("Failed to load snapshot index")
			continue
		}

		for _, snap := range index.Snapshots {
			// 转换 Tags
			tags := make([]entity.Tag, 0, len(snap.Tags))
			for k, v := range snap.Tags {
				tags = append(tags, entity.Tag{Key: k, Value: v})
			}

			snapshot := &entity.EBSSnapshot{
				SnapshotID:   snap.ID,
				VolumeID:     index.VolumeID,
				Description:  snap.Description,
				VolumeSizeGB: snap.SizeGB,
				State:        snap.State,
				Progress:     fmt.Sprintf("%d%%", snap.Progress),
				StartTime:    snap.StartTime.Format(time.RFC3339),
				OwnerID:      snap.OwnerID,
				Tags:         tags,
			}

			// 应用过滤条件
			if s.matchesSnapshotFilters(snapshot, req.Filters) {
				snapshots = append(snapshots, snapshot)
			}
		}
	}

	log.Debug().Int("count", len(snapshots)).Msg("Described snapshots")
	return snapshots, nil
}

// DeleteSnapshot 删除快照元数据
func (s *LibvirtMetadataStore) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	log.Debug().Str("snapshot_id", snapshotID).Msg("Deleting snapshot metadata")

	// 1. 从索引获取快照信息
	s.index.RLock()
	snapIdx, exists := s.index.Snapshots[snapshotID]
	s.index.RUnlock()

	if !exists {
		// 快照不存在,认为已删除
		return nil
	}

	volumeID := snapIdx.VolumeID

	// 2. 加载卷的快照索引
	indexPath := getSnapshotIndexPath(s.config.BasePath, volumeID)
	var index SnapshotIndex
	if err := loadJSONFile(indexPath, &index); err != nil {
		return fmt.Errorf("load snapshot index: %w", err)
	}

	// 3. 从索引中删除快照
	newSnapshots := make([]Snapshot, 0, len(index.Snapshots))
	for _, snap := range index.Snapshots {
		if snap.ID != snapshotID {
			newSnapshots = append(newSnapshots, snap)
		}
	}
	index.Snapshots = newSnapshots

	// 4. 保存索引
	if err := saveJSONFile(indexPath, index); err != nil {
		return fmt.Errorf("save snapshot index: %w", err)
	}

	// 5. 从内存索引删除
	s.removeSnapshotFromIndex(snapshotID, volumeID)

	log.Info().Str("snapshot_id", snapshotID).Msg("Snapshot metadata deleted")
	return nil
}

// ==================== 辅助函数 ====================

// getSnapshotsByIDs 根据快照 ID 列表获取快照
func (s *LibvirtMetadataStore) getSnapshotsByIDs(ctx context.Context, snapshotIDs []string) ([]*entity.EBSSnapshot, error) {
	snapshots := make([]*entity.EBSSnapshot, 0, len(snapshotIDs))

	for _, id := range snapshotIDs {
		snapshot, err := s.GetSnapshot(ctx, id)
		if err != nil {
			log.Warn().Str("snapshot_id", id).Err(err).Msg("Failed to get snapshot")
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// matchesSnapshotFilters 检查快照是否匹配过滤条件
func (s *LibvirtMetadataStore) matchesSnapshotFilters(snapshot *entity.EBSSnapshot, filters []entity.Filter) bool {
	for _, filter := range filters {
		switch filter.Name {
		case "snapshot-id":
			matched := false
			for _, value := range filter.Values {
				if snapshot.SnapshotID == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "volume-id":
			matched := false
			for _, value := range filter.Values {
				if snapshot.VolumeID == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "status":
			matched := false
			for _, value := range filter.Values {
				if snapshot.State == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "tag":
			// 标签过滤
			for _, tagPair := range filter.Values {
				// 解析 tag pair (format: "key=value")
				matched := false
				for _, tag := range snapshot.Tags {
					if fmt.Sprintf("%s=%s", tag.Key, tag.Value) == tagPair {
						matched = true
						break
					}
				}
				if !matched {
					return false
				}
			}
		}
	}

	return true
}

// updateSnapshotIndex 更新快照索引
func (s *LibvirtMetadataStore) updateSnapshotIndex(snapshot *entity.EBSSnapshot) {
	s.index.Lock()
	defer s.index.Unlock()

	// 转换 Tags
	tags := make(map[string]string)
	for _, tag := range snapshot.Tags {
		tags[tag.Key] = tag.Value
	}

	// 解析 StartTime
	startTime, _ := time.Parse(time.RFC3339, snapshot.StartTime)

	idx := &SnapshotIndexItem{
		ID:         snapshot.SnapshotID,
		VolumeID:   snapshot.VolumeID,
		State:      snapshot.State,
		SizeGB:     snapshot.VolumeSizeGB,
		CreateTime: startTime,
		Tags:       tags,
	}

	s.index.Snapshots[snapshot.SnapshotID] = idx

	// 更新卷快照索引
	if _, exists := s.index.SnapshotsByVolume[snapshot.VolumeID]; !exists {
		s.index.SnapshotsByVolume[snapshot.VolumeID] = []string{}
	}
	s.index.SnapshotsByVolume[snapshot.VolumeID] = append(
		s.index.SnapshotsByVolume[snapshot.VolumeID],
		snapshot.SnapshotID,
	)
}

// removeSnapshotFromIndex 从索引中删除快照
func (s *LibvirtMetadataStore) removeSnapshotFromIndex(snapshotID string, volumeID string) {
	s.index.Lock()
	defer s.index.Unlock()

	delete(s.index.Snapshots, snapshotID)

	// 从卷快照索引删除
	if ids, exists := s.index.SnapshotsByVolume[volumeID]; exists {
		s.index.SnapshotsByVolume[volumeID] = removeFromSlice(ids, snapshotID)
	}
}
