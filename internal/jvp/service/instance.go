// Package service 提供业务逻辑层的服务实现
// 包括 Storage Service、Image Service 和 Instance Service
package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/cloudinit"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// InstanceService 实例服务，管理虚拟机实例
type InstanceService struct {
	storageService      *StorageService
	imageService        *ImageService
	keyPairService      *KeyPairService
	libvirtClient       libvirt.LibvirtClient
	virtCustomizeClient virtcustomize.VirtCustomizeClient
	idGen               *idgen.Generator
}

// NewInstanceService 创建新的 Instance Service
func NewInstanceService(
	storageService *StorageService,
	imageService *ImageService,
	keyPairService *KeyPairService,
	libvirtClient libvirt.LibvirtClient,
) (*InstanceService, error) {
	// 创建 virt-customize 客户端（如果失败，返回 nil，后续使用时再处理）
	virtCustomizeClient, _ := virtcustomize.NewClient()

	return &InstanceService{
		storageService:      storageService,
		imageService:        imageService,
		keyPairService:      keyPairService,
		libvirtClient:       libvirtClient,
		virtCustomizeClient: virtCustomizeClient,
		idGen:               idgen.New(),
	}, nil
}

// GetLibvirtClient 获取 libvirt 客户端（用于控制台访问）
func (s *InstanceService) GetLibvirtClient() libvirt.LibvirtClient {
	return s.libvirtClient
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

	// 处理密钥对（如果指定）
	if len(req.KeyPairIDs) > 0 && s.keyPairService != nil {
		var publicKeys []string
		for _, keyPairID := range req.KeyPairIDs {
			keyPair, err := s.keyPairService.GetKeyPairByID(ctx, keyPairID)
			if err != nil {
				// 清理已创建的 volume
				_ = s.storageService.DeleteVolume(ctx, volume.ID)
				return nil, fmt.Errorf("get keypair %s: %w", keyPairID, err)
			}
			publicKeys = append(publicKeys, keyPair.PublicKey)
		}

		// 将公钥注入到 UserData 配置中
		if req.UserData == nil {
			req.UserData = &entity.UserDataConfig{}
		}
		if req.UserData.StructuredUserData == nil {
			req.UserData.StructuredUserData = &entity.StructuredUserData{}
		}

		// 合并公钥到现有用户配置
		if len(req.UserData.StructuredUserData.Users) == 0 {
			// 如果没有用户配置，创建默认用户
			req.UserData.StructuredUserData.Users = []entity.User{
				{
					Name:              "ubuntu",
					SSHAuthorizedKeys: publicKeys,
				},
			}
		} else {
			// 合并到第一个用户
			req.UserData.StructuredUserData.Users[0].SSHAuthorizedKeys = append(
				req.UserData.StructuredUserData.Users[0].SSHAuthorizedKeys,
				publicKeys...,
			)
		}
	}

	// 处理 UserData（如果提供）
	if req.UserData != nil {
		cloudInitConfig, cloudInitUserData, err := s.convertUserDataToCloudInit(ctx, instanceID, req.UserData)
		if err != nil {
			// 清理已创建的 volume
			_ = s.storageService.DeleteVolume(ctx, volume.ID)
			return nil, fmt.Errorf("convert userdata to cloud-init: %w", err)
		}

		if cloudInitConfig != nil {
			domainConfig.CloudInit = cloudInitConfig
			logger.Info().Msg("Cloud-init config added to domain")
		}
		if cloudInitUserData != nil {
			domainConfig.CloudInitUserData = cloudInitUserData
			logger.Info().Msg("Cloud-init userdata added to domain")
		}
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

	// Instance 已存储在 libvirt 中，不需要额外保存
	return instance, nil
}

// formatDomainUUID 格式化 Domain UUID
func formatDomainUUID(uuid [16]byte) string {
	return hex.EncodeToString(uuid[:])
}

// convertUserDataToCloudInit 将 entity.UserDataConfig 转换为 cloudinit 配置
func (s *InstanceService) convertUserDataToCloudInit(
	ctx context.Context,
	instanceID string,
	userDataConfig *entity.UserDataConfig,
) (*cloudinit.Config, *cloudinit.UserData, error) {
	if userDataConfig == nil {
		return nil, nil, nil
	}

	// 如果提供了原始 YAML，直接使用 UserData
	if userDataConfig.RawUserData != "" {
		// 移除可能的 #cloud-config header（如果存在）
		rawData := strings.TrimSpace(userDataConfig.RawUserData)
		if strings.HasPrefix(rawData, "#cloud-config") {
			// 移除 header
			lines := strings.Split(rawData, "\n")
			if len(lines) > 1 {
				rawData = strings.Join(lines[1:], "\n")
			} else {
				rawData = ""
			}
		}

		// 解析为 UserData 结构
		var userData cloudinit.UserData
		if err := yaml.Unmarshal([]byte(rawData), &userData); err != nil {
			return nil, nil, fmt.Errorf("invalid raw userdata YAML: %w", err)
		}

		// 创建最小的 Config（仅用于 hostname）
		config := &cloudinit.Config{
			Hostname: instanceID,
		}

		return config, &userData, nil
	}

	// 使用结构化配置
	if userDataConfig.StructuredUserData == nil {
		return nil, nil, nil
	}

	structured := userDataConfig.StructuredUserData

	// 转换为 cloudinit.Config
	config := &cloudinit.Config{
		Hostname:    structured.Hostname,
		DisableRoot: structured.DisableRoot,
		Timezone:    structured.Timezone,
		Packages:    structured.Packages,
		Commands:    structured.RunCmd,
	}

	// 转换 Groups
	if len(structured.Groups) > 0 {
		config.Groups = make([]cloudinit.Group, len(structured.Groups))
		for i, g := range structured.Groups {
			config.Groups[i] = cloudinit.Group{
				Name:    g.Name,
				Members: g.Members,
			}
		}
	}

	// 转换 Users
	if len(structured.Users) > 0 {
		config.Users = make([]cloudinit.User, len(structured.Users))
		for i, u := range structured.Users {
			user := cloudinit.User{
				Name:              u.Name,
				Groups:            u.Groups,
				SSHAuthorizedKeys: u.SSHAuthorizedKeys,
				Shell:             u.Shell,
			}

			// 处理密码
			if u.HashedPasswd != "" {
				user.Passwd = u.HashedPasswd
			} else if u.PlainTextPasswd != "" {
				// 如果提供了明文密码，需要 hash
				hashed, err := cloudinit.HashPassword(u.PlainTextPasswd)
				if err != nil {
					return nil, nil, fmt.Errorf("hash password for user %s: %w", u.Name, err)
				}
				user.Passwd = hashed
			}

			// 处理 sudo
			if u.Sudo != "" {
				user.Sudo = u.Sudo
			}

			config.Users[i] = user
		}
	}

	// 转换 WriteFiles
	if len(structured.WriteFiles) > 0 {
		config.WriteFiles = make([]cloudinit.File, len(structured.WriteFiles))
		for i, f := range structured.WriteFiles {
			config.WriteFiles[i] = cloudinit.File{
				Path:        f.Path,
				Content:     f.Content,
				Owner:       f.Owner,
				Permissions: f.Permissions,
			}
		}
	}

	// 设置默认 hostname（如果未设置）
	if config.Hostname == "" {
		config.Hostname = instanceID
	}

	return config, nil, nil
}

// DescribeInstances 描述实例
func (s *InstanceService) DescribeInstances(ctx context.Context, req *entity.DescribeInstancesRequest) ([]entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Describing instances from libvirt")

	// 直接从 libvirt 获取所有 domain
	domains, err := s.libvirtClient.GetVMSummaries()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get VMs from libvirt")
		return nil, fmt.Errorf("get VMs from libvirt: %w", err)
	}

	logger.Debug().
		Int("total_domains", len(domains)).
		Msg("Retrieved domains from libvirt")

	// 转换为 Instance 对象
	instances := make([]entity.Instance, 0, len(domains))
	for _, domain := range domains {
		// 获取详细信息
		domainInfo, err := s.libvirtClient.GetDomainInfo(domain.UUID)
		if err != nil {
			logger.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to get domain info, skipping")
			continue
		}

		// 获取状态
		state, _, err := s.libvirtClient.GetDomainState(domain)
		if err != nil {
			logger.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to get domain state")
			state = 5 // Unknown
		}

		instance := entity.Instance{
			ID:         domain.Name,                   // 使用 domain name 作为 ID
			Name:       domain.Name,
			State:      convertDomainState(state),
			DomainUUID: formatDomainUUID(domain.UUID),
			DomainName: domain.Name,
			VCPUs:      domainInfo.VCPUs,
			MemoryMB:   domainInfo.Memory / 1024, // 转换为 MB
			CreatedAt:  "",                        // libvirt 不提供创建时间
		}

		// TODO: 如果需要 ImageID 和 VolumeID，可以从 domain metadata 读取
		instances = append(instances, instance)
	}

	// 应用过滤器（如果需要）
	instances = s.applyInstanceFilters(instances, req)

	// 按 name 升序排序
	s.sortInstancesByName(instances)

	logger.Info().
		Int("total", len(instances)).
		Msg("Describe instances completed")

	return instances, nil
}

