package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
)

// StoragePoolServiceInterface 存储池服务接口
type StoragePoolServiceInterface interface {
	ListStoragePools(ctx context.Context, nodeName string) ([]entity.StoragePool, error)
	DescribeStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error)
	CreateStoragePool(ctx context.Context, nodeName, name, poolType, path string) (*entity.StoragePool, error)
	DeleteStoragePool(ctx context.Context, nodeName, poolName string, deleteVolumes bool) error
	StartStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error)
	StopStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error)
	RefreshStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error)
}

// StoragePoolAPI 存储池 API
type StoragePoolAPI struct {
	storagePoolService StoragePoolServiceInterface
}

// NewStoragePoolAPI 创建存储池 API
func NewStoragePoolAPI(storagePoolService *service.StoragePoolService) *StoragePoolAPI {
	return &StoragePoolAPI{
		storagePoolService: storagePoolService,
	}
}

// RegisterRoutes 注册路由 - Action 风格
func (a *StoragePoolAPI) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/list-storage-pools", ginx.Adapt5(a.ListStoragePools))
	r.POST("/describe-storage-pool", ginx.Adapt5(a.DescribeStoragePool))
	r.POST("/create-storage-pool", ginx.Adapt5(a.CreateStoragePool))
	r.POST("/delete-storage-pool", ginx.Adapt5(a.DeleteStoragePool))
	r.POST("/start-storage-pool", ginx.Adapt5(a.StartStoragePool))
	r.POST("/stop-storage-pool", ginx.Adapt5(a.StopStoragePool))
	r.POST("/refresh-storage-pool", ginx.Adapt5(a.RefreshStoragePool))
}

// ListStoragePools 列举存储池
func (a *StoragePoolAPI) ListStoragePools(ctx *gin.Context, req *entity.ListStoragePoolsRequest) (*entity.ListStoragePoolsResponse, error) {
	pools, err := a.storagePoolService.ListStoragePools(ctx.Request.Context(), req.NodeName)
	if err != nil {
		return nil, err
	}

	return &entity.ListStoragePoolsResponse{
		Pools: pools,
	}, nil
}

// DescribeStoragePool 查询存储池详情
func (a *StoragePoolAPI) DescribeStoragePool(ctx *gin.Context, req *entity.DescribeStoragePoolRequest) (*entity.DescribeStoragePoolResponse, error) {
	pool, err := a.storagePoolService.DescribeStoragePool(ctx.Request.Context(), req.NodeName, req.PoolName)
	if err != nil {
		return nil, err
	}

	return &entity.DescribeStoragePoolResponse{
		Pool: pool,
	}, nil
}

// CreateStoragePool 创建存储池
func (a *StoragePoolAPI) CreateStoragePool(ctx *gin.Context, req *entity.CreateStoragePoolRequest) (*entity.CreateStoragePoolResponse, error) {
	pool, err := a.storagePoolService.CreateStoragePool(
		ctx.Request.Context(),
		req.NodeName,
		req.Name,
		req.Type,
		req.Path,
	)
	if err != nil {
		return nil, err
	}

	return &entity.CreateStoragePoolResponse{
		Pool: pool,
	}, nil
}

// DeleteStoragePool 删除存储池
func (a *StoragePoolAPI) DeleteStoragePool(ctx *gin.Context, req *entity.DeleteStoragePoolRequest) (*entity.DeleteStoragePoolResponse, error) {
	if err := a.storagePoolService.DeleteStoragePool(ctx.Request.Context(), req.NodeName, req.PoolName, req.DeleteVolumes); err != nil {
		return nil, err
	}

	message := "Storage pool deleted successfully"
	if req.DeleteVolumes {
		message = "Storage pool and all volumes deleted successfully"
	}

	return &entity.DeleteStoragePoolResponse{
		Message: message,
	}, nil
}

// StartStoragePool 启动存储池
func (a *StoragePoolAPI) StartStoragePool(ctx *gin.Context, req *entity.StartStoragePoolRequest) (*entity.StartStoragePoolResponse, error) {
	pool, err := a.storagePoolService.StartStoragePool(ctx.Request.Context(), req.NodeName, req.PoolName)
	if err != nil {
		return nil, err
	}

	return &entity.StartStoragePoolResponse{
		Pool: pool,
	}, nil
}

// StopStoragePool 停止存储池
func (a *StoragePoolAPI) StopStoragePool(ctx *gin.Context, req *entity.StopStoragePoolRequest) (*entity.StopStoragePoolResponse, error) {
	pool, err := a.storagePoolService.StopStoragePool(ctx.Request.Context(), req.NodeName, req.PoolName)
	if err != nil {
		return nil, err
	}

	return &entity.StopStoragePoolResponse{
		Pool: pool,
	}, nil
}

// RefreshStoragePool 刷新存储池
func (a *StoragePoolAPI) RefreshStoragePool(ctx *gin.Context, req *entity.RefreshStoragePoolRequest) (*entity.RefreshStoragePoolResponse, error) {
	pool, err := a.storagePoolService.RefreshStoragePool(ctx.Request.Context(), req.NodeName, req.PoolName)
	if err != nil {
		return nil, err
	}

	return &entity.RefreshStoragePoolResponse{
		Pool: pool,
	}, nil
}
