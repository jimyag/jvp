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
	CreateVolumeFromURL(ctx context.Context, req *entity.CreateVolumeFromURLRequest) (*entity.Volume, error)
	ListVolumes(ctx context.Context, req *entity.ListVolumesRequest) ([]entity.Volume, error)
	DescribeVolume(ctx context.Context, req *entity.DescribeVolumeRequest) (*entity.Volume, error)
	ResizeVolume(ctx context.Context, req *entity.ResizeVolumeRequest) (*entity.Volume, error)
	DeleteVolume(ctx context.Context, req *entity.DeleteVolumeRequest) error
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
	// Action 风格 API
	router.POST("/create-volume", ginx.Adapt5(v.CreateVolume))
	router.POST("/create-volume-from-url", ginx.Adapt5(v.CreateVolumeFromURL))
	router.POST("/list-volumes", ginx.Adapt5(v.ListVolumes))
	router.POST("/describe-volume", ginx.Adapt5(v.DescribeVolume))
	router.POST("/resize-volume", ginx.Adapt5(v.ResizeVolume))
	router.POST("/delete-volume", ginx.Adapt5(v.DeleteVolume))
}

func (v *Volume) CreateVolume(ctx *gin.Context, req *entity.CreateVolumeRequest) (*entity.CreateVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("name", req.Name).
		Uint64("size_gb", req.SizeGB).
		Str("format", req.Format).
		Msg("API: CreateVolume called")

	volume, err := v.volumeService.CreateVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create volume")
		return nil, err
	}

	logger.Info().
		Str("volume_id", volume.ID).
		Msg("Volume created successfully")

	return &entity.CreateVolumeResponse{
		Volume: volume,
	}, nil
}

func (v *Volume) ListVolumes(ctx *gin.Context, req *entity.ListVolumesRequest) (*entity.ListVolumesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Msg("API: ListVolumes called")

	volumes, err := v.volumeService.ListVolumes(ctx, req)
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
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("volume_id", req.VolumeID).
		Msg("API: DescribeVolume called")

	volume, err := v.volumeService.DescribeVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe volume")
		return nil, err
	}

	logger.Info().
		Str("volume_id", volume.ID).
		Msg("Volume described successfully")

	return &entity.DescribeVolumeResponse{
		Volume: volume,
	}, nil
}

func (v *Volume) ResizeVolume(ctx *gin.Context, req *entity.ResizeVolumeRequest) (*entity.ResizeVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("volume_id", req.VolumeID).
		Uint64("new_size_gb", req.NewSizeGB).
		Msg("API: ResizeVolume called")

	volume, err := v.volumeService.ResizeVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to resize volume")
		return nil, err
	}

	logger.Info().
		Str("volume_id", req.VolumeID).
		Uint64("new_size_gb", req.NewSizeGB).
		Msg("Volume resized successfully")

	return &entity.ResizeVolumeResponse{
		Volume: volume,
	}, nil
}

func (v *Volume) DeleteVolume(ctx *gin.Context, req *entity.DeleteVolumeRequest) (*entity.DeleteVolumeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("volume_id", req.VolumeID).
		Msg("API: DeleteVolume called")

	err := v.volumeService.DeleteVolume(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to delete volume")
		return nil, err
	}

	logger.Info().
		Str("volume_id", req.VolumeID).
		Msg("Volume deleted successfully")

	return &entity.DeleteVolumeResponse{
		Message: "Volume deleted successfully",
	}, nil
}

func (v *Volume) CreateVolumeFromURL(ctx *gin.Context, req *entity.CreateVolumeFromURLRequest) (*entity.CreateVolumeFromURLResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("name", req.Name).
		Str("url", req.URL).
		Msg("API: CreateVolumeFromURL called")

	volume, err := v.volumeService.CreateVolumeFromURL(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create volume from URL")
		return nil, err
	}

	logger.Info().
		Str("volume_id", volume.ID).
		Str("path", volume.Path).
		Msg("Volume created from URL successfully")

	return &entity.CreateVolumeFromURLResponse{
		Volume: volume,
	}, nil
}