// applyInstanceFilters 应用过滤器
func (s *InstanceService) applyInstanceFilters(instances []entity.Instance, req *entity.DescribeInstancesRequest) []entity.Instance {
	if req == nil {
		return instances
	}

	// 如果指定了 InstanceIDs，只返回匹配的
	if len(req.InstanceIDs) > 0 {
		filtered := make([]entity.Instance, 0)
		idSet := make(map[string]bool)
		for _, id := range req.InstanceIDs {
			idSet[id] = true
		}
		for _, instance := range instances {
			if idSet[instance.ID] {
				filtered = append(filtered, instance)
			}
		}
		return filtered
	}

	return instances
}

// convertDomainState 转换 libvirt 状态为 JVP 状态
func convertDomainState(state uint8) string {
	switch state {
	case 1: // Running
		return "running"
	case 3: // Paused
		return "stopped"
	case 4: // Shutdown
		return "stopped"
	case 5: // Shutoff
		return "stopped"
	case 6: // Crashed
		return "failed"
	default:
		return "pending"
	}
}

// GetInstance 获取单个实例信息
func (s *InstanceService) GetInstance(ctx context.Context, instanceID string) (*entity.Instance, error) {
	// instanceID 就是 domain name
	domain, err := s.libvirtClient.GetDomainByName(instanceID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// 获取详细信息
	domainInfo, err := s.libvirtClient.GetDomainInfo(domain.UUID)
	if err != nil {
		return nil, fmt.Errorf("get domain info: %w", err)
	}

	// 获取状态
	state, _, err := s.libvirtClient.GetDomainState(domain)
	if err != nil {
		state = 5 // Unknown
	}

	instance := &entity.Instance{
		ID:         domain.Name,
		Name:       domain.Name,
		State:      convertDomainState(state),
		DomainUUID: formatDomainUUID(domain.UUID),
		DomainName: domain.Name,
		VCPUs:      domainInfo.VCPUs,
		MemoryMB:   domainInfo.Memory / 1024,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	return instance, nil
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
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		// 获取 domain
		domain, err := s.libvirtClient.GetDomainByName(instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 删除 domain（会先停止运行中的实例）
		logger.Info().
			Str("instanceID", instanceID).
			Msg("Deleting domain from libvirt")
		err = s.libvirtClient.DeleteDomain(domain, libvirtlib.DomainUndefineFlagsValues(0))
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to delete domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to delete domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain deleted successfully")

		// 删除关联的 volume（可选，根据配置决定）
		if instance.VolumeID != "" {
			logger.Info().
				Str("instanceID", instanceID).
				Str("volumeID", instance.VolumeID).
				Msg("Deleting associated volume")
			if err := s.storageService.DeleteVolume(ctx, instance.VolumeID); err != nil {
				logger.Warn().
					Str("instanceID", instanceID).
					Str("volumeID", instance.VolumeID).
					Err(err).
					Msg("Failed to delete associated volume, continuing")
			}
		}

		// Domain 已从 libvirt 删除，不需要额外操作
		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "terminated",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "terminated").
			Msg("Instance terminated successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to terminate")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances terminated successfully")

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
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		if instance.State == "stopped" {
			// 已经停止，跳过
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Instance already stopped, skipping")
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
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 停止 domain
		if req.Force {
			// 强制停止
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Force stopping domain")
			err = s.libvirtClient.DestroyDomain(domain)
		} else {
			// 优雅停止
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Gracefully stopping domain")
			err = s.libvirtClient.StopDomain(domain)
		}

		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Bool("force", req.Force).
				Err(err).
				Msg("Failed to stop domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to stop domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain stop command sent successfully")

		// 状态已在 libvirt 中更新，不需要额外操作

		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "stopped",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "stopped").
			Msg("Instance stopped successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to stop")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances stopped successfully")

	return changes, nil
}

// StartInstances 启动实例
func (s *InstanceService) StartInstances(ctx context.Context, req *entity.StartInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Starting instances")

	var changes []entity.InstanceStateChange
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		if instance.State == "running" {
			// 已经运行，跳过
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Instance already running, skipping")
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
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 启动 domain
		logger.Info().
			Str("instanceID", instanceID).
			Msg("Starting domain")
		err = s.libvirtClient.StartDomain(domain)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to start domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to start domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain start command sent successfully")

		// 状态已在 libvirt 中更新，不需要额外操作
		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "running",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "running").
			Msg("Instance started successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to start")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances started successfully")

	return changes, nil
}

// RebootInstances 重启实例
func (s *InstanceService) RebootInstances(ctx context.Context, req *entity.RebootInstancesRequest) ([]entity.InstanceStateChange, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Strs("instanceIDs", req.InstanceIDs).
		Msg("Rebooting instances")

	var changes []entity.InstanceStateChange
	var lastError error

	for _, instanceID := range req.InstanceIDs {
		// 获取当前状态
		instance, err := s.GetInstance(ctx, instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get instance")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Instance not found", err)
		}
		previousState := instance.State

		if instance.State == "stopped" {
			// 如果已停止，先启动
			logger.Info().
				Str("instanceID", instanceID).
				Msg("Instance is stopped, starting before reboot")
			_, err = s.StartInstances(ctx, &entity.StartInstancesRequest{
				InstanceIDs: []string{instanceID},
			})
			if err != nil {
				logger.Error().
					Str("instanceID", instanceID).
					Err(err).
					Msg("Failed to start instance before reboot")
				lastError = err
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to start instance before reboot", err)
			}
			previousState = "running"
		}

		// 获取 domain
		domain, err := s.libvirtClient.GetDomainByName(instanceID)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to get domain from libvirt")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
		}

		// 重启 domain
		logger.Info().
			Str("instanceID", instanceID).
			Msg("Rebooting domain")
		err = s.libvirtClient.RebootDomain(domain)
		if err != nil {
			logger.Error().
				Str("instanceID", instanceID).
				Err(err).
				Msg("Failed to reboot domain")
			lastError = err
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to reboot domain", err)
		}

		logger.Info().
			Str("instanceID", instanceID).
			Msg("Domain reboot command sent successfully")

		// 状态已在 libvirt 中更新，不需要额外操作
		changes = append(changes, entity.InstanceStateChange{
			InstanceID:    instanceID,
			CurrentState:  "running",
			PreviousState: previousState,
		})

		logger.Info().
			Str("instanceID", instanceID).
			Str("previousState", previousState).
			Str("currentState", "running").
			Msg("Instance rebooted successfully")
	}

	if lastError != nil {
		logger.Error().
			Err(lastError).
			Msg("Some instances failed to reboot")
		return changes, lastError
	}

	logger.Info().
		Int("successCount", len(changes)).
		Msg("All instances rebooted successfully")

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

	// 属性已在 libvirt 中更新，重新获取实例信息以获取最新状态
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

// ResetPassword 重置实例密码
// 按优先级尝试三种方案：
// 1. qemu-guest-agent（优先，不需要停止实例）
// 2. cloud-init（如果 guest agent 不可用，需要重启实例）
// 3. virt-customize（最后选择，需要停止实例）
func (s *InstanceService) ResetPassword(ctx context.Context, req *entity.ResetPasswordRequest) (*entity.ResetPasswordResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instance_id", req.InstanceID).
		Int("user_count", len(req.Users)).
		Msg("Resetting instance password")

	// 1. 验证实例存在
	instance, err := s.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return nil, apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("Instance %s not found", req.InstanceID),
			404,
		)
	}

	// 2. 构建用户密码映射
	usersMap := make(map[string]string)
	userList := make([]string, 0, len(req.Users))
	for _, user := range req.Users {
		usersMap[user.Username] = user.NewPassword
		userList = append(userList, user.Username)
	}

	// 3. 记录原始状态
	wasRunning := instance.State == "running"

	// 4. 按优先级尝试不同的密码重置策略
	var strategyUsed string
	var resetErr error

	// 策略 1: qemu-guest-agent（优先，不需要停止实例，仅适用于运行中的实例）
	if wasRunning {
		logger.Info().
			Str("instance_id", req.InstanceID).
			Msg("Trying qemu-guest-agent strategy")

		guestAgentStrategy := NewQemuGuestAgentStrategy(s.libvirtClient)
		resetErr = guestAgentStrategy.ResetPassword(ctx, req.InstanceID, usersMap)
		if resetErr == nil {
			strategyUsed = guestAgentStrategy.Name()
			logger.Info().
				Str("instance_id", req.InstanceID).
				Str("strategy", strategyUsed).
				Msg("Password reset successful via qemu-guest-agent")
		} else {
			logger.Warn().
				Err(resetErr).
				Str("instance_id", req.InstanceID).
				Msg("qemu-guest-agent strategy failed, trying cloud-init")
		}
	}

	// 策略 2: cloud-init（如果 guest agent 失败，仅适用于运行中的实例）
	if resetErr != nil && wasRunning {
		logger.Info().
			Str("instance_id", req.InstanceID).
			Msg("Trying cloud-init strategy")

		cloudInitStrategy := NewCloudInitStrategy(s.libvirtClient, "")
		resetErr = cloudInitStrategy.ResetPassword(ctx, req.InstanceID, usersMap)
		if resetErr == nil {
			strategyUsed = cloudInitStrategy.Name()
			logger.Info().
				Str("instance_id", req.InstanceID).
				Str("strategy", strategyUsed).
				Msg("Password reset successful via cloud-init (requires restart)")
			// 注意：cloud-init 需要重启实例才能生效
			// 这里返回成功，但需要用户重启实例
		} else {
			logger.Warn().
				Err(resetErr).
				Str("instance_id", req.InstanceID).
				Msg("cloud-init strategy failed, falling back to virt-customize")
		}
	}

	// 策略 3: virt-customize（最后选择，需要停止实例）
	// 如果实例是停止状态，或者前面的策略都失败了，使用 virt-customize
	if resetErr != nil || !wasRunning {
		logger.Info().
			Str("instance_id", req.InstanceID).
			Msg("Trying virt-customize strategy")

		// 验证 virt-customize 客户端是否可用
		if s.virtCustomizeClient == nil {
			return nil, apierror.NewErrorWithStatus(
				"ServiceUnavailable",
				"virt-customize command not found, please install libguestfs-tools",
				503,
			)
		}

		// 如果实例正在运行，需要先停止
		if wasRunning {
			logger.Info().
				Str("instance_id", req.InstanceID).
				Msg("Stopping instance before virt-customize password reset")

			stopReq := &entity.StopInstancesRequest{
				InstanceIDs: []string{req.InstanceID},
				Force:       false,
			}
			_, err := s.StopInstances(ctx, stopReq)
			if err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to stop instance", err)
			}

			// 等待实例完全停止
			maxWait := 30 * time.Second
			waitInterval := 1 * time.Second
			waited := time.Duration(0)
			for waited < maxWait {
				instance, err := s.GetInstance(ctx, req.InstanceID)
				if err == nil && instance.State == "stopped" {
					break
				}
				time.Sleep(waitInterval)
				waited += waitInterval
			}

			// 再次检查状态
			instance, err = s.GetInstance(ctx, req.InstanceID)
			if err != nil || instance.State != "stopped" {
				return nil, apierror.NewErrorWithStatus(
					"InternalError",
					fmt.Sprintf("Instance %s failed to stop within timeout", req.InstanceID),
					500,
				)
			}
		}

		// 获取实例的磁盘路径
		disks, err := s.libvirtClient.GetDomainDisks(req.InstanceID)
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get instance disks", err)
		}

		if len(disks) == 0 || disks[0].Source.File == "" {
			return nil, apierror.NewErrorWithStatus(
				"InvalidParameter",
				fmt.Sprintf("Instance %s has no disk", req.InstanceID),
				400,
			)
		}

		diskPath := disks[0].Source.File

		// 调用 virt-customize 重置密码（直接传入 diskPath，避免重复调用 GetDomainDisks 和 ValidateDiskPath）
		virtCustomizeStrategy := NewVirtCustomizeStrategy(s.virtCustomizeClient, s.libvirtClient)
		resetErr = virtCustomizeStrategy.ResetPassword(ctx, diskPath, usersMap)
		if resetErr == nil {
			strategyUsed = virtCustomizeStrategy.Name()
			logger.Info().
				Str("instance_id", req.InstanceID).
				Str("strategy", strategyUsed).
				Msg("Password reset successful via virt-customize")
		}
	}

	// 5. 检查重置结果
	if resetErr != nil {
		logger.Error().
			Err(resetErr).
			Str("instance_id", req.InstanceID).
			Msg("All password reset strategies failed")

		// 如果之前是运行状态，尝试恢复
		if wasRunning && req.AutoStart {
			_, _ = s.StartInstances(ctx, &entity.StartInstancesRequest{
				InstanceIDs: []string{req.InstanceID},
			})
		}

		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to reset passwords", resetErr)
	}

	// 6. 如果之前是运行状态且 AutoStart=true，启动实例（仅 virt-customize 策略需要）
	if wasRunning && req.AutoStart && strategyUsed == "virt-customize" {
		logger.Info().
			Str("instance_id", req.InstanceID).
			Msg("Starting instance after password reset")

		_, err := s.StartInstances(ctx, &entity.StartInstancesRequest{
			InstanceIDs: []string{req.InstanceID},
		})
		if err != nil {
			logger.Warn().
				Err(err).
				Str("instance_id", req.InstanceID).
				Msg("Failed to start instance after password reset")
			// 密码重置成功，但启动失败，返回警告但不失败
		}
	}

	logger.Info().
		Str("instance_id", req.InstanceID).
		Str("strategy", strategyUsed).
		Strs("users", userList).
		Msg("Password reset successfully")

	message := fmt.Sprintf("Password reset successfully via %s", strategyUsed)
	if strategyUsed == "cloud-init" {
		message += " (instance restart required)"
	}

	return &entity.ResetPasswordResponse{
		InstanceID: req.InstanceID,
		Success:    true,
		Message:    message,
		Users:      userList,
	}, nil
}

