// Package service 提供业务逻辑层的服务实现
// 包括 Storage Service、Image Service 和 Instance Service
package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
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
	instanceRepo   repository.InstanceRepository
}

// NewInstanceService 创建新的 Instance Service
func NewInstanceService(
	storageService *StorageService,
	imageService *ImageService,
	libvirtClient *libvirt.Client,
	repo *repository.Repository,
) (*InstanceService, error) {
	return &InstanceService{
		storageService: storageService,
		imageService:   imageService,
		libvirtClient:  libvirtClient,
		idGen:          idgen.New(),
		instanceRepo:   repository.NewInstanceRepository(repo.DB()),
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

	// 8. 保存到数据库
	instanceModel, err := instanceEntityToModel(instance)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to convert instance to model, skipping database save")
	} else {
		if err := s.instanceRepo.Create(ctx, instanceModel); err != nil {
			logger.Warn().Err(err).Msg("Failed to save instance to database")
			// 不返回错误，因为实例已经创建成功
		} else {
			logger.Info().Str("instance_id", instanceID).Msg("Instance saved to database")
		}
	}

	return instance, nil
}

// formatDomainUUID 格式化 Domain UUID
func formatDomainUUID(uuid [16]byte) string {
	return hex.EncodeToString(uuid[:])
}

// DescribeInstances 描述实例
func (s *InstanceService) DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Describing instances")

	// 从数据库查询
	filters := make(map[string]interface{})
	if len(req.InstanceIDs) > 0 {
		// 如果指定了 InstanceIDs，逐个查询
		var instances []entity.Instance
		for _, instanceID := range req.InstanceIDs {
			instanceModel, err := s.instanceRepo.GetByID(ctx, instanceID)
			if err != nil {
				// 如果数据库中没有，尝试从 Libvirt 查询（兼容旧数据）
				instance, err := s.GetInstance(ctx, instanceID)
				if err != nil {
					continue
				}
				instances = append(instances, *instance)
			} else {
				instance, err := instanceModelToEntity(instanceModel)
				if err != nil {
					continue
				}
				instances = append(instances, *instance)
			}
		}
		return instances, nil
	}

	// 列出所有实例
	instanceModels, err := s.instanceRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("list instances from database: %w", err)
	}

	instances := make([]entity.Instance, 0, len(instanceModels))
	for _, instanceModel := range instanceModels {
		instance, err := instanceModelToEntity(instanceModel)
		if err != nil {
			logger.Warn().Err(err).Str("instance_id", instanceModel.ID).Msg("Failed to convert instance model to entity")
			continue
		}
		instances = append(instances, *instance)
	}

	// TODO: 应用过滤器

	return instances, nil
}

// GetInstance 获取单个实例信息
func (s *InstanceService) GetInstance(ctx context.Context, instanceID string) (*entity.Instance, error) {
	// 优先从数据库查询
	instanceModel, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err == nil {
		return instanceModelToEntity(instanceModel)
	}

	// 如果数据库中没有，从 Libvirt 查询（兼容旧数据）
	domain, err := s.libvirtClient.GetDomainByName(instanceID)
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}

	return s.domainToInstance(ctx, domain)
}

// domainToInstance 将 libvirt Domain 转换为 Instance
func (s *InstanceService) domainToInstance(ctx context.Context, domain libvirtlib.Domain) (*entity.Instance, error) {
	// 获取 domain 信息
	domainInfo, err := s.libvirtClient.GetDomainInfo(domain.UUID)
	if err != nil {
		return nil, fmt.Errorf("get domain info: %w", err)
	}

	// 转换状态
	instanceState := mapLibvirtStateToInstanceState(domainInfo.State)

	// 获取磁盘信息（用于获取 VolumeID）
	disks, err := s.libvirtClient.GetDomainDisks(domain.Name)
	if err != nil {
		// 如果获取磁盘失败，不影响基本信息
		disks = []libvirt.DomainDisk{}
	}

	var volumeID string
	if len(disks) > 0 && disks[0].Source.File != "" {
		// 从磁盘路径提取 volume ID（假设路径格式为 /var/lib/jvp/images/{volumeID}.qcow2）
		// TODO: 更可靠的方式是从存储服务查询
		volumeID = extractVolumeIDFromPath(disks[0].Source.File)
	}

	instance := &entity.Instance{
		ID:         domain.Name,
		Name:       domain.Name,
		State:      instanceState,
		ImageID:    "", // TODO: 从元数据或标签获取
		VolumeID:   volumeID,
		MemoryMB:   domainInfo.Memory / 1024, // 转换为 MB
		VCPUs:      domainInfo.VCPUs,
		CreatedAt:  "", // TODO: 从文件系统或元数据获取
		DomainUUID: formatDomainUUID(domain.UUID),
		DomainName: domain.Name,
	}

	return instance, nil
}

