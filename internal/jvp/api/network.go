package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
)

// NetworkServiceInterface 网络服务接口
type NetworkServiceInterface interface {
	ListNetworks(ctx context.Context, nodeName string) ([]entity.Network, error)
	DescribeNetwork(ctx context.Context, nodeName, networkName string) (*entity.Network, error)
	CreateNetwork(ctx context.Context, req *entity.CreateNetworkRequest) (*entity.Network, error)
	DeleteNetwork(ctx context.Context, nodeName, networkName string) error
	StartNetwork(ctx context.Context, nodeName, networkName string) (*entity.Network, error)
	StopNetwork(ctx context.Context, nodeName, networkName string) (*entity.Network, error)
	ListAvailableNetworkSources(ctx context.Context, nodeName string) (*entity.NetworkSources, error)
}

// NetworkAPI 网络 API
type NetworkAPI struct {
	networkService NetworkServiceInterface
}

// NewNetworkAPI 创建网络 API
func NewNetworkAPI(networkService *service.NetworkService) *NetworkAPI {
	return &NetworkAPI{
		networkService: networkService,
	}
}

// RegisterRoutes 注册路由 - Action 风格
func (a *NetworkAPI) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/list-networks", ginx.Adapt5(a.ListNetworks))
	r.POST("/describe-network", ginx.Adapt5(a.DescribeNetwork))
	r.POST("/create-network", ginx.Adapt5(a.CreateNetwork))
	r.POST("/delete-network", ginx.Adapt5(a.DeleteNetwork))
	r.POST("/start-network", ginx.Adapt5(a.StartNetwork))
	r.POST("/stop-network", ginx.Adapt5(a.StopNetwork))
	r.POST("/list-network-sources", ginx.Adapt5(a.ListNetworkSources))
}

// ListNetworks 列举网络
func (a *NetworkAPI) ListNetworks(ctx *gin.Context, req *entity.ListNetworksRequest) (*entity.ListNetworksResponse, error) {
	networks, err := a.networkService.ListNetworks(ctx.Request.Context(), req.NodeName)
	if err != nil {
		return nil, err
	}

	return &entity.ListNetworksResponse{
		Networks: networks,
	}, nil
}

// DescribeNetwork 查询网络详情
func (a *NetworkAPI) DescribeNetwork(ctx *gin.Context, req *entity.DescribeNetworkRequest) (*entity.DescribeNetworkResponse, error) {
	network, err := a.networkService.DescribeNetwork(ctx.Request.Context(), req.NodeName, req.NetworkName)
	if err != nil {
		return nil, err
	}

	return &entity.DescribeNetworkResponse{
		Network: network,
	}, nil
}

// CreateNetwork 创建网络
func (a *NetworkAPI) CreateNetwork(ctx *gin.Context, req *entity.CreateNetworkRequest) (*entity.CreateNetworkResponse, error) {
	network, err := a.networkService.CreateNetwork(ctx.Request.Context(), req)
	if err != nil {
		return nil, err
	}

	return &entity.CreateNetworkResponse{
		Network: network,
	}, nil
}

// DeleteNetwork 删除网络
func (a *NetworkAPI) DeleteNetwork(ctx *gin.Context, req *entity.DeleteNetworkRequest) (*entity.DeleteNetworkResponse, error) {
	if err := a.networkService.DeleteNetwork(ctx.Request.Context(), req.NodeName, req.NetworkName); err != nil {
		return nil, err
	}

	return &entity.DeleteNetworkResponse{
		Message: "Network deleted successfully",
	}, nil
}

// StartNetwork 启动网络
func (a *NetworkAPI) StartNetwork(ctx *gin.Context, req *entity.StartNetworkRequest) (*entity.StartNetworkResponse, error) {
	network, err := a.networkService.StartNetwork(ctx.Request.Context(), req.NodeName, req.NetworkName)
	if err != nil {
		return nil, err
	}

	return &entity.StartNetworkResponse{
		Network: network,
	}, nil
}

// StopNetwork 停止网络
func (a *NetworkAPI) StopNetwork(ctx *gin.Context, req *entity.StopNetworkRequest) (*entity.StopNetworkResponse, error) {
	network, err := a.networkService.StopNetwork(ctx.Request.Context(), req.NodeName, req.NetworkName)
	if err != nil {
		return nil, err
	}

	return &entity.StopNetworkResponse{
		Network: network,
	}, nil
}

// ListNetworkSources 列举可用网络源
func (a *NetworkAPI) ListNetworkSources(ctx *gin.Context, req *entity.ListNetworkSourcesRequest) (*entity.ListNetworkSourcesResponse, error) {
	sources, err := a.networkService.ListAvailableNetworkSources(ctx.Request.Context(), req.NodeName)
	if err != nil {
		return nil, err
	}

	return &entity.ListNetworkSourcesResponse{
		Sources: sources,
	}, nil
}
