package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

type Instance struct {
}

func NewInstance() *Instance {
	return &Instance{}
}

func (i *Instance) RegisterRoutes(router *gin.RouterGroup) {
	instanceRouter := router.Group("/instance")
	instanceRouter.POST("/run-instances", ginx.Adapt5(i.RunInstances))
}

func (i *Instance) RunInstances(ctx *gin.Context, req *entity.RunInstanceRequest) (*entity.RunInstanceResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msgf("RunInstances: %+v", req)
	return nil, nil
}
