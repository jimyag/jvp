package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// VolumeService 存储卷服务
type VolumeService struct {
	nodeService        *NodeService
	storagePoolService *StoragePoolService
	qemuImgClient      qemuimg.QemuImgClient
	idGen              *idgen.Generator
}

// NewVolumeService 创建新的 Volume Service
func NewVolumeService(
	nodeService *NodeService,
	storagePoolService *StoragePoolService,
) *VolumeService {
	return &VolumeService{
		nodeService:        nodeService,
		storagePoolService: storagePoolService,
		qemuImgClient:      qemuimg.New(""),
		idGen:              idgen.New(),
	}
}

// CreateVolume 创建存储卷
func (s *VolumeService) CreateVolume(ctx context.Context, req *entity.CreateVolumeRequest) (*entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("requested_name", req.Name).
		Uint64("size_gb", req.SizeGB).
		Msg("Creating volume")

	// 生成 Volume ID
	volumeID, err := s.idGen.GenerateVolumeID()
	if err != nil {
		return nil, fmt.Errorf("generate volume ID: %w", err)
	}

	// 确定卷名称
	volumeName := req.Name
	if volumeName == "" {
		// 如果没有提供名称，使用 volumeID
		volumeName = volumeID
	}

	logger.Info().
		Str("volume_id", volumeID).
		Str("volume_name", volumeName).
		Msg("Generated volume ID and name")

	// 确定格式
	format := req.Format
	if format == "" {
		format = "qcow2"
	}

	// 根据格式添加扩展名
	fileExtension := ".qcow2"
	if format == "raw" {
		fileExtension = ".raw"
	}
	fileName := volumeName + fileExtension

	// 获取节点的存储服务
	nodeStorage, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get node storage: %w", err)
	}

	// 创建存储卷
	volInfo, err := nodeStorage.CreateVolume(req.PoolName, fileName, req.SizeGB, format)
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	// 构建返回的 Volume 对象
	volume := &entity.Volume{
		ID:          volumeID,
		Name:        volInfo.Name,
		NodeName:    req.NodeName,
		Pool:        req.PoolName,
		Path:        volInfo.Path,
		CapacityB:   volInfo.CapacityB,
		SizeGB:      req.SizeGB,
		AllocationB: volInfo.AllocationB,
		Format:      volInfo.Format,
	}

	logger.Info().
		Str("volume_id", volumeID).
		Str("path", volInfo.Path).
		Msg("Volume created successfully")

	return volume, nil
}

// ListVolumes 列举存储池中的所有卷
func (s *VolumeService) ListVolumes(ctx context.Context, req *entity.ListVolumesRequest) ([]entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Msg("Listing volumes in pool")

	// 获取节点的存储服务
	nodeStorage, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get node storage: %w", err)
	}

	// 列举卷
	volInfos, err := nodeStorage.ListVolumes(req.PoolName)
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}

	volumes := make([]entity.Volume, 0, len(volInfos))
	for _, volInfo := range volInfos {
		// 跳过 _templates_ 目录本身和目录中的文件
		if volInfo.Name == TemplatesDirName || strings.Contains(volInfo.Path, "/"+TemplatesDirName+"/") {
			continue
		}

		// 从文件名提取 volume ID (去掉扩展名)
		volumeID := strings.TrimSuffix(volInfo.Name, ".qcow2")
		volumeID = strings.TrimSuffix(volumeID, ".raw")
		volumeID = strings.TrimSuffix(volumeID, ".iso")
		volumeID = strings.TrimSuffix(volumeID, ".img")

		volume := entity.Volume{
			ID:          volumeID,
			Name:        volInfo.Name,
			NodeName:    req.NodeName,
			Pool:        req.PoolName,
			Path:        volInfo.Path,
			CapacityB:   volInfo.CapacityB,
			SizeGB:      volInfo.CapacityB / (1024 * 1024 * 1024),
			AllocationB: volInfo.AllocationB,
			Format:      volInfo.Format,
		}
		volumes = append(volumes, volume)
	}

	logger.Info().
		Int("count", len(volumes)).
		Msg("Volumes listed successfully")

	return volumes, nil
}

// DescribeVolume 查询卷详情
func (s *VolumeService) DescribeVolume(ctx context.Context, req *entity.DescribeVolumeRequest) (*entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("volume_id", req.VolumeID).
		Msg("Describing volume")

	// 获取节点的存储服务
	nodeStorage, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get node storage: %w", err)
	}

	// 查询卷信息
	volumeName := req.VolumeID + ".qcow2"
	volInfo, err := nodeStorage.GetVolume(req.PoolName, volumeName)
	if err != nil {
		// 尝试其他扩展名
		volumeName = req.VolumeID + ".raw"
		volInfo, err = nodeStorage.GetVolume(req.PoolName, volumeName)
		if err != nil {
			volumeName = req.VolumeID + ".img"
			volInfo, err = nodeStorage.GetVolume(req.PoolName, volumeName)
			if err != nil {
				return nil, fmt.Errorf("get volume: %w", err)
			}
		}
	}

	volume := &entity.Volume{
		ID:          req.VolumeID,
		Name:        volInfo.Name,
		NodeName:    req.NodeName,
		Pool:        req.PoolName,
		Path:        volInfo.Path,
		CapacityB:   volInfo.CapacityB,
		SizeGB:      volInfo.CapacityB / (1024 * 1024 * 1024),
		AllocationB: volInfo.AllocationB,
		Format:      volInfo.Format,
	}

	logger.Info().
		Str("volume_id", req.VolumeID).
		Msg("Volume described successfully")

	return volume, nil
}

