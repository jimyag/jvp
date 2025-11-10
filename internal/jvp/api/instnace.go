package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

type Instance struct {
	instanceService *service.InstanceService
}

func NewInstance(instanceService *service.InstanceService) *Instance {
	return &Instance{
		instanceService: instanceService,
	}
}

func (i *Instance) RegisterRoutes(router *gin.RouterGroup) {
	instanceRouter := router.Group("/instances")
	instanceRouter.POST("/run", ginx.Adapt5(i.RunInstances))
}

func (i *Instance) RunInstances(ctx *gin.Context, req *entity.RunInstanceRequest) (*entity.RunInstanceResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("RunInstances called")

	// 调用 Instance Service 创建实例
	instance, err := i.instanceService.RunInstance(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to run instance")
		return nil, err
	}

	logger.Info().
		Str("instance_id", instance.ID).
		Msg("Instance created successfully")

	return &entity.RunInstanceResponse{
		Instance: instance,
	}, nil
}
