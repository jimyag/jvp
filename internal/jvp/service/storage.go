// Package service 提供业务逻辑层的服务实现
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/rs/zerolog"
)

// StorageService 存储服务，管理 libvirt storage pool 和 volume
type StorageService struct {
	libvirtClient libvirt.LibvirtClient
	idGen         *idgen.Generator
}

// NewStorageService 创建新的 Storage Service
func NewStorageService(
	libvirtClient libvirt.LibvirtClient,
) *StorageService {
	return &StorageService{
		libvirtClient: libvirtClient,
		idGen:         idgen.New(),
	}
}

// GetPool 获取存储池信息
func (s *StorageService) GetPool(ctx context.Context, poolName string) (*entity.StoragePool, error) {
	poolInfo, err := s.libvirtClient.GetStoragePool(poolName)
	if err != nil {
		return nil, fmt.Errorf("get storage pool %s: %w", poolName, err)
	}

	return &entity.StoragePool{
		Name:       poolInfo.Name,
		State:      poolInfo.State,
		Capacity:   poolInfo.CapacityB,
		Allocation: poolInfo.AllocationB,
		Available:  poolInfo.AvailableB,
		Path:       poolInfo.Path,
	}, nil
}

// CreateVolume 创建存储卷
func (s *StorageService) CreateVolume(ctx context.Context, req *entity.CreateInternalVolumeRequest) (*entity.Volume, error) {
	logger := zerolog.Ctx(ctx)

	// 如果没有提供 VolumeID，自动生成
	volumeID := req.VolumeID
	if volumeID == "" {
		var err error
		volumeID, err = s.idGen.GenerateVolumeID()
		if err != nil {
			return nil, fmt.Errorf("generate volume ID: %w", err)
		}
	}

	logger.Info().
		Str("pool_name", req.PoolName).
		Str("volume_id", volumeID).
		Uint64("size_gb", req.SizeGB).
		Msg("Creating volume")

	// 确定格式
	format := req.Format
	if format == "" {
		format = "qcow2"
	}

	// 创建 volume
	volumeName := volumeID + ".qcow2"
	volInfo, err := s.libvirtClient.CreateVolume(req.PoolName, volumeName, req.SizeGB, format)
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	logger.Info().
		Str("volume_id", volumeID).
		Str("path", volInfo.Path).
		Msg("Volume created successfully")

	return &entity.Volume{
		ID:          volumeID,
		Name:        volInfo.Name,
		Pool:        req.PoolName,
		Path:        volInfo.Path,
		CapacityB:   volInfo.CapacityB,
		AllocationB: volInfo.AllocationB,
		Format:      volInfo.Format,
	}, nil
}

// ListVolumes 列出存储池中的所有卷
func (s *StorageService) ListVolumes(ctx context.Context, poolName string) ([]*entity.Volume, error) {
	volInfos, err := s.libvirtClient.ListVolumes(poolName)
	if err != nil {
		return nil, fmt.Errorf("list volumes in pool %s: %w", poolName, err)
	}

	volumes := make([]*entity.Volume, 0, len(volInfos))
	for _, volInfo := range volInfos {
		// 跳过 _templates_ 目录本身和目录中的文件
		if volInfo.Name == TemplatesDirName || strings.Contains(volInfo.Path, "/"+TemplatesDirName+"/") {
			continue
		}

		// 从文件名提取 volume ID（去掉 .qcow2 后缀）
		volumeID := strings.TrimSuffix(volInfo.Name, ".qcow2")

		volumes = append(volumes, &entity.Volume{
			ID:          volumeID,
			Name:        volInfo.Name,
			Pool:        poolName,
			Path:        volInfo.Path,
			CapacityB:   volInfo.CapacityB,
			AllocationB: volInfo.AllocationB,
			Format:      volInfo.Format,
		})
	}

	return volumes, nil
}

// ListStoragePools 列出所有存储池
func (s *StorageService) ListStoragePools(ctx context.Context, includeVolumes bool) ([]entity.StoragePool, error) {
	logger := zerolog.Ctx(ctx)

	// 获取所有存储池
	poolInfos, err := s.libvirtClient.ListStoragePools()
	if err != nil {
		return nil, fmt.Errorf("list storage pools: %w", err)
	}

	pools := make([]entity.StoragePool, 0, len(poolInfos))
	for _, poolInfo := range poolInfos {
		volumeCount := 0

		// 如果需要包含卷列表，获取卷数量
		if includeVolumes {
			volInfos, err := s.libvirtClient.ListVolumes(poolInfo.Name)
			if err != nil {
				logger.Warn().
					Str("pool_name", poolInfo.Name).
					Err(err).
					Msg("Failed to list volumes in pool")
				// 继续处理其他池，不中断
			} else {
				// 过滤掉 _templates_ 目录本身和目录中的文件
				for _, volInfo := range volInfos {
					if volInfo.Name != TemplatesDirName && !strings.Contains(volInfo.Path, "/"+TemplatesDirName+"/") {
						volumeCount++
					}
				}
			}
		}

		pool := entity.StoragePool{
			Name:        poolInfo.Name,
			State:       poolInfo.State,
			Capacity:    poolInfo.CapacityB,
			Allocation:  poolInfo.AllocationB,
			Available:   poolInfo.AvailableB,
			Path:        poolInfo.Path,
			VolumeCount: volumeCount,
		}

		pools = append(pools, pool)
	}

	logger.Info().
		Int("pool_count", len(pools)).
		Bool("include_volumes", includeVolumes).
		Msg("Listed storage pools")

	return pools, nil
}

// GetStoragePool 获取单个存储池的详细信息
func (s *StorageService) GetStoragePool(ctx context.Context, poolName string, includeVolumes bool) (*entity.StoragePool, error) {
	logger := zerolog.Ctx(ctx)

	// 获取存储池信息
	poolInfo, err := s.libvirtClient.GetStoragePool(poolName)
	if err != nil {
		return nil, fmt.Errorf("get storage pool %s: %w", poolName, err)
	}

	volumeCount := 0

	// 如果需要包含卷列表，获取卷数量
	if includeVolumes {
		volInfos, err := s.libvirtClient.ListVolumes(poolName)
		if err != nil {
			return nil, fmt.Errorf("list volumes in pool %s: %w", poolName, err)
		}
		// 过滤掉 _templates_ 目录本身和目录中的文件
		for _, volInfo := range volInfos {
			if volInfo.Name != TemplatesDirName && !strings.Contains(volInfo.Path, "/"+TemplatesDirName+"/") {
				volumeCount++
			}
		}
	}

	pool := &entity.StoragePool{
		Name:        poolInfo.Name,
		State:       poolInfo.State,
		Capacity:    poolInfo.CapacityB,
		Allocation:  poolInfo.AllocationB,
		Available:   poolInfo.AvailableB,
		Path:        poolInfo.Path,
		VolumeCount: volumeCount,
	}

	logger.Info().
		Str("pool_name", poolName).
		Bool("include_volumes", includeVolumes).
		Msg("Retrieved storage pool")

	return pool, nil
}
