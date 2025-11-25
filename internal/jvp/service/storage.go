// Package service 提供业务逻辑层的服务实现
// 包括 Storage Service 和 Image Service，用于管理存储资源和镜像模板
package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// StorageService 存储服务，管理 libvirt storage pool 和 volume
type StorageService struct {
	libvirtClient libvirt.LibvirtClient
	qemuImgClient qemuimg.QemuImgClient
	idGen         *idgen.Generator

	// 默认 pool 配置
	defaultPoolName string
	defaultPoolPath string
	imagesPoolName  string
	imagesPoolPath  string
}

// NewStorageService 创建新的 Storage Service
func NewStorageService(
	libvirtClient libvirt.LibvirtClient,
	dataDir string,
) (*StorageService, error) {
	// 默认配置
	defaultPoolName := "default"
	defaultPoolPath := fmt.Sprintf("%s/images", dataDir)
	imagesPoolName := "images"
	imagesPoolPath := fmt.Sprintf("%s/images/images", dataDir)

	service := &StorageService{
		libvirtClient:   libvirtClient,
		qemuImgClient:   qemuimg.New(""), // 默认使用真实客户端
		idGen:           idgen.New(),
		defaultPoolName: defaultPoolName,
		defaultPoolPath: defaultPoolPath,
		imagesPoolName:  imagesPoolName,
		imagesPoolPath:  imagesPoolPath,
	}

	// 初始化时确保必需的 pool 存在
	ctx := context.Background()
	if err := service.EnsurePool(ctx, defaultPoolName); err != nil {
		return nil, fmt.Errorf("ensure default pool: %w", err)
	}
	if err := service.EnsurePool(ctx, imagesPoolName); err != nil {
		return nil, fmt.Errorf("ensure images pool: %w", err)
	}

	return service, nil
}

// EnsurePool 确保存储池存在，如果不存在则创建
func (s *StorageService) EnsurePool(ctx context.Context, poolName string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("pool_name", poolName).Msg("Ensuring storage pool exists")

	var poolPath string
	switch poolName {
	case s.defaultPoolName:
		poolPath = s.defaultPoolPath
	case s.imagesPoolName:
		poolPath = s.imagesPoolPath
	default:
		return fmt.Errorf("unknown pool name: %s", poolName)
	}

	err := s.libvirtClient.EnsureStoragePool(poolName, "dir", poolPath)
	if err != nil {
		return fmt.Errorf("ensure storage pool %s: %w", poolName, err)
	}

	logger.Info().Str("pool_name", poolName).Str("path", poolPath).Msg("Storage pool ensured")
	return nil
}

