package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
)

// NodeServiceInterface 定义节点服务的接口
type NodeServiceInterface interface {
	ListNodes(ctx context.Context) ([]*entity.Node, error)
	DescribeNode(ctx context.Context, nodeName string) (*entity.Node, error)
	DescribeNodeSummary(ctx context.Context, nodeName string) (*entity.NodeSummary, error)
	DescribeNodePCI(ctx context.Context, nodeName string) ([]entity.PCIDevice, error)
	DescribeNodeUSB(ctx context.Context, nodeName string) ([]entity.USBDevice, error)
	DescribeNodeNet(ctx context.Context, nodeName string) (*service.NodeNetworkInfo, error)
	DescribeNodeDisks(ctx context.Context, nodeName string) ([]entity.Disk, error)
	EnableNode(ctx context.Context, nodeName string) error
	DisableNode(ctx context.Context, nodeName string) error
}

// NodeAPI 节点 API
type NodeAPI struct {
	nodeService NodeServiceInterface
}

// NewNodeAPI 创建节点 API
func NewNodeAPI(nodeService *service.NodeService) *NodeAPI {
	return &NodeAPI{
		nodeService: nodeService,
	}
}

// RegisterRoutes 注册路由
func (a *NodeAPI) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/list-nodes", ginx.Adapt5(a.ListNodes))
	r.POST("/describe-node", ginx.Adapt5(a.DescribeNode))
	r.POST("/describe-node-summary", ginx.Adapt5(a.DescribeNodeSummary))
	r.POST("/describe-node-pci", ginx.Adapt5(a.DescribeNodePCI))
	r.POST("/describe-node-usb", ginx.Adapt5(a.DescribeNodeUSB))
	r.POST("/describe-node-net", ginx.Adapt5(a.DescribeNodeNet))
	r.POST("/describe-node-disks", ginx.Adapt5(a.DescribeNodeDisks))
	r.POST("/enable-node", ginx.Adapt5(a.EnableNode))
	r.POST("/disable-node", ginx.Adapt5(a.DisableNode))
}

// ListNodesRequest 列举节点请求
type ListNodesRequest struct {
	State string `json:"state"` // 状态过滤（可选）
	Type  string `json:"type"`  // 类型过滤（可选）
	Name  string `json:"name"`  // 名称过滤（可选）
}

// ListNodesResponse 列举节点响应
type ListNodesResponse struct {
	Nodes []*entity.Node `json:"nodes"`
}

// ListNodes 列举节点
func (a *NodeAPI) ListNodes(ctx *gin.Context, req *ListNodesRequest) (*ListNodesResponse, error) {
	nodes, err := a.nodeService.ListNodes(ctx.Request.Context())
	if err != nil {
		return nil, err
	}

	// TODO: 根据请求参数过滤节点
	// 当前简化处理，返回所有节点

	return &ListNodesResponse{Nodes: nodes}, nil
}

// DescribeNodeRequest 查询节点详情请求
type DescribeNodeRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DescribeNode 查询节点详情
func (a *NodeAPI) DescribeNode(ctx *gin.Context, req *DescribeNodeRequest) (*entity.Node, error) {
	node, err := a.nodeService.DescribeNode(ctx.Request.Context(), req.Name)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// DescribeNodeSummaryRequest 查询节点概要信息请求
type DescribeNodeSummaryRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DescribeNodeSummary 查询节点概要信息
func (a *NodeAPI) DescribeNodeSummary(ctx *gin.Context, req *DescribeNodeSummaryRequest) (*entity.NodeSummary, error) {
	summary, err := a.nodeService.DescribeNodeSummary(ctx.Request.Context(), req.Name)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// DescribeNodePCIRequest 查询节点 PCI 设备请求
type DescribeNodePCIRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DescribeNodePCIResponse 查询节点 PCI 设备响应
type DescribeNodePCIResponse struct {
	Devices []entity.PCIDevice `json:"devices"`
}

// DescribeNodePCI 查询节点 PCI 设备
func (a *NodeAPI) DescribeNodePCI(ctx *gin.Context, req *DescribeNodePCIRequest) (*DescribeNodePCIResponse, error) {
	devices, err := a.nodeService.DescribeNodePCI(ctx.Request.Context(), req.Name)
	if err != nil {
		return nil, err
	}

	return &DescribeNodePCIResponse{Devices: devices}, nil
}

// DescribeNodeUSBRequest 查询节点 USB 设备请求
type DescribeNodeUSBRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DescribeNodeUSBResponse 查询节点 USB 设备响应
type DescribeNodeUSBResponse struct {
	Devices []entity.USBDevice `json:"devices"`
}

// DescribeNodeUSB 查询节点 USB 设备
func (a *NodeAPI) DescribeNodeUSB(ctx *gin.Context, req *DescribeNodeUSBRequest) (*DescribeNodeUSBResponse, error) {
	devices, err := a.nodeService.DescribeNodeUSB(ctx.Request.Context(), req.Name)
	if err != nil {
		return nil, err
	}

	return &DescribeNodeUSBResponse{Devices: devices}, nil
}

// DescribeNodeNetRequest 查询节点网络接口请求
type DescribeNodeNetRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DescribeNodeNet 查询节点网络接口
func (a *NodeAPI) DescribeNodeNet(ctx *gin.Context, req *DescribeNodeNetRequest) (*service.NodeNetworkInfo, error) {
	netInfo, err := a.nodeService.DescribeNodeNet(ctx.Request.Context(), req.Name)
	if err != nil {
		return nil, err
	}

	return netInfo, nil
}

// DescribeNodeDisksRequest 查询节点物理磁盘请求
type DescribeNodeDisksRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DescribeNodeDisksResponse 查询节点物理磁盘响应
type DescribeNodeDisksResponse struct {
	Disks []entity.Disk `json:"disks"`
}

// DescribeNodeDisks 查询节点物理磁盘
func (a *NodeAPI) DescribeNodeDisks(ctx *gin.Context, req *DescribeNodeDisksRequest) (*DescribeNodeDisksResponse, error) {
	disks, err := a.nodeService.DescribeNodeDisks(ctx.Request.Context(), req.Name)
	if err != nil {
		return nil, err
	}

	return &DescribeNodeDisksResponse{Disks: disks}, nil
}

// EnableNodeRequest 启用节点请求
type EnableNodeRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// EnableNodeResponse 启用节点响应
type EnableNodeResponse struct {
	Message string `json:"message"`
}

// EnableNode 启用节点
func (a *NodeAPI) EnableNode(ctx *gin.Context, req *EnableNodeRequest) (*EnableNodeResponse, error) {
	if err := a.nodeService.EnableNode(ctx.Request.Context(), req.Name); err != nil {
		return nil, err
	}

	return &EnableNodeResponse{Message: "node enabled successfully"}, nil
}

// DisableNodeRequest 禁用节点请求
type DisableNodeRequest struct {
	Name string `json:"name" binding:"required"` // 节点名称
}

// DisableNodeResponse 禁用节点响应
type DisableNodeResponse struct {
	Message string `json:"message"`
}

// DisableNode 禁用节点
func (a *NodeAPI) DisableNode(ctx *gin.Context, req *DisableNodeRequest) (*DisableNodeResponse, error) {
	if err := a.nodeService.DisableNode(ctx.Request.Context(), req.Name); err != nil {
		return nil, err
	}

	return &DisableNodeResponse{Message: "node disabled successfully"}, nil
}
