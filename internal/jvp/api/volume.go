package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// VolumeServiceInterface 定义卷服务的接口
type VolumeServiceInterface interface {
	CreateVolume(ctx context.Context, req *entity.CreateVolumeRequest) (*entity.Volume, error)
	DeleteVolume(ctx context.Context, volumeID string) error
	AttachVolume(ctx context.Context, req *entity.AttachVolumeRequest) (*entity.VolumeAttachment, error)
	DetachVolume(ctx context.Context, req *entity.DetachVolumeRequest) error
	ListVolumes(ctx context.Context) ([]entity.Volume, error)
	GetVolume(ctx context.Context, volumeID string) (*entity.Volume, error)
	ResizeVolume(ctx context.Context, volumeID string, newSizeGB uint64) error
}

type Volume struct {
	volumeService VolumeServiceInterface
}

func NewVolume(volumeService *service.VolumeService) *Volume {
	return &Volume{
		volumeService: volumeService,
	}
}

func (v *Volume) RegisterRoutes(router *gin.RouterGroup) {
	volumeRouter := router.Group("/volumes")
	volumeRouter.POST("/create", ginx.Adapt5(v.CreateVolume))
	volumeRouter.POST("/delete", ginx.Adapt5(v.DeleteVolume))
	volumeRouter.POST("/attach", ginx.Adapt5(v.AttachVolume))
	volumeRouter.POST("/detach", ginx.Adapt5(v.DetachVolume))
	volumeRouter.POST("/list", ginx.Adapt5(v.ListVolumes))
	volumeRouter.POST("/describe", ginx.Adapt5(v.DescribeVolume))
	volumeRouter.POST("/resize", ginx.Adapt5(v.ResizeVolume))
}

func (v *Volume) CreateVolume(ctx *gin.Context, req *entity.CreateVolumeRequest) (*entity.CreateVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("CreateVolume called")

	volume, err := v.volumeService.CreateVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", volume.ID).
		Msg("Volume created successfully")

	return &entity.CreateVolumeResponse{
		Volume: volume,
	}, nil
}

func (v *Volume) DeleteVolume(ctx *gin.Context, req *entity.DeleteVolumeRequest) (*entity.DeleteVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("DeleteVolume called")

	err := v.volumeService.DeleteVolume(ctx, req.VolumeID)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to delete volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("Volume deleted successfully")

	return &entity.DeleteVolumeResponse{
		Return: true,
	}, nil
}

func (v *Volume) AttachVolume(ctx *gin.Context, req *entity.AttachVolumeRequest) (*entity.AttachVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("AttachVolume called")

	attachment, err := v.volumeService.AttachVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to attach volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("Volume attached successfully")

	return &entity.AttachVolumeResponse{
		Attachment: attachment,
	}, nil
}

func (v *Volume) DetachVolume(ctx *gin.Context, req *entity.DetachVolumeRequest) (*entity.DetachVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("DetachVolume called")

	err := v.volumeService.DetachVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to detach volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("Volume detached successfully")

	return &entity.DetachVolumeResponse{
		Return: true,
	}, nil
}

func (v *Volume) ListVolumes(ctx *gin.Context, req *entity.ListVolumesRequest) (*entity.ListVolumesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("ListVolumes called")

	volumes, err := v.volumeService.ListVolumes(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to list volumes")
		return nil, err
	}

	logger.Info().
		Int("count", len(volumes)).
		Msg("Volumes listed successfully")

	return &entity.ListVolumesResponse{
		Volumes: volumes,
	}, nil
}

func (v *Volume) DescribeVolume(ctx *gin.Context, req *entity.DescribeVolumeRequest) (*entity.DescribeVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("DescribeVolume called")

	volume, err := v.volumeService.GetVolume(ctx, req.VolumeID)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", volume.ID).
		Msg("Volume described successfully")

	return &entity.DescribeVolumeResponse{
		Volume: volume,
	}, nil
}

func (v *Volume) ResizeVolume(ctx *gin.Context, req *entity.ResizeVolumeRequest) (*entity.ResizeVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Uint64("newSizeGB", req.NewSizeGB).
		Msg("ResizeVolume called")

	err := v.volumeService.ResizeVolume(ctx, req.VolumeID, req.NewSizeGB)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to resize volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Uint64("newSizeGB", req.NewSizeGB).
		Msg("Volume resized successfully")

	return &entity.ResizeVolumeResponse{
		Return: true,
	}, nil
}
