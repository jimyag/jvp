package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// SnapshotServiceInterface 定义快照服务接口
type SnapshotServiceInterface interface {
	CreateSnapshot(ctx context.Context, req *entity.CreateSnapshotRequest) (*entity.Snapshot, error)
	ListSnapshots(ctx context.Context, req *entity.ListSnapshotsRequest) ([]entity.Snapshot, error)
	DescribeSnapshot(ctx context.Context, req *entity.DescribeSnapshotRequest) (*entity.Snapshot, error)
	DeleteSnapshot(ctx context.Context, req *entity.DeleteSnapshotRequest) error
	RevertSnapshot(ctx context.Context, req *entity.RevertSnapshotRequest) error
}

type Snapshot struct {
	snapshotService SnapshotServiceInterface
}

func NewSnapshot(snapshotService *service.SnapshotService) *Snapshot {
	return &Snapshot{
		snapshotService: snapshotService,
	}
}

func (s *Snapshot) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create-snapshot", ginx.Adapt5(s.CreateSnapshot))
	router.POST("/list-snapshots", ginx.Adapt5(s.ListSnapshots))
	router.POST("/describe-snapshot", ginx.Adapt5(s.DescribeSnapshot))
	router.POST("/delete-snapshot", ginx.Adapt5(s.DeleteSnapshot))
	router.POST("/revert-snapshot", ginx.Adapt5(s.RevertSnapshot))
}

func (s *Snapshot) CreateSnapshot(ctx *gin.Context, req *entity.CreateSnapshotRequest) (*entity.CreateSnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Bool("with_memory", req.WithMemory).
		Msg("API: CreateSnapshot called")

	snapshot, err := s.snapshotService.CreateSnapshot(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create snapshot")
		return nil, err
	}

	return &entity.CreateSnapshotResponse{
		Snapshot: snapshot,
	}, nil
}

func (s *Snapshot) ListSnapshots(ctx *gin.Context, req *entity.ListSnapshotsRequest) (*entity.ListSnapshotsResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Msg("API: ListSnapshots called")

	snapshots, err := s.snapshotService.ListSnapshots(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list snapshots")
		return nil, err
	}

	return &entity.ListSnapshotsResponse{
		Snapshots: snapshots,
	}, nil
}

func (s *Snapshot) DescribeSnapshot(ctx *gin.Context, req *entity.DescribeSnapshotRequest) (*entity.DescribeSnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Msg("API: DescribeSnapshot called")

	snapshot, err := s.snapshotService.DescribeSnapshot(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to describe snapshot")
		return nil, err
	}

	return &entity.DescribeSnapshotResponse{
		Snapshot: snapshot,
	}, nil
}

func (s *Snapshot) DeleteSnapshot(ctx *gin.Context, req *entity.DeleteSnapshotRequest) (*entity.DeleteSnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Bool("delete_children", req.DeleteChildren).
		Bool("metadata_only", req.MetadataOnly).
		Bool("disks_only", req.DisksOnly).
		Msg("API: DeleteSnapshot called")

	if err := s.snapshotService.DeleteSnapshot(ctx, req); err != nil {
		logger.Error().Err(err).Msg("Failed to delete snapshot")
		return nil, err
	}

	return &entity.DeleteSnapshotResponse{
		Message: "Snapshot deleted successfully",
	}, nil
}

func (s *Snapshot) RevertSnapshot(ctx *gin.Context, req *entity.RevertSnapshotRequest) (*entity.RevertSnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Bool("start_after_revert", req.StartAfterRevert).
		Bool("force", req.Force).
		Msg("API: RevertSnapshot called")

	if err := s.snapshotService.RevertSnapshot(ctx, req); err != nil {
		logger.Error().Err(err).Msg("Failed to revert snapshot")
		return nil, err
	}

	return &entity.RevertSnapshotResponse{
		Message: "Snapshot reverted successfully",
	}, nil
}
