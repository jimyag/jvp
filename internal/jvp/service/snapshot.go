package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// SnapshotService EBS Snapshot 服务
type SnapshotService struct {
	storageService *StorageService
	libvirtClient  *libvirt.Client
	idGen          *idgen.Generator
	snapshotRepo   repository.SnapshotRepository
}

// NewSnapshotService 创建新的 Snapshot Service
func NewSnapshotService(
	storageService *StorageService,
	libvirtClient *libvirt.Client,
	repo *repository.Repository,
) *SnapshotService {
	return &SnapshotService{
		storageService: storageService,
		libvirtClient:  libvirtClient,
		idGen:          idgen.New(),
		snapshotRepo:   repository.NewSnapshotRepository(repo.DB()),
	}
}

// CreateEBSSnapshot 创建 EBS 快照
func (s *SnapshotService) CreateEBSSnapshot(ctx context.Context, req *entity.CreateSnapshotRequest) (*entity.EBSSnapshot, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("Creating EBS snapshot")

	// 生成 Snapshot ID
	snapshotID, err := s.idGen.GenerateSnapshotID()
	if err != nil {
		return nil, fmt.Errorf("generate snapshot ID: %w", err)
	}

	// 获取卷信息
	volume, err := s.storageService.GetVolume(ctx, req.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	// 使用 qemu-img 创建快照
	qemuImgClient := qemuimg.New("")
	err = qemuImgClient.Snapshot(ctx, volume.Path, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	snapshot := &entity.EBSSnapshot{
		SnapshotID:   snapshotID,
		VolumeID:     req.VolumeID,
		State:        "completed",
		StartTime:    time.Now().Format(time.RFC3339),
		Progress:     "100%",
		OwnerID:      "default",
		Description:  req.Description,
		Encrypted:    false,
		VolumeSizeGB: volume.CapacityB / (1024 * 1024 * 1024),
		Tags:         extractTags(req.TagSpecifications, "snapshot"),
	}

	// 保存到数据库
	snapshotModel, err := snapshotEntityToModel(snapshot)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to convert snapshot to model", err)
	}
	if err := s.snapshotRepo.Create(ctx, snapshotModel); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save snapshot to database", err)
	}
	logger.Info().Str("snapshotID", snapshotID).Msg("Snapshot saved to database")

	logger.Info().
		Str("snapshotID", snapshotID).
		Msg("EBS snapshot created successfully")

	return snapshot, nil
}

// DeleteEBSSnapshot 删除 EBS 快照
func (s *SnapshotService) DeleteEBSSnapshot(ctx context.Context, snapshotID string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("snapshotID", snapshotID).Msg("Deleting EBS snapshot")

	// 从数据库软删除
	if err := s.snapshotRepo.Delete(ctx, snapshotID); err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete snapshot from database", err)
	}

	// TODO: 删除快照文件

	logger.Info().Str("snapshotID", snapshotID).Msg("EBS snapshot deleted successfully")
	return nil
}

// DescribeEBSSnapshots 描述 EBS 快照
func (s *SnapshotService) DescribeEBSSnapshots(ctx context.Context, req *entity.DescribeSnapshotsRequest) ([]entity.EBSSnapshot, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Describing EBS snapshots")

	var snapshots []entity.EBSSnapshot

	// 构建过滤器
	filters := make(map[string]interface{})
	if len(req.SnapshotIDs) > 0 {
		// 如果指定了 SnapshotIDs，逐个查询
		for _, snapshotID := range req.SnapshotIDs {
			snapshotModel, err := s.snapshotRepo.GetByID(ctx, snapshotID)
			if err != nil {
				logger.Warn().Err(err).Str("snapshotID", snapshotID).Msg("Snapshot not found, skipping")
				continue
			}
			snapshot, err := snapshotModelToEntity(snapshotModel)
			if err != nil {
				logger.Warn().Err(err).Str("snapshotID", snapshotID).Msg("Failed to convert snapshot model to entity")
				continue
			}
			snapshots = append(snapshots, *snapshot)
		}
	} else {
		// 应用过滤器
		if len(req.Filters) > 0 {
			for _, filter := range req.Filters {
				switch filter.Name {
				case "state":
					if len(filter.Values) > 0 {
						filters["state"] = filter.Values[0]
					}
				case "volume-id":
					if len(filter.Values) > 0 {
						filters["volume_id"] = filter.Values[0]
					}
				case "owner-id":
					if len(filter.Values) > 0 {
						filters["owner_id"] = filter.Values[0]
					}
				}
			}
		}

		// 从数据库查询
		snapshotModels, err := s.snapshotRepo.List(ctx, filters)
		if err != nil {
			return nil, fmt.Errorf("list snapshots from database: %w", err)
		}

		for _, snapshotModel := range snapshotModels {
			snapshot, err := snapshotModelToEntity(snapshotModel)
			if err != nil {
				logger.Warn().Err(err).Str("snapshotID", snapshotModel.ID).Msg("Failed to convert snapshot model to entity")
				continue
			}
			snapshots = append(snapshots, *snapshot)
		}
	}

	// TODO: 应用分页

	return snapshots, nil
}

// CopyEBSSnapshot 复制 EBS 快照
func (s *SnapshotService) CopyEBSSnapshot(ctx context.Context, req *entity.CopySnapshotRequest) (*entity.EBSSnapshot, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("sourceSnapshotID", req.SourceSnapshotID).
		Str("sourceRegion", req.SourceRegion).
		Msg("Copying EBS snapshot")

	// 生成新的 Snapshot ID
	snapshotID, err := s.idGen.GenerateSnapshotID()
	if err != nil {
		return nil, fmt.Errorf("generate snapshot ID: %w", err)
	}

	// TODO: 复制快照文件

	// 获取源快照信息
	sourceSnapshot, err := s.DescribeEBSSnapshots(ctx, &entity.DescribeSnapshotsRequest{
		SnapshotIDs: []string{req.SourceSnapshotID},
	})
	if err != nil || len(sourceSnapshot) == 0 {
		return nil, fmt.Errorf("source snapshot %s not found", req.SourceSnapshotID)
	}
	source := sourceSnapshot[0]

	snapshot := &entity.EBSSnapshot{
		SnapshotID:   snapshotID,
		VolumeID:     source.VolumeID,
		State:        "pending",
		StartTime:    time.Now().Format(time.RFC3339),
		Progress:     "0%",
		OwnerID:      "default",
		Description:  req.Description,
		Encrypted:    req.Encrypted,
		VolumeSizeGB: source.VolumeSizeGB,
		Tags:         extractTags(req.TagSpecifications, "snapshot"),
	}

	// 保存到数据库
	snapshotModel, err := snapshotEntityToModel(snapshot)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to convert snapshot to model", err)
	}
	if err := s.snapshotRepo.Create(ctx, snapshotModel); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save snapshot to database", err)
	}
	logger.Info().Str("snapshotID", snapshotID).Msg("Snapshot saved to database")

	logger.Info().
		Str("snapshotID", snapshotID).
		Msg("EBS snapshot copied successfully")

	return snapshot, nil
}