// ResizeVolume 扩容卷
func (s *VolumeService) ResizeVolume(ctx context.Context, req *entity.ResizeVolumeRequest) (*entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("volume_id", req.VolumeID).
		Uint64("new_size_gb", req.NewSizeGB).
		Msg("Resizing volume")

	// 先查询卷信息
	describeReq := &entity.DescribeVolumeRequest{
		NodeName: req.NodeName,
		PoolName: req.PoolName,
		VolumeID: req.VolumeID,
	}
	volume, err := s.DescribeVolume(ctx, describeReq)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	// 检查新大小是否大于当前大小
	currentSizeGB := volume.CapacityB / (1024 * 1024 * 1024)
	if req.NewSizeGB <= currentSizeGB {
		return nil, fmt.Errorf("new size (%d GB) must be larger than current size (%d GB)", req.NewSizeGB, currentSizeGB)
	}

	// 获取节点的存储服务
	nodeStorage, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get node storage: %w", err)
	}

	// 使用 libvirt API 调整大小
	err = nodeStorage.ResizeVolume(req.PoolName, volume.Name, req.NewSizeGB)
	if err != nil {
		return nil, fmt.Errorf("resize volume: %w", err)
	}

	// 重新查询卷信息
	updatedVolume, err := s.DescribeVolume(ctx, describeReq)
	if err != nil {
		return nil, fmt.Errorf("get updated volume: %w", err)
	}

	logger.Info().
		Str("volume_id", req.VolumeID).
		Uint64("new_size_gb", req.NewSizeGB).
		Msg("Volume resized successfully")

	return updatedVolume, nil
}

// DeleteVolume 删除存储卷
func (s *VolumeService) DeleteVolume(ctx context.Context, req *entity.DeleteVolumeRequest) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("volume_id", req.VolumeID).
		Msg("Deleting volume")

	// 获取节点的存储服务
	nodeStorage, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return fmt.Errorf("get node storage: %w", err)
	}

	// 删除卷
	volumeName := req.VolumeID + ".qcow2"
	err = nodeStorage.DeleteVolume(req.PoolName, volumeName)
	if err != nil {
		// 尝试其他扩展名
		volumeName = req.VolumeID + ".raw"
		err = nodeStorage.DeleteVolume(req.PoolName, volumeName)
		if err != nil {
			volumeName = req.VolumeID + ".img"
			err = nodeStorage.DeleteVolume(req.PoolName, volumeName)
			if err != nil {
				return fmt.Errorf("delete volume: %w", err)
			}
		}
	}

	logger.Info().
		Str("volume_id", req.VolumeID).
		Msg("Volume deleted successfully")

	return nil
}

// CreateVolumeFromURL 从 URL 下载并创建存储卷
func (s *VolumeService) CreateVolumeFromURL(ctx context.Context, req *entity.CreateVolumeFromURLRequest) (*entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Str("name", req.Name).
		Str("url", req.URL).
		Msg("Creating volume from URL")

	// 生成 Volume ID
	volumeID, err := s.idGen.GenerateVolumeID()
	if err != nil {
		return nil, fmt.Errorf("generate volume ID: %w", err)
	}

	// 获取节点的存储服务
	nodeStorage, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get node storage: %w", err)
	}

	// 下载文件到存储池
	if err := downloadToPool(nodeStorage, req.PoolName, req.Name, req.URL); err != nil {
		return nil, fmt.Errorf("download volume from URL: %w", err)
	}

	// 获取下载后的卷信息
	volInfo, err := nodeStorage.GetVolume(req.PoolName, req.Name)
	if err != nil {
		return nil, fmt.Errorf("get volume info: %w", err)
	}

	// 构建返回的 Volume 对象
	volume := &entity.Volume{
		ID:          volumeID,
		Name:        volInfo.Name,
		NodeName:    req.NodeName,
		Pool:        req.PoolName,
		Path:        volInfo.Path,
		CapacityB:   volInfo.CapacityB,
		SizeGB:      volInfo.CapacityB / (1024 * 1024 * 1024),
		AllocationB: volInfo.AllocationB,
		Format:      volInfo.Format,
	}

	logger.Info().
		Str("volume_id", volumeID).
		Str("path", volInfo.Path).
		Msg("Volume created from URL successfully")

	return volume, nil
}
