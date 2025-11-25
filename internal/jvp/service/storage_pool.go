package service

import (
	"context"
	"fmt"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
)

// StoragePoolService 存储池服务
// 纯粹调用 libvirt API，不存储额外数据
type StoragePoolService struct {
	nodeStorage *NodeStorage
}

// NewStoragePoolService 创建存储池服务
func NewStoragePoolService(nodeStorage *NodeStorage) *StoragePoolService {
	return &StoragePoolService{
		nodeStorage: nodeStorage,
	}
}

// getLibvirtClient 获取 libvirt 客户端（本地或远程节点）
func (s *StoragePoolService) getLibvirtClient(nodeName string) (*libvirt.Client, error) {
	if nodeName == "" {
		// 本地节点
		return libvirt.New()
	}
	// 远程节点
	return s.nodeStorage.GetConnection(nodeName)
}

// ListStoragePools 列举存储池
func (s *StoragePoolService) ListStoragePools(ctx context.Context, nodeName string) ([]entity.StoragePool, error) {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 调用 libvirt API 列举存储池
	poolInfos, err := client.ListStoragePools()
	if err != nil {
		return nil, fmt.Errorf("list storage pools: %w", err)
	}

	// 转换为实体
	pools := make([]entity.StoragePool, 0, len(poolInfos))
	for _, poolInfo := range poolInfos {
		pools = append(pools, entity.StoragePool{
			Name:       poolInfo.Name,
			UUID:       "", // libvirt.StoragePoolInfo doesn't provide UUID
			State:      poolInfo.State,
			Type:       "", // libvirt.StoragePoolInfo doesn't provide Type
			Capacity:   poolInfo.CapacityB,
			Allocation: poolInfo.AllocationB,
			Available:  poolInfo.AvailableB,
			Path:       poolInfo.Path,
		})
	}

	return pools, nil
}

// DescribeStoragePool 查询存储池详情
func (s *StoragePoolService) DescribeStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error) {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 调用 libvirt API 获取存储池信息
	poolInfo, err := client.GetStoragePool(poolName)
	if err != nil {
		return nil, fmt.Errorf("get storage pool %s: %w", poolName, err)
	}

	// 获取存储池中的卷数量
	volumes, err := client.ListVolumes(poolName)
	volumeCount := 0
	if err == nil {
		volumeCount = len(volumes)
	}

	return &entity.StoragePool{
		Name:        poolInfo.Name,
		UUID:        "", // libvirt.StoragePoolInfo doesn't provide UUID
		State:       poolInfo.State,
		Type:        "", // libvirt.StoragePoolInfo doesn't provide Type
		Capacity:    poolInfo.CapacityB,
		Allocation:  poolInfo.AllocationB,
		Available:   poolInfo.AvailableB,
		Path:        poolInfo.Path,
		VolumeCount: volumeCount,
	}, nil
}

// CreateStoragePool 创建存储池
func (s *StoragePoolService) CreateStoragePool(ctx context.Context, nodeName, name, poolType, path string) (*entity.StoragePool, error) {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 默认类型为 dir
	if poolType == "" {
		poolType = "dir"
	}

	// 调用 libvirt API 创建存储池
	if err := client.EnsureStoragePool(name, poolType, path); err != nil {
		return nil, fmt.Errorf("create storage pool %s: %w", name, err)
	}

	// 获取创建后的存储池信息
	return s.DescribeStoragePool(ctx, nodeName, name)
}

// DeleteStoragePool 删除存储池
func (s *StoragePoolService) DeleteStoragePool(ctx context.Context, nodeName, poolName string, deleteVolumes bool) error {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return fmt.Errorf("get libvirt client: %w", err)
	}

	// 调用 libvirt API 删除存储池
	if err := client.DeleteStoragePool(poolName, deleteVolumes); err != nil {
		return fmt.Errorf("delete storage pool %s: %w", poolName, err)
	}

	return nil
}

// StartStoragePool 启动存储池
func (s *StoragePoolService) StartStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error) {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 调用 libvirt API 启动存储池
	if err := client.StartStoragePool(poolName); err != nil {
		return nil, fmt.Errorf("start storage pool %s: %w", poolName, err)
	}

	// 返回更新后的存储池信息
	return s.DescribeStoragePool(ctx, nodeName, poolName)
}

// StopStoragePool 停止存储池
func (s *StoragePoolService) StopStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error) {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 调用 libvirt API 停止存储池
	if err := client.StopStoragePool(poolName); err != nil {
		return nil, fmt.Errorf("stop storage pool %s: %w", poolName, err)
	}

	// 返回更新后的存储池信息
	return s.DescribeStoragePool(ctx, nodeName, poolName)
}

// RefreshStoragePool 刷新存储池
func (s *StoragePoolService) RefreshStoragePool(ctx context.Context, nodeName, poolName string) (*entity.StoragePool, error) {
	// 获取 libvirt 客户端
	client, err := s.getLibvirtClient(nodeName)
	if err != nil {
		return nil, fmt.Errorf("get libvirt client: %w", err)
	}

	// 调用 libvirt API 刷新存储池
	if err := client.RefreshStoragePool(poolName); err != nil {
		return nil, fmt.Errorf("refresh storage pool %s: %w", poolName, err)
	}

	// 返回更新后的存储池信息
	return s.DescribeStoragePool(ctx, nodeName, poolName)
}