// GetPool 获取存储池信息
func (s *StorageService) GetPool(ctx context.Context, poolName string) (*entity.StoragePool, error) {
	poolInfo, err := s.libvirtClient.GetStoragePool(poolName)
	if err != nil {
		return nil, fmt.Errorf("get storage pool %s: %w", poolName, err)
	}

	return &entity.StoragePool{
		Name:        poolInfo.Name,
		State:       poolInfo.State,
		CapacityB:   poolInfo.CapacityB,
		AllocationB: poolInfo.AllocationB,
		AvailableB:  poolInfo.AvailableB,
		Path:        poolInfo.Path,
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

	// 确保 pool 存在
	if err := s.EnsurePool(ctx, req.PoolName); err != nil {
		return nil, err
	}

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

// CreateVolumeFromImage 从镜像创建存储卷
func (s *StorageService) CreateVolumeFromImage(
	ctx context.Context,
	req *entity.CreateVolumeFromImageRequest,
	imagePath string,
	imageSizeGB uint64,
) (*entity.Volume, error) {
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
		Str("image_id", req.ImageID).
		Str("volume_id", volumeID).
		Uint64("size_gb", req.SizeGB).
		Msg("Creating volume from image")

	// 确保 default pool 存在
	if err := s.EnsurePool(ctx, s.defaultPoolName); err != nil {
		return nil, err
	}

	// 在 default pool 中创建 volume（先创建空文件用于注册到 libvirt）
	volumeName := volumeID + ".qcow2"
	volInfo, err := s.libvirtClient.CreateVolume(s.defaultPoolName, volumeName, req.SizeGB, "qcow2")
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	// 从镜像克隆到 volume
	// 策略：如果镜像大小 <= 目标大小，使用 backing file（节省空间）
	if imageSizeGB <= req.SizeGB {
		// 删除 CreateVolume 创建的空文件，因为 CreateFromBackingFile 需要创建新文件
		// 注意：这里只删除文件，不删除 libvirt volume 定义，后续会重新创建文件
		if err := os.Remove(volInfo.Path); err != nil && !os.IsNotExist(err) {
			// 清理已创建的 volume
			_ = s.libvirtClient.DeleteVolume(s.defaultPoolName, volumeName)
			return nil, fmt.Errorf("remove empty volume file: %w", err)
		}

		// 使用 backing file 创建新文件（不会修改原始镜像文件）
		err = s.qemuImgClient.CreateFromBackingFile(ctx, "qcow2", "qcow2", imagePath, volInfo.Path)
		if err != nil {
			// 清理已创建的 volume
			_ = s.libvirtClient.DeleteVolume(s.defaultPoolName, volumeName)
			return nil, fmt.Errorf("create volume from backing file: %w", err)
		}

		// 如果需要调整大小
		if imageSizeGB < req.SizeGB {
			err = s.qemuImgClient.Resize(ctx, volInfo.Path, req.SizeGB)
			if err != nil {
				// 清理已创建的 volume
				_ = s.libvirtClient.DeleteVolume(s.defaultPoolName, volumeName)
				return nil, fmt.Errorf("resize volume: %w", err)
			}
		}
	} else {
		// 完整复制（镜像太大，不能使用 backing file）
		err = s.qemuImgClient.Convert(ctx, "qcow2", "qcow2", imagePath, volInfo.Path)
		if err != nil {
			// 清理已创建的 volume
			_ = s.libvirtClient.DeleteVolume(s.defaultPoolName, volumeName)
			return nil, fmt.Errorf("convert image: %w", err)
		}

		// 调整大小
		err = s.qemuImgClient.Resize(ctx, volInfo.Path, req.SizeGB)
		if err != nil {
			// 清理已创建的 volume
			_ = s.libvirtClient.DeleteVolume(s.defaultPoolName, volumeName)
			return nil, fmt.Errorf("resize volume: %w", err)
		}
	}

	// 重新获取 volume 信息（因为大小可能已改变）
	volInfo, err = s.libvirtClient.GetVolume(s.defaultPoolName, volumeName)
	if err != nil {
		return nil, fmt.Errorf("get volume info: %w", err)
	}

	logger.Info().
		Str("volume_id", volumeID).
		Str("path", volInfo.Path).
		Msg("Volume created from image successfully")

	return &entity.Volume{
		ID:          volumeID,
		Name:        volInfo.Name,
		Pool:        s.defaultPoolName,
		Path:        volInfo.Path,
		CapacityB:   volInfo.CapacityB,
		AllocationB: volInfo.AllocationB,
		Format:      volInfo.Format,
	}, nil
}

// DeleteVolume 删除存储卷
func (s *StorageService) DeleteVolume(ctx context.Context, volumeID string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("volume_id", volumeID).Msg("Deleting volume")

	// 查找 volume 所在的 pool（先尝试 default pool）
	volumeName := volumeID + ".qcow2"
	err := s.libvirtClient.DeleteVolume(s.defaultPoolName, volumeName)
	if err != nil {
		// 尝试 images pool
		err = s.libvirtClient.DeleteVolume(s.imagesPoolName, volumeName)
		if err != nil {
			return fmt.Errorf("delete volume %s: %w", volumeID, err)
		}
	}

	logger.Info().Str("volume_id", volumeID).Msg("Volume deleted successfully")
	return nil
}

// GetVolume 获取存储卷信息
func (s *StorageService) GetVolume(ctx context.Context, volumeID string) (*entity.Volume, error) {
	volumeName := volumeID + ".qcow2"

	// 先尝试在 default pool 中查找
	volInfo, err := s.libvirtClient.GetVolume(s.defaultPoolName, volumeName)
	if err != nil {
		// 尝试 images pool
		volInfo, err = s.libvirtClient.GetVolume(s.imagesPoolName, volumeName)
		if err != nil {
			return nil, fmt.Errorf("get volume %s: %w", volumeID, err)
		}
		return &entity.Volume{
			ID:          volumeID,
			Name:        volInfo.Name,
			Pool:        s.imagesPoolName,
			Path:        volInfo.Path,
			CapacityB:   volInfo.CapacityB,
			AllocationB: volInfo.AllocationB,
			Format:      volInfo.Format,
		}, nil
	}

	return &entity.Volume{
		ID:          volumeID,
		Name:        volInfo.Name,
		Pool:        s.defaultPoolName,
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

// determineVolumeType 根据卷的名称、格式和池名称判断卷类型
// 参考 Flint 的实现逻辑:
// - ISO: format 为 iso 的安装镜像文件
// - Template: 存储在 image/template 池中的云镜像,或文件名包含云镜像关键词
// - Disk: 其他普通虚拟机磁盘文件
func determineVolumeType(volumeName, format, poolName string) string {
	// ISO 格式直接返回 iso
	if format == "iso" {
		return "iso"
	}

	// 根据池名称判断 - image/template 池中的文件都是模板
	poolNameLower := strings.ToLower(poolName)
	if strings.Contains(poolNameLower, "image") || strings.Contains(poolNameLower, "template") {
		return "template"
	}

	// 根据文件名判断 - 云镜像通常包含这些关键词
	volumeNameLower := strings.ToLower(volumeName)
	templateKeywords := []string{"cloudimg", "cloud-init", "cloud-base", "genericcloud"}
	for _, keyword := range templateKeywords {
		if strings.Contains(volumeNameLower, keyword) {
			return "template"
		}
	}

	// 默认为普通磁盘
	return "disk"
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
		pool := entity.StoragePool{
			Name:        poolInfo.Name,
			State:       poolInfo.State,
			CapacityB:   poolInfo.CapacityB,
			AllocationB: poolInfo.AllocationB,
			AvailableB:  poolInfo.AvailableB,
			Path:        poolInfo.Path,
		}

		// 如果需要包含卷列表
		if includeVolumes {
			volInfos, err := s.libvirtClient.ListVolumes(poolInfo.Name)
			if err != nil {
				logger.Warn().
					Str("pool_name", poolInfo.Name).
					Err(err).
					Msg("Failed to list volumes in pool")
				// 继续处理其他池，不中断
				pool.Volumes = []entity.Volume{}
			} else {
				volumes := make([]entity.Volume, 0, len(volInfos))
				for _, volInfo := range volInfos {
					// 从文件名提取 volume ID（去掉 .qcow2 后缀）
					volumeID := strings.TrimSuffix(volInfo.Name, ".qcow2")
					volumeType := determineVolumeType(volInfo.Name, volInfo.Format, poolInfo.Name)

					volumes = append(volumes, entity.Volume{
						ID:          volumeID,
						Name:        volInfo.Name,
						Pool:        poolInfo.Name,
						Path:        volInfo.Path,
						CapacityB:   volInfo.CapacityB,
						AllocationB: volInfo.AllocationB,
						Format:      volInfo.Format,
						VolumeType:  volumeType,
					})
				}
				pool.Volumes = volumes
				pool.VolumeCount = len(volumes)
			}
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

	pool := &entity.StoragePool{
		Name:        poolInfo.Name,
		State:       poolInfo.State,
		CapacityB:   poolInfo.CapacityB,
		AllocationB: poolInfo.AllocationB,
		AvailableB:  poolInfo.AvailableB,
		Path:        poolInfo.Path,
	}

	// 如果需要包含卷列表
	if includeVolumes {
		volInfos, err := s.libvirtClient.ListVolumes(poolName)
		if err != nil {
			return nil, fmt.Errorf("list volumes in pool %s: %w", poolName, err)
		}

		volumes := make([]entity.Volume, 0, len(volInfos))
		for _, volInfo := range volInfos {
			// 从文件名提取 volume ID（去掉 .qcow2 后缀）
			volumeID := strings.TrimSuffix(volInfo.Name, ".qcow2")

			volumes = append(volumes, entity.Volume{
				ID:          volumeID,
				Name:        volInfo.Name,
				Pool:        poolName,
				Path:        volInfo.Path,
				CapacityB:   volInfo.CapacityB,
				AllocationB: volInfo.AllocationB,
				Format:      volInfo.Format,
			})
		}
		pool.Volumes = volumes
		pool.VolumeCount = len(volumes)
	}

	logger.Info().
		Str("pool_name", poolName).
		Bool("include_volumes", includeVolumes).
		Msg("Retrieved storage pool")

	return pool, nil
}
