package service

import (
	"context"
	"fmt"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
)

// NetworkService 网络服务
// 管理 libvirt 虚拟网络
type NetworkService struct {
	nodeStorage   *NodeStorage
	bridgeService *BridgeService
}

// NewNetworkService 创建网络服务
func NewNetworkService(nodeStorage *NodeStorage, bridgeService *BridgeService) *NetworkService {
	return &NetworkService{
		nodeStorage:   nodeStorage,
		bridgeService: bridgeService,
	}
}

// getLibvirtClient 获取 libvirt 客户端（本地或远程节点）
func (s *NetworkService) getLibvirtClient(nodeName string) (*libvirt.Client, error) {
	if nodeName == "" {
		return libvirt.New()
	}
	return s.nodeStorage.GetConnection(nodeName)
}

// ListNetworks 列举所有 libvirt 网络
func (s *NetworkService) ListNetworks(ctx context.Context, nodeName string) ([]entity.Network, error) {
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	networkInfos, err := client.ListNetworksInfo()
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}

	networks := make([]entity.Network, 0, len(networkInfos))
	for _, info := range networkInfos {
		state := "inactive"
		if info.Active {
			state = "active"
		}
		networks = append(networks, entity.Network{
			Name:       info.Name,
			UUID:       info.UUID,
			NodeName:   nodeName,
			Type:       "libvirt",
			Mode:       info.Mode,
			Bridge:     info.Bridge,
			State:      state,
			Autostart:  info.Autostart,
			Persistent: info.Persistent,
			IPAddress:  info.IPAddress,
			Netmask:    info.Netmask,
			DHCPStart:  info.DHCPStart,
			DHCPEnd:    info.DHCPEnd,
		})
	}

	return networks, nil
}

// DescribeNetwork 查询网络详情
func (s *NetworkService) DescribeNetwork(ctx context.Context, nodeName, networkName string) (*entity.Network, error) {
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	info, err := client.GetNetwork(networkName)
	if err != nil {
		return nil, fmt.Errorf("get network %s: %w", networkName, err)
	}

	state := "inactive"
	if info.Active {
		state = "active"
	}

	return &entity.Network{
		Name:       info.Name,
		UUID:       info.UUID,
		NodeName:   nodeName,
		Type:       "libvirt",
		Mode:       info.Mode,
		Bridge:     info.Bridge,
		State:      state,
		Autostart:  info.Autostart,
		Persistent: info.Persistent,
		IPAddress:  info.IPAddress,
		Netmask:    info.Netmask,
		DHCPStart:  info.DHCPStart,
		DHCPEnd:    info.DHCPEnd,
	}, nil
}

// CreateNetwork 创建网络
func (s *NetworkService) CreateNetwork(ctx context.Context, req *entity.CreateNetworkRequest) (*entity.Network, error) {
	client, err := s.getLibvirtClient(req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 默认模式为 nat
	mode := req.Mode
	if mode == "" {
		mode = "nat"
	}

	config := libvirt.NetworkConfig{
		Name:      req.Name,
		Mode:      mode,
		IPAddress: req.IPAddress,
		Netmask:   req.Netmask,
		DHCPStart: req.DHCPStart,
		DHCPEnd:   req.DHCPEnd,
		Autostart: req.Autostart,
	}

	info, err := client.CreateNetwork(config)
	if err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}

	state := "inactive"
	if info.Active {
		state = "active"
	}

	return &entity.Network{
		Name:       info.Name,
		UUID:       info.UUID,
		NodeName:   req.NodeName,
		Type:       "libvirt",
		Mode:       info.Mode,
		Bridge:     info.Bridge,
		State:      state,
		Autostart:  info.Autostart,
		Persistent: info.Persistent,
		IPAddress:  info.IPAddress,
		Netmask:    info.Netmask,
		DHCPStart:  info.DHCPStart,
		DHCPEnd:    info.DHCPEnd,
	}, nil
}

// DeleteNetwork 删除网络
func (s *NetworkService) DeleteNetwork(ctx context.Context, nodeName, networkName string) error {
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return fmt.Errorf("get libvirt client: %w", err)
	}

	if err := client.DeleteNetwork(networkName); err != nil {
		return fmt.Errorf("delete network %s: %w", networkName, err)
	}

	return nil
}

// StartNetwork 启动网络
func (s *NetworkService) StartNetwork(ctx context.Context, nodeName, networkName string) (*entity.Network, error) {
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	if err := client.StartNetwork(networkName); err != nil {
		return nil, fmt.Errorf("start network %s: %w", networkName, err)
	}

	return s.DescribeNetwork(ctx, nodeName, networkName)
}

// StopNetwork 停止网络
func (s *NetworkService) StopNetwork(ctx context.Context, nodeName, networkName string) (*entity.Network, error) {
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	if err := client.StopNetwork(networkName); err != nil {
		return nil, fmt.Errorf("stop network %s: %w", networkName, err)
	}

	return s.DescribeNetwork(ctx, nodeName, networkName)
}

// ListAvailableNetworkSources 列举可用于创建 VM 的网络源
func (s *NetworkService) ListAvailableNetworkSources(ctx context.Context, nodeName string) (*entity.NetworkSources, error) {
	// 获取 libvirt 网络
	networks, err := s.ListNetworks(ctx, nodeName)
	if err != nil {
		// 网络获取失败不是致命错误
		networks = []entity.Network{}
	}

	// 获取宿主机网桥
	var bridges []entity.HostBridge
	if s.bridgeService != nil {
		bridges, err = s.bridgeService.ListBridges(ctx, nodeName)
		if err != nil {
			// 网桥获取失败不是致命错误
			bridges = []entity.HostBridge{}
		}
	}

	return &entity.NetworkSources{
		LibvirtNetworks: networks,
		HostBridges:     bridges,
	}, nil
}
