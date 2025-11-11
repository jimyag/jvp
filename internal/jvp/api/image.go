package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// ImageServiceInterface 定义镜像服务的接口
type ImageServiceInterface interface {
	CreateImageFromInstance(ctx context.Context, req *entity.CreateImageFromInstanceRequest) (*entity.Image, error)
	DescribeImages(ctx context.Context, req *entity.DescribeImagesRequest) ([]entity.Image, error)
	RegisterImage(ctx context.Context, req *entity.RegisterImageRequest) (*entity.Image, error)
	DeleteImage(ctx context.Context, imageID string) error
}

type Image struct {
	imageService ImageServiceInterface
}

func NewImage(imageService *service.ImageService) *Image {
	return &Image{
		imageService: imageService,
	}
}

func (i *Image) RegisterRoutes(router *gin.RouterGroup) {
	imageRouter := router.Group("/images")
	imageRouter.POST("/create", ginx.Adapt5(i.CreateImage))
	imageRouter.POST("/describe", ginx.Adapt5(i.DescribeImages))
	imageRouter.POST("/register", ginx.Adapt5(i.RegisterImage))
	imageRouter.POST("/deregister", ginx.Adapt5(i.DeregisterImage))
}

func (i *Image) CreateImage(ctx *gin.Context, req *entity.CreateImageFromInstanceRequest) (*entity.CreateImageFromInstanceResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instanceID", req.InstanceID).
		Str("imageName", req.ImageName).
		Msg("CreateImage called")

	image, err := i.imageService.CreateImageFromInstance(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create image from instance")
		return nil, err
	}

	logger.Info().
		Str("imageID", image.ID).
		Str("instanceID", req.InstanceID).
		Msg("Image created successfully")

	return &entity.CreateImageFromInstanceResponse{
		Image: image,
	}, nil
}

func (i *Image) DescribeImages(ctx *gin.Context, req *entity.DescribeImagesRequest) (*entity.DescribeImagesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("DescribeImages called")

	images, err := i.imageService.DescribeImages(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe images")
		return nil, err
	}

	logger.Info().
		Int("count", len(images)).
		Msg("Images described successfully")

	return &entity.DescribeImagesResponse{
		Images: images,
	}, nil
}

func (i *Image) RegisterImage(ctx *gin.Context, req *entity.RegisterImageRequest) (*entity.Image, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("name", req.Name).
		Str("path", req.Path).
		Msg("RegisterImage called")

	image, err := i.imageService.RegisterImage(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to register image")
		return nil, err
	}

	logger.Info().
		Str("imageID", image.ID).
		Str("name", req.Name).
		Msg("Image registered successfully")

	return image, nil
}

func (i *Image) DeregisterImage(ctx *gin.Context, req *entity.DeregisterImageRequest) (*entity.DeregisterImageResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("imageID", req.ImageID).
		Msg("DeregisterImage called")

	err := i.imageService.DeleteImage(ctx, req.ImageID)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to deregister image")
		return nil, err
	}

	logger.Info().
		Str("imageID", req.ImageID).
		Msg("Image deregistered successfully")

	return &entity.DeregisterImageResponse{
		Return: true,
	}, nil
}