// mapLibvirtStateToInstanceState 将 libvirt 状态映射到实例状态
func mapLibvirtStateToInstanceState(libvirtState string) string {
	switch libvirtState {
	case "Running":
		return "running"
	case "ShutOff":
		return "stopped"
	case "ShuttingDown":
		return "stopping"
	case "Paused":
		return "paused"
	case "Crashed":
		return "failed"
	default:
		return "pending"
	}
}

// extractVolumeIDFromPath 从磁盘路径提取 volume ID
func extractVolumeIDFromPath(path string) string {
	// 假设路径格式为 /var/lib/jvp/images/{volumeID}.qcow2
	// 提取文件名并去掉 .qcow2 后缀
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		return strings.TrimSuffix(filename, ".qcow2")
	}
	return ""
}

// TerminateInstances 终止实例
func (s *InstanceService) TerminateInstances(ctx context.Context, req *entity.TerminateInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Terminating instances")

	var changes []entity.InstanceStateChange

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			// 如果实例不存在，跳过
			continue
		}
		previousState := instance.State

		// 获取 domain
		domain, err := s.libvirtClient.GetDomainByName(instanceID)
		if err != nil {
			logger.Warn().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain, skipping")
			continue
		}

		// 删除 domain（会先停止运行中的实例）
		err = s.libvirtClient.DeleteDomain(domain, libvirtlib.DomainUndefineFlagsValues(0))
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to delete domain")
			continue
		}

		// 删除关联的 volume（可选，根据配置决定）
		if instance.VolumeID != "" {
			_ = s.storageService.DeleteVolume(ctx, instance.VolumeID)
		}

		// 更新数据库状态（软删除）
		instanceModel, err := s.instanceRepo.GetByID(ctx, instanceID)
		if err == nil {
			instanceModel.State = "terminated"
			if err := s.instanceRepo.Update(ctx, instanceModel); err != nil {
				logger.Warn().Err(err).Str("instance_id", instanceID).Msg("Failed to update instance state in database")
			}
			// 软删除
			if err := s.instanceRepo.Delete(ctx, instanceID); err != nil {
				logger.Warn().Err(err).Str("instance_id", instanceID).Msg("Failed to delete instance from database")
			}
		}

		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "terminated",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Instance terminated successfully")
	}

	return changes, nil
}

// StopInstances 停止实例
func (s *InstanceService) StopInstances(ctx context.Context, req *entity.StopInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Bool("force", req.Force).
		Msg("Stopping instances")

	var changes []entity.InstanceStateChange

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			continue
		}
		previousState := instance.State

		if instance.State == "stopped" {
			// 已经停止，跳过
			changes = append(changes, entity.InstanceStateChange{
				InstanceID:    instanceID,
				CurrentState:  "stopped",
				PreviousState: previousState,
			})
			continue
		}

		// 获取 domain
		domain, err := s.libvirtClient.GetDomainByName(instanceID)
		if err != nil {
			continue
		}

		// 停止 domain
		if req.Force {
			// 强制停止
			err = s.libvirtClient.DestroyDomain(domain)
		} else {
			// 优雅停止
			err = s.libvirtClient.StopDomain(domain)
		}

		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to stop domain")
			continue
		}

		// 更新数据库状态
		instanceModel, err := s.instanceRepo.GetByID(ctx, instanceID)
		if err == nil {
			instanceModel.State = "stopped"
			if err := s.instanceRepo.Update(ctx, instanceModel); err != nil {
				logger.Warn().Err(err).Str("instance_id", instanceID).Msg("Failed to update instance state in database")
			}
		}

		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "stopped",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Instance stopped successfully")
	}

	return changes, nil
}

// StartInstances 启动实例
func (s *InstanceService) StartInstances(ctx context.Context, req *entity.StartInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Starting instances")

	var changes []entity.InstanceStateChange

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			continue
		}
		previousState := instance.State

		if instance.State == "running" {
			// 已经运行，跳过
			changes = append(changes, entity.InstanceStateChange{
				InstanceID:    instanceID,
				CurrentState:  "running",
				PreviousState: previousState,
			})
			continue
		}

		// 获取 domain
		domain, err := s.libvirtClient.GetDomainByName(instanceID)
		if err != nil {
			continue
		}

		// 启动 domain
		err = s.libvirtClient.StartDomain(domain)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to start domain")
			continue
		}

		// 更新数据库状态
		instanceModel, err := s.instanceRepo.GetByID(ctx, instanceID)
		if err == nil {
			instanceModel.State = "running"
			if err := s.instanceRepo.Update(ctx, instanceModel); err != nil {
				logger.Warn().Err(err).Str("instance_id", instanceID).Msg("Failed to update instance state in database")
			}
		}

		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "running",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Instance started successfully")
	}

	return changes, nil
}

