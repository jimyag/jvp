package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// StoragePoolServiceInterface 存储池服务接口
type StoragePoolServiceInterface interface {
	ListStoragePools(ctx context.Context, includeVolumes bool) ([]entity.StoragePool, error)
	GetStoragePool(ctx context.Context, poolName string, includeVolumes bool) (*entity.StoragePool, error)
}

type StoragePoolAPI struct {
	storageService StoragePoolServiceInterface
}

func NewStoragePoolAPI(storageService *service.StorageService) *StoragePoolAPI {
	return &StoragePoolAPI{
		storageService: storageService,
	}
}

func (s *StoragePoolAPI) RegisterRoutes(router *gin.RouterGroup) {
	poolRouter := router.Group("/storage/pools")
	poolRouter.POST("/list", ginx.Adapt5(s.ListStoragePools))
	poolRouter.POST("/describe", ginx.Adapt5(s.DescribeStoragePool))
}

func (s *StoragePoolAPI) ListStoragePools(ctx *gin.Context, req *entity.ListStoragePoolsRequest) (*entity.ListStoragePoolsResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Bool("includeVolumes", req.IncludeVolumes).
		Msg("ListStoragePools called")

	pools, err := s.storageService.ListStoragePools(ctx, req.IncludeVolumes)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to list storage pools")
		return nil, err
	}

	logger.Info().
		Int("count", len(pools)).
		Msg("Storage pools listed successfully")

	return &entity.ListStoragePoolsResponse{
		Pools: pools,
	}, nil
}

func (s *StoragePoolAPI) DescribeStoragePool(ctx *gin.Context, req *entity.DescribeStoragePoolRequest) (*entity.DescribeStoragePoolResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("poolName", req.PoolName).
		Bool("includeVolumes", req.IncludeVolumes).
		Msg("DescribeStoragePool called")

	pool, err := s.storageService.GetStoragePool(ctx, req.PoolName, req.IncludeVolumes)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe storage pool")
		return nil, err
	}

	logger.Info().
		Str("poolName", req.PoolName).
		Msg("Storage pool described successfully")

	return &entity.DescribeStoragePoolResponse{
		Pool: pool,
	}, nil
}
