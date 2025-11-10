// Package service 提供业务逻辑层的服务实现
// 包括 Storage Service、Image Service 和 Instance Service
package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/rs/zerolog"
)

// InstanceService 实例服务，管理虚拟机实例
type InstanceService struct {
	storageService *StorageService
	imageService   *ImageService
	libvirtClient  *libvirt.Client
	idGen          *idgen.Generator
}

// NewInstanceService 创建新的 Instance Service
func NewInstanceService(
	storageService *StorageService,
	imageService *ImageService,
	libvirtClient *libvirt.Client,
) (*InstanceService, error) {
	return &InstanceService{
		storageService: storageService,
		imageService:   imageService,
		libvirtClient:  libvirtClient,
		idGen:          idgen.New(),
	}, nil
}

// RunInstance 创建并启动实例
func (s *InstanceService) RunInstance(ctx context.Context, req *entity.RunInstanceRequest) (*entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Creating instance")

	// 1. 生成 Instance ID
	instanceID, err := s.idGen.GenerateInstanceID()
	if err != nil {
		return nil, fmt.Errorf("generate instance ID: %w", err)
	}

	// 2. 确定镜像（默认使用 ubuntu-jammy）
	imageID := req.ImageID
	if imageID == "" {
		imageID = "ubuntu-jammy"
	}

	// 获取镜像信息
	image, err := s.imageService.GetImage(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}

	logger.Info().
		Str("instance_id", instanceID).
		Str("image_id", imageID).
		Msg("Using image for instance")

	// 3. 确定实例配置（写死参数）
	sizeGB := req.SizeGB
	if sizeGB == 0 {
		sizeGB = 20 // 默认 20GB
	}

	memoryMB := req.MemoryMB
	if memoryMB == 0 {
		memoryMB = 2048 // 默认 2GB
	}

	vcpus := req.VCPUs
	if vcpus == 0 {
		vcpus = 2 // 默认 2 核
	}

	// 4. 从镜像创建 Volume
	volumeReq := &entity.CreateVolumeFromImageRequest{
		ImageID:  imageID,
		VolumeID: "", // 自动生成
		SizeGB:   sizeGB,
	}

	volume, err := s.storageService.CreateVolumeFromImage(ctx, volumeReq, image.Path, image.SizeGB)
	if err != nil {
		return nil, fmt.Errorf("create volume from image: %w", err)
	}

	logger.Info().
		Str("volume_id", volume.ID).
		Str("volume_path", volume.Path).
		Msg("Volume created from image")

	// 5. 创建 Libvirt Domain
	domainConfig := &libvirt.CreateVMConfig{
		Name:     instanceID,      // 使用 instance ID 作为 domain 名称
		Memory:   memoryMB * 1024, // 转换为 KB
		VCPUs:    vcpus,
		DiskPath: volume.Path,
		// 使用默认值
		DiskBus:       "virtio",
		NetworkType:   "bridge",
		NetworkSource: "br0",
		OSType:        "hvm",
		Architecture:  "x86_64",
		Autostart:     false,
	}

	domain, err := s.libvirtClient.CreateDomain(domainConfig, true) // true = 立即启动
	if err != nil {
		// 清理已创建的 volume
		_ = s.storageService.DeleteVolume(ctx, volume.ID)
		return nil, fmt.Errorf("create libvirt domain: %w", err)
	}

	logger.Info().
		Str("domain_name", domain.Name).
		Msg("Libvirt domain created and started")

	// 6. 格式化 Domain UUID
	domainUUIDStr := formatDomainUUID(domain.UUID)

	// 7. 构建 Instance 对象
	instance := &entity.Instance{
		ID:         instanceID,
		Name:       instanceID, // 默认使用 ID 作为名称
		State:      "running",  // 刚创建并启动，状态为 running
		ImageID:    imageID,
		VolumeID:   volume.ID,
		MemoryMB:   memoryMB,
		VCPUs:      vcpus,
		CreatedAt:  time.Now().Format(time.RFC3339),
		DomainUUID: domainUUIDStr,
		DomainName: domain.Name,
	}

	logger.Info().
		Str("instance_id", instanceID).
		Str("domain_name", domain.Name).
		Msg("Instance created successfully")

	return instance, nil
}

// formatDomainUUID 格式化 Domain UUID
func formatDomainUUID(uuid [16]byte) string {
	return hex.EncodeToString(uuid[:])
}