// RebootInstances 重启实例
func (s *InstanceService) RebootInstances(ctx context.Context, req *entity.RebootInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Rebooting instances")

	var changes []entity.InstanceStateChange

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			continue
		}
		previousState := instance.State

		if instance.State == "stopped" {
			// 如果已停止，先启动
			_, err = s.StartInstances(ctx, &entity.StartInstancesRequest{
				InstanceIDs: []string{instanceID},
			})
			if err != nil {
				continue
			}
			previousState = "running"
		}

		// 获取 domain
		domain, err := s.libvirtClient.GetDomainByName(instanceID)
		if err != nil {
			continue
		}

		// 重启 domain
		err = s.libvirtClient.RebootDomain(domain)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to reboot domain")
			continue
		}

		// 更新数据库状态
		instanceModel, err := s.instanceRepo.GetByID(ctx, instanceID)
		if err == nil {
			instanceModel.State = "running" // 重启后状态为 running
			if err := s.instanceRepo.Update(ctx, instanceModel); err != nil {
				logger.Warn().Err(err).Str("instance_id", instanceID).Msg("Failed to update instance state in database")
			}
		}

		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "running",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Instance rebooted successfully")
	}

	return changes, nil
}

// ModifyInstanceAttribute 修改实例属性
func (s *InstanceService) ModifyInstanceAttribute(ctx context.Context, req *entity.ModifyInstanceAttributeRequest) (*entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instanceID", req.InstanceID).
		Interface("request", req).
		Msg("Modifying instance attribute")

	// 获取当前实例信息
	instance, err := s.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	// 获取 domain
	domain, err := s.libvirtClient.GetDomainByName(req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}

	// 修改内存
	if req.MemoryMB != nil {
		memoryKB := *req.MemoryMB * 1024
		err = s.libvirtClient.ModifyDomainMemory(domain, memoryKB, req.Live)
		if err != nil {
			return nil, fmt.Errorf("modify memory: %w", err)
		}
		instance.MemoryMB = *req.MemoryMB
		logger.Info().
			Str("instanceID", req.InstanceID).
			Uint64("memoryMB", *req.MemoryMB).
			Msg("Instance memory modified")
	}

	// 修改 VCPU
	if req.VCPUs != nil {
		err = s.libvirtClient.ModifyDomainVCPU(domain, *req.VCPUs, req.Live)
		if err != nil {
			return nil, fmt.Errorf("modify VCPU: %w", err)
		}
		instance.VCPUs = *req.VCPUs
		logger.Info().
			Str("instanceID", req.InstanceID).
			Uint16("vcpus", *req.VCPUs).
			Msg("Instance VCPU modified")
	}

	// 修改名称（TODO: 需要更新 domain XML 的 name 字段）
	if req.Name != nil {
		// TODO: 实现修改 domain 名称
		instance.Name = *req.Name
		logger.Info().
			Str("instanceID", req.InstanceID).
			Str("name", *req.Name).
			Msg("Instance name modified")
	}

	// 更新数据库
	instanceModel, err := s.instanceRepo.GetByID(ctx, req.InstanceID)
	if err == nil {
		// 更新修改的属性
		if req.MemoryMB != nil {
			instanceModel.MemoryMB = *req.MemoryMB
		}
		if req.VCPUs != nil {
			instanceModel.VCPUs = *req.VCPUs
		}
		if req.Name != nil {
			instanceModel.Name = *req.Name
		}
		if err := s.instanceRepo.Update(ctx, instanceModel); err != nil {
			logger.Warn().Err(err).Str("instance_id", req.InstanceID).Msg("Failed to update instance in database")
		}
	}

	// 重新获取实例信息以获取最新状态
	updatedInstance, err := s.GetInstance(ctx, req.InstanceID)
	if err != nil {
		// 如果获取失败，返回修改后的实例信息
		return instance, nil
	}

	logger.Info().
		Str("instanceID", req.InstanceID).
		Msg("Instance attribute modified successfully")

	return updatedInstance, nil
}