// ListVMTemplates 列出所有可用的 VM 模板
// VM Template 是指带有快照的虚拟机，可以基于快照克隆新的 VM
func (s *InstanceService) ListVMTemplates(ctx context.Context) ([]entity.VMTemplate, error) {
	logger := zerolog.Ctx(ctx)

	// 直接从 libvirt 获取所有 domain（包括不在 metadata store 中的）
	domains, err := s.libvirtClient.GetVMSummaries()
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs from libvirt: %w", err)
	}

	templates := make([]entity.VMTemplate, 0)

	// 遍历每个 domain，检查是否有快照
	for _, domain := range domains {
		// 获取 domain 的快照列表
		snapshots, err := s.libvirtClient.ListSnapshots(domain.Name)
		if err != nil {
			logger.Warn().
				Str("domain_name", domain.Name).
				Err(err).
				Msg("Failed to list snapshots for domain")
			continue
		}

		logger.Debug().
			Str("domain_name", domain.Name).
			Int("snapshot_count", len(snapshots)).
			Msg("Checked domain for snapshots")

		// 如果 domain 有快照，将其作为模板
		if len(snapshots) > 0 {
			// 获取 domain 详细信息
			domainInfo, err := s.libvirtClient.GetDomainInfo(domain.UUID)
			if err != nil {
				logger.Warn().
					Str("domain_name", domain.Name).
					Err(err).
					Msg("Failed to get domain info")
				continue
			}

			template := entity.VMTemplate{
				ID:          formatDomainUUID(domain.UUID),
				Name:        domain.Name + "-template",
				Description: fmt.Sprintf("Template based on %s with %d snapshots", domain.Name, len(snapshots)),
				SourceVM:    domain.Name,
				VCPUs:       domainInfo.VCPUs,
				Memory:      domainInfo.Memory / 1024, // 转换为 MB
				DiskSize:    20,                       // 默认磁盘大小，可以后续优化从实际磁盘获取
				CreatedAt:   time.Now().Format(time.RFC3339),
			}
			templates = append(templates, template)

			logger.Debug().
				Str("domain_name", domain.Name).
				Str("template_id", template.ID).
				Msg("Added domain as template")
		}
	}

	logger.Info().
		Int("template_count", len(templates)).
		Msg("Listed VM templates")

	return templates, nil
}

// sortInstancesByName 按 name 升序排序实例
func (s *InstanceService) sortInstancesByName(instances []entity.Instance) {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})
}
