package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

type Snapshot struct {
	snapshotService *service.SnapshotService
}

func NewSnapshot(snapshotService *service.SnapshotService) *Snapshot {
	return &Snapshot{
		snapshotService: snapshotService,
	}
}

func (s *Snapshot) RegisterRoutes(router *gin.RouterGroup) {
	snapshotRouter := router.Group("/snapshots")
	snapshotRouter.POST("/create", ginx.Adapt5(s.CreateSnapshot))
	snapshotRouter.POST("/delete", ginx.Adapt5(s.DeleteSnapshot))
	snapshotRouter.POST("/describe", ginx.Adapt5(s.DescribeSnapshots))
	snapshotRouter.POST("/copy", ginx.Adapt5(s.CopySnapshot))
}

func (s *Snapshot) CreateSnapshot(ctx *gin.Context, req *entity.CreateSnapshotRequest) (*entity.CreateSnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("CreateSnapshot called")

	snapshot, err := s.snapshotService.CreateEBSSnapshot(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create snapshot")
		return nil, err
	}

	logger.Info().
		Str("snapshotID", snapshot.SnapshotID).
		Msg("Snapshot created successfully")

	return &entity.CreateSnapshotResponse{
		Snapshot: snapshot,
	}, nil
}

func (s *Snapshot) DeleteSnapshot(ctx *gin.Context, req *entity.DeleteSnapshotRequest) (*entity.DeleteSnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("snapshotID", req.SnapshotID).
		Msg("DeleteSnapshot called")

	err := s.snapshotService.DeleteEBSSnapshot(ctx, req.SnapshotID)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to delete snapshot")
		return nil, err
	}

	logger.Info().
		Str("snapshotID", req.SnapshotID).
		Msg("Snapshot deleted successfully")

	return &entity.DeleteSnapshotResponse{
		Return: true,
	}, nil
}

func (s *Snapshot) DescribeSnapshots(ctx *gin.Context, req *entity.DescribeSnapshotsRequest) (*entity.DescribeSnapshotsResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("DescribeSnapshots called")

	snapshots, err := s.snapshotService.DescribeEBSSnapshots(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe snapshots")
		return nil, err
	}

	logger.Info().
		Int("count", len(snapshots)).
		Msg("Snapshots described successfully")

	return &entity.DescribeSnapshotsResponse{
		Snapshots: snapshots,
	}, nil
}

func (s *Snapshot) CopySnapshot(ctx *gin.Context, req *entity.CopySnapshotRequest) (*entity.CopySnapshotResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("sourceSnapshotID", req.SourceSnapshotID).
		Msg("CopySnapshot called")

	snapshot, err := s.snapshotService.CopyEBSSnapshot(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to copy snapshot")
		return nil, err
	}

	logger.Info().
		Str("snapshotID", snapshot.SnapshotID).
		Msg("Snapshot copied successfully")

	return &entity.CopySnapshotResponse{
		SnapshotID: snapshot.SnapshotID,
		Snapshot:   snapshot,
	}, nil
}
