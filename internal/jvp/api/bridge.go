package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
)

// BridgeServiceInterface 网桥服务接口
type BridgeServiceInterface interface {
	ListBridges(ctx context.Context, nodeName string) ([]entity.HostBridge, error)
	CreateBridge(ctx context.Context, req *entity.CreateBridgeRequest) (*entity.HostBridge, error)
	DeleteBridge(ctx context.Context, nodeName, bridgeName string) error
	ListAvailableInterfaces(ctx context.Context, nodeName string) ([]entity.NetworkInterface, error)
}

// BridgeAPI 网桥 API
type BridgeAPI struct {
	bridgeService BridgeServiceInterface
}

// NewBridgeAPI 创建网桥 API
func NewBridgeAPI(bridgeService *service.BridgeService) *BridgeAPI {
	return &BridgeAPI{
		bridgeService: bridgeService,
	}
}

// RegisterRoutes 注册路由 - Action 风格
func (a *BridgeAPI) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/list-bridges", ginx.Adapt5(a.ListBridges))
	r.POST("/create-bridge", ginx.Adapt5(a.CreateBridge))
	r.POST("/delete-bridge", ginx.Adapt5(a.DeleteBridge))
	r.POST("/list-available-interfaces", ginx.Adapt5(a.ListAvailableInterfaces))
}

// ListBridges 列举网桥
func (a *BridgeAPI) ListBridges(ctx *gin.Context, req *entity.ListBridgesRequest) (*entity.ListBridgesResponse, error) {
	bridges, err := a.bridgeService.ListBridges(ctx.Request.Context(), req.NodeName)
	if err != nil {
		return nil, err
	}

	return &entity.ListBridgesResponse{
		Bridges: bridges,
	}, nil
}

// CreateBridge 创建网桥
func (a *BridgeAPI) CreateBridge(ctx *gin.Context, req *entity.CreateBridgeRequest) (*entity.CreateBridgeResponse, error) {
	bridge, err := a.bridgeService.CreateBridge(ctx.Request.Context(), req)
	if err != nil {
		return nil, err
	}

	return &entity.CreateBridgeResponse{
		Bridge: bridge,
	}, nil
}

// DeleteBridge 删除网桥
func (a *BridgeAPI) DeleteBridge(ctx *gin.Context, req *entity.DeleteBridgeRequest) (*entity.DeleteBridgeResponse, error) {
	if err := a.bridgeService.DeleteBridge(ctx.Request.Context(), req.NodeName, req.BridgeName); err != nil {
		return nil, err
	}

	return &entity.DeleteBridgeResponse{
		Message: "Bridge deleted successfully",
	}, nil
}

// ListAvailableInterfaces 列举可用于绑定到网桥的网络接口
func (a *BridgeAPI) ListAvailableInterfaces(ctx *gin.Context, req *entity.ListAvailableInterfacesRequest) (*entity.ListAvailableInterfacesResponse, error) {
	interfaces, err := a.bridgeService.ListAvailableInterfaces(ctx.Request.Context(), req.NodeName)
	if err != nil {
		return nil, err
	}

	return &entity.ListAvailableInterfacesResponse{
		Interfaces: interfaces,
	}, nil
}
