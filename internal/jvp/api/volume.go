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
	CreateEBSVolume(ctx context.Context, req *entity.CreateVolumeRequest) (*entity.EBSVolume, error)
	DeleteEBSVolume(ctx context.Context, volumeID string) error
	AttachEBSVolume(ctx context.Context, req *entity.AttachVolumeRequest) (*entity.VolumeAttachment, error)
	DetachEBSVolume(ctx context.Context, req *entity.DetachVolumeRequest) (*entity.VolumeAttachment, error)
	DescribeEBSVolumes(ctx context.Context, req *entity.DescribeVolumesRequest) ([]entity.EBSVolume, error)
	ModifyEBSVolume(ctx context.Context, req *entity.ModifyVolumeRequest) (*entity.VolumeModification, error)
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
	volumeRouter.POST("/describe", ginx.Adapt5(v.DescribeVolumes))
	volumeRouter.POST("/modify", ginx.Adapt5(v.ModifyVolume))
}

func (v *Volume) CreateVolume(ctx *gin.Context, req *entity.CreateVolumeRequest) (*entity.CreateVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("CreateVolume called")

	volume, err := v.volumeService.CreateEBSVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", volume.VolumeID).
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

	err := v.volumeService.DeleteEBSVolume(ctx, req.VolumeID)
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

	attachment, err := v.volumeService.AttachEBSVolume(ctx, req)
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

	attachment, err := v.volumeService.DetachEBSVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to detach volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("Volume detached successfully")

	return &entity.DetachVolumeResponse{
		Attachment: attachment,
	}, nil
}

func (v *Volume) DescribeVolumes(ctx *gin.Context, req *entity.DescribeVolumesRequest) (*entity.DescribeVolumesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("DescribeVolumes called")

	volumes, err := v.volumeService.DescribeEBSVolumes(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe volumes")
		return nil, err
	}

	logger.Info().
		Int("count", len(volumes)).
		Msg("Volumes described successfully")

	return &entity.DescribeVolumesResponse{
		Volumes: volumes,
	}, nil
}

func (v *Volume) ModifyVolume(ctx *gin.Context, req *entity.ModifyVolumeRequest) (*entity.ModifyVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Interface("request", req).
		Msg("ModifyVolume called")

	modification, err := v.volumeService.ModifyEBSVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to modify volume")
		return nil, err
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("Volume modified successfully")

	return &entity.ModifyVolumeResponse{
		VolumeModification: modification,
	}, nil
}
