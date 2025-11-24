package service

import (
	"context"
	"fmt"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// VolumeService 存储卷服务（简化版，移除EBS概念）
type VolumeService struct {
	storageService  *StorageService
	instanceService *InstanceService
	libvirtClient   libvirt.LibvirtClient
	qemuImgClient   qemuimg.QemuImgClient
	idGen           *idgen.Generator
}

// NewVolumeService 创建新的 Volume Service
func NewVolumeService(
	storageService *StorageService,
	instanceService *InstanceService,
	libvirtClient libvirt.LibvirtClient,
) *VolumeService {
	return &VolumeService{
		storageService:  storageService,
		instanceService: instanceService,
		libvirtClient:   libvirtClient,
		qemuImgClient:   qemuimg.New(""),
		idGen:           idgen.New(),
	}
}

// CreateVolume 创建存储卷
func (s *VolumeService) CreateVolume(ctx context.Context, req *entity.CreateVolumeRequest) (*entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Uint64("sizeGB", req.SizeGB).
		Msg("Creating volume")

	// 生成 Volume ID
	volumeID, err := s.idGen.GenerateVolumeID()
	if err != nil {
		return nil, fmt.Errorf("generate volume ID: %w", err)
	}

	// 创建存储卷
	internalReq := &entity.CreateInternalVolumeRequest{
		PoolName: "default",
		VolumeID: volumeID,
		SizeGB:   req.SizeGB,
		Format:   "qcow2",
	}

	internalVolume, err := s.storageService.CreateVolume(ctx, internalReq)
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	volume := &entity.Volume{
		ID:        volumeID,
		Name:      volumeID,
		Pool:      "default",
		Path:      internalVolume.Path,
		CapacityB: internalVolume.CapacityB,
		SizeGB:    req.SizeGB,
		Format:    internalVolume.Format,
		State:     "available",
	}

	logger.Info().
		Str("volumeID", volumeID).
		Msg("Volume created successfully")

	return volume, nil
}

// DeleteVolume 删除存储卷
func (s *VolumeService) DeleteVolume(ctx context.Context, volumeID string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("volumeID", volumeID).Msg("Deleting volume")

	// 检查卷是否被附加
	volume, err := s.GetVolume(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("get volume: %w", err)
	}

	attachments := s.findVolumeAttachments(ctx, volume.Path)
	if len(attachments) > 0 {
		return fmt.Errorf("volume %s is attached to instance(s), cannot delete", volumeID)
	}

	// 删除存储卷
	err = s.storageService.DeleteVolume(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("delete volume: %w", err)
	}

	logger.Info().Str("volumeID", volumeID).Msg("Volume deleted successfully")
	return nil
}

// AttachVolume 将卷附加到实例
func (s *VolumeService) AttachVolume(ctx context.Context, req *entity.AttachVolumeRequest) (*entity.VolumeAttachment, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("Attaching volume to instance")

	// 获取卷信息
	volume, err := s.GetVolume(ctx, req.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	// 检查卷是否已被附加
	attachments := s.findVolumeAttachments(ctx, volume.Path)
	if len(attachments) > 0 {
		return nil, fmt.Errorf("volume %s is already attached", req.VolumeID)
	}

	// 获取 domain 的磁盘列表，确定可用的设备名
	disks, err := s.libvirtClient.GetDomainDisks(req.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("get domain disks: %w", err)
	}

	// 确定设备名
	device := req.Device
	if device == "" {
		// 自动分配设备名（从 /dev/vdb 开始）
		usedDevices := make(map[string]bool)
		for _, disk := range disks {
			if disk.Device == "disk" {
				usedDevices[disk.Target.Dev] = true
			}
		}
		// 从 vdb 开始查找可用设备
		for i := 1; i < 26; i++ {
			candidate := fmt.Sprintf("/dev/vd%c", 'a'+i)
			if !usedDevices[candidate] {
				device = candidate
				break
			}
		}
		if device == "" {
			return nil, fmt.Errorf("no available device slot for domain")
		}
	} else {
		// 检查设备是否已被使用
		for _, disk := range disks {
			if disk.Target.Dev == device && disk.Device == "disk" {
				return nil, fmt.Errorf("device %s already in use", device)
			}
		}
	}

	// 附加磁盘到 domain
	err = s.libvirtClient.AttachDiskToDomain(req.InstanceID, volume.Path, device)
	if err != nil {
		return nil, fmt.Errorf("attach disk to domain: %w", err)
	}

	attachment := &entity.VolumeAttachment{
		VolumeID:   req.VolumeID,
		InstanceID: req.InstanceID,
		Device:     device,
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Str("device", device).
		Msg("Volume attached successfully")

	return attachment, nil
}

// DetachVolume 从实例分离卷
func (s *VolumeService) DetachVolume(ctx context.Context, req *entity.DetachVolumeRequest) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("Detaching volume from instance")

	// 获取卷信息
	volume, err := s.GetVolume(ctx, req.VolumeID)
	if err != nil {
		return fmt.Errorf("get volume: %w", err)
	}

	// 查找附加信息
	attachments := s.findVolumeAttachments(ctx, volume.Path)
	var attachment *entity.VolumeAttachment
	for i := range attachments {
		if req.InstanceID == "" || attachments[i].InstanceID == req.InstanceID {
			attachment = &attachments[i]
			break
		}
	}

	if attachment == nil {
		return fmt.Errorf("volume %s is not attached to instance %s", req.VolumeID, req.InstanceID)
	}

	// 从 domain 分离磁盘
	err = s.libvirtClient.DetachDiskFromDomain(attachment.InstanceID, attachment.Device)
	if err != nil {
		return fmt.Errorf("detach disk from domain: %w", err)
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", attachment.InstanceID).
		Msg("Volume detached successfully")

	return nil
}

// ListVolumes 列出所有存储卷
func (s *VolumeService) ListVolumes(ctx context.Context) ([]entity.Volume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Listing volumes from libvirt storage pools")

	// 从 StorageService 获取所有卷
	pools, err := s.storageService.ListStoragePools(ctx, true)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to list storage pools")
		return nil, fmt.Errorf("list storage pools: %w", err)
	}

	// 收集所有卷
	var volumes []entity.Volume
	for _, pool := range pools {
		for _, vol := range pool.Volumes {
			// 跳过镜像池和模板池
			if pool.Name == "images" || pool.Name == "template" {
				continue
			}

			// 查找卷的附加关系（实时查询）
			attachments := s.findVolumeAttachments(ctx, vol.Path)

			// 计算状态
			state := "available"
			if len(attachments) > 0 {
				state = "in-use"
			}

			volume := entity.Volume{
				ID:          vol.ID,
				Name:        vol.Name,
				Pool:        pool.Name,
				Path:        vol.Path,
				CapacityB:   vol.CapacityB,
				SizeGB:      vol.CapacityB / (1024 * 1024 * 1024), // 转换为 GB
				AllocationB: vol.AllocationB,
				Format:      vol.Format,
				State:       state,
				VolumeType:  vol.VolumeType,
				Attachments: attachments,
			}

			volumes = append(volumes, volume)
		}
	}

	logger.Info().
		Int("total", len(volumes)).
		Msg("List volumes completed")

	return volumes, nil
}

// GetVolume 获取单个存储卷
func (s *VolumeService) GetVolume(ctx context.Context, volumeID string) (*entity.Volume, error) {
	volumes, err := s.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}

	for i := range volumes {
		if volumes[i].ID == volumeID {
			return &volumes[i], nil
		}
	}

	return nil, fmt.Errorf("volume %s not found", volumeID)
}

// ResizeVolume 调整卷大小
func (s *VolumeService) ResizeVolume(ctx context.Context, volumeID string, newSizeGB uint64) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", volumeID).
		Uint64("newSizeGB", newSizeGB).
		Msg("Resizing volume")

	// 获取卷路径
	volume, err := s.GetVolume(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("get volume: %w", err)
	}

	currentSizeGB := volume.CapacityB / (1024 * 1024 * 1024)
	if newSizeGB <= currentSizeGB {
		return fmt.Errorf("new size must be larger than current size (%d GB)", currentSizeGB)
	}

	// 使用 qemu-img resize 调整大小
	err = s.qemuImgClient.Resize(ctx, volume.Path, newSizeGB)
	if err != nil {
		return fmt.Errorf("resize volume: %w", err)
	}

	logger.Info().
		Str("volumeID", volumeID).
		Uint64("newSizeGB", newSizeGB).
		Msg("Volume resized successfully")

	return nil
}

// findVolumeAttachments 查找卷的附加关系（实时从 libvirt 查询）
func (s *VolumeService) findVolumeAttachments(ctx context.Context, volumePath string) []entity.VolumeAttachment {
	logger := zerolog.Ctx(ctx)
	var attachments []entity.VolumeAttachment

	// 获取所有 domain
	domains, err := s.libvirtClient.GetVMSummaries()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get VM summaries")
		return attachments
	}

	// 遍历每个 domain，检查是否使用了这个卷
	for _, domain := range domains {
		disks, err := s.libvirtClient.GetDomainDisks(domain.Name)
		if err != nil {
			continue
		}

		for _, disk := range disks {
			if disk.Device == "disk" && disk.Source.File == volumePath {
				attachment := entity.VolumeAttachment{
					VolumeID:   volumePath, // 使用路径作为临时ID
					InstanceID: domain.Name,
					Device:     disk.Target.Dev,
				}
				attachments = append(attachments, attachment)
			}
		}
	}

	return attachments
}
