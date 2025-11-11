package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// InstanceServiceInterface 定义实例服务的接口
type InstanceServiceInterface interface {
	RunInstance(ctx context.Context, req *entity.RunInstanceRequest) (*entity.Instance, error)
	DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]entity.Instance, error)
	TerminateInstances(ctx context.Context, req *entity.TerminateInstancesRequest) ([]entity.InstanceStateChange, error)
	StopInstances(ctx context.Context, req *entity.StopInstancesRequest) ([]entity.InstanceStateChange, error)
	StartInstances(ctx context.Context, req *entity.StartInstancesRequest) ([]entity.InstanceStateChange, error)
	RebootInstances(ctx context.Context, req *entity.RebootInstancesRequest) ([]entity.InstanceStateChange, error)
	ModifyInstanceAttribute(ctx context.Context, req *entity.ModifyInstanceAttributeRequest) (*entity.Instance, error)
	ResetPassword(ctx context.Context, req *entity.ResetPasswordRequest) (*entity.ResetPasswordResponse, error)
}

type Instance struct {
	instanceService InstanceServiceInterface
}

func NewInstance(instanceService *service.InstanceService) *Instance {
	return &Instance{
		instanceService: instanceService,
	}
}

func (i *Instance) RegisterRoutes(router *gin.RouterGroup) {
	instanceRouter := router.Group("/instances")
	instanceRouter.POST("/run", ginx.Adapt5(i.RunInstances))
	instanceRouter.POST("/describe", ginx.Adapt5(i.DescribeInstances))
	instanceRouter.POST("/terminate", ginx.Adapt5(i.TerminateInstances))
	instanceRouter.POST("/stop", ginx.Adapt5(i.StopInstances))
	instanceRouter.POST("/start", ginx.Adapt5(i.StartInstances))
	instanceRouter.POST("/reboot", ginx.Adapt5(i.RebootInstances))
	instanceRouter.POST("/modify-attribute", ginx.Adapt5(i.ModifyInstanceAttribute))
	instanceRouter.POST("/reset-password", ginx.Adapt5(i.ResetPassword))
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

func (i *Instance) DescribeInstances(ctx *gin.Context, req *entity.DescribeInstancesRequest) (*entity.DescribeInstancesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("DescribeInstances called")

	instances, err := i.instanceService.DescribeInstances(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe instances")
		return nil, err
	}

	logger.Info().
		Int("count", len(instances)).
		Msg("Instances described successfully")

	return &entity.DescribeInstancesResponse{
		Instances: instances,
	}, nil
}

func (i *Instance) TerminateInstances(ctx *gin.Context, req *entity.TerminateInstancesRequest) (*entity.TerminateInstancesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("TerminateInstances called")

	changes, err := i.instanceService.TerminateInstances(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to terminate instances")
		return nil, err
	}

	logger.Info().
		Int("count", len(changes)).
		Msg("Instances terminated successfully")

	return &entity.TerminateInstancesResponse{
		TerminatingInstances: changes,
	}, nil
}

func (i *Instance) StopInstances(ctx *gin.Context, req *entity.StopInstancesRequest) (*entity.StopInstancesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Bool("force", req.Force).
		Msg("StopInstances called")

	changes, err := i.instanceService.StopInstances(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to stop instances")
		return nil, err
	}

	logger.Info().
		Int("count", len(changes)).
		Msg("Instances stopped successfully")

	return &entity.StopInstancesResponse{
		StoppingInstances: changes,
	}, nil
}

func (i *Instance) StartInstances(ctx *gin.Context, req *entity.StartInstancesRequest) (*entity.StartInstancesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("StartInstances called")

	changes, err := i.instanceService.StartInstances(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to start instances")
		return nil, err
	}

	logger.Info().
		Int("count", len(changes)).
		Msg("Instances started successfully")

	return &entity.StartInstancesResponse{
		StartingInstances: changes,
	}, nil
}

func (i *Instance) RebootInstances(ctx *gin.Context, req *entity.RebootInstancesRequest) (*entity.RebootInstancesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("RebootInstances called")

	changes, err := i.instanceService.RebootInstances(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to reboot instances")
		return nil, err
	}

	logger.Info().
		Int("count", len(changes)).
		Msg("Instances rebooted successfully")

	return &entity.RebootInstancesResponse{
		RebootingInstances: changes,
	}, nil
}

func (i *Instance) ModifyInstanceAttribute(ctx *gin.Context, req *entity.ModifyInstanceAttributeRequest) (*entity.ModifyInstanceAttributeResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instanceID", req.InstanceID).
		Interface("request", req).
		Msg("ModifyInstanceAttribute called")

	instance, err := i.instanceService.ModifyInstanceAttribute(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to modify instance attribute")
		return nil, err
	}

	logger.Info().
		Str("instanceID", req.InstanceID).
		Msg("Instance attribute modified successfully")

	return &entity.ModifyInstanceAttributeResponse{
		Instance: instance,
	}, nil
}

func (i *Instance) ResetPassword(ctx *gin.Context, req *entity.ResetPasswordRequest) (*entity.ResetPasswordResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instance_id", req.InstanceID).
		Int("user_count", len(req.Users)).
		Msg("ResetPassword called")

	// 调用 Instance Service 重置密码
	response, err := i.instanceService.ResetPassword(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", req.InstanceID).
			Msg("Failed to reset password")
		return nil, err
	}

	logger.Info().
		Str("instance_id", req.InstanceID).
		Strs("users", response.Users).
		Msg("Password reset successfully")

	return response, nil
}
