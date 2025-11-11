package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// VolumeService EBS Volume 服务
type VolumeService struct {
	storageService  *StorageService
	instanceService *InstanceService
	libvirtClient   libvirt.LibvirtClient
	qemuImgClient   qemuimg.QemuImgClient
	idGen           *idgen.Generator
	volumeRepo      repository.VolumeRepository
	snapshotRepo    repository.SnapshotRepository
}

// NewVolumeService 创建新的 Volume Service
func NewVolumeService(
	storageService *StorageService,
	instanceService *InstanceService,
	libvirtClient libvirt.LibvirtClient,
	repo *repository.Repository,
) *VolumeService {
	return &VolumeService{
		storageService:  storageService,
		instanceService: instanceService,
		libvirtClient:   libvirtClient,
		qemuImgClient:   qemuimg.New(""),
		idGen:           idgen.New(),
		volumeRepo:      repository.NewVolumeRepository(repo.DB()),
		snapshotRepo:    repository.NewSnapshotRepository(repo.DB()),
	}
}

// CreateEBSVolume 创建 EBS 卷
func (s *VolumeService) CreateEBSVolume(ctx context.Context, req *entity.CreateVolumeRequest) (*entity.EBSVolume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Uint64("sizeGB", req.SizeGB).
		Str("volumeType", req.VolumeType).
		Msg("Creating EBS volume")

	// 生成 Volume ID
	volumeID, err := s.idGen.GenerateVolumeID()
	if err != nil {
		return nil, fmt.Errorf("generate volume ID: %w", err)
	}

	// 确定大小（如果从快照创建，使用快照大小；否则使用请求的大小）
	sizeGB := req.SizeGB
	var sourceVolumePath string
	if req.SnapshotID != "" {
		// 从快照创建卷
		// 获取快照信息
		snapshotModel, err := s.snapshotRepo.GetByID(ctx, req.SnapshotID)
		if err != nil {
			return nil, fmt.Errorf("snapshot %s not found: %w", req.SnapshotID, err)
		}
		if snapshotModel.State != "completed" {
			return nil, fmt.Errorf("snapshot %s is not completed (state: %s)", req.SnapshotID, snapshotModel.State)
		}

		// 如果未指定大小，使用快照大小
		if sizeGB == 0 {
			sizeGB = snapshotModel.VolumeSizeGB
		}

		// 获取源卷信息
		sourceVolume, err := s.storageService.GetVolume(ctx, snapshotModel.VolumeID)
		if err != nil {
			return nil, fmt.Errorf("get source volume %s: %w", snapshotModel.VolumeID, err)
		}
		sourceVolumePath = sourceVolume.Path
	}

	// 创建内部 Volume
	internalReq := &entity.CreateInternalVolumeRequest{
		PoolName: "default",
		VolumeID: volumeID,
		SizeGB:   sizeGB,
		Format:   "qcow2",
	}

	internalVolume, err := s.storageService.CreateVolume(ctx, internalReq)
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	// 如果从快照创建，需要从源卷复制数据
	if req.SnapshotID != "" && sourceVolumePath != "" {
		// 删除 CreateVolume 创建的空文件，因为 Convert 需要创建新文件
		if err := os.Remove(internalVolume.Path); err != nil && !os.IsNotExist(err) {
			// 清理已创建的 volume
			_ = s.storageService.DeleteVolume(ctx, volumeID)
			return nil, fmt.Errorf("remove empty volume file: %w", err)
		}

		// 从源卷复制到新卷（这会包含快照状态）
		err = s.qemuImgClient.Convert(ctx, "qcow2", "qcow2", sourceVolumePath, internalVolume.Path)
		if err != nil {
			// 清理已创建的 volume
			_ = s.storageService.DeleteVolume(ctx, volumeID)
			return nil, fmt.Errorf("convert volume from snapshot: %w", err)
		}

		// 如果需要调整大小
		sourceSizeGB := internalVolume.CapacityB / (1024 * 1024 * 1024)
		if sourceSizeGB < sizeGB {
			err = s.qemuImgClient.Resize(ctx, internalVolume.Path, sizeGB)
			if err != nil {
				// 清理已创建的 volume
				_ = s.storageService.DeleteVolume(ctx, volumeID)
				return nil, fmt.Errorf("resize volume: %w", err)
			}
		}

		// 重新获取 volume 信息（因为大小可能已改变）
		_, err = s.storageService.GetVolume(ctx, volumeID)
		if err != nil {
			return nil, fmt.Errorf("get volume info: %w", err)
		}
	}

	// 转换为 EBS Volume
	ebsVolume := &entity.EBSVolume{
		VolumeID:         volumeID,
		SizeGB:           sizeGB,
		SnapshotID:       req.SnapshotID,
		AvailabilityZone: req.AvailabilityZone,
		State:            "available",
		VolumeType:       req.VolumeType,
		Iops:             req.Iops,
		Encrypted:        req.Encrypted,
		KmsKeyID:         req.KmsKeyID,
		Attachments:      []entity.VolumeAttachment{},
		CreateTime:       time.Now().Format(time.RFC3339),
		Tags:             extractTags(req.TagSpecifications, "volume"),
	}

	// 保存到数据库
	volumeModel, err := volumeEntityToModel(ebsVolume)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to convert volume to model", err)
	}
	if err := s.volumeRepo.Create(ctx, volumeModel); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save volume to database", err)
	}
	logger.Info().Str("volumeID", volumeID).Msg("Volume saved to database")

	logger.Info().
		Str("volumeID", volumeID).
		Msg("EBS volume created successfully")

	return ebsVolume, nil
}

// DeleteEBSVolume 删除 EBS 卷
func (s *VolumeService) DeleteEBSVolume(ctx context.Context, volumeID string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("volumeID", volumeID).Msg("Deleting EBS volume")

	// 检查卷是否被附加
	volume, err := s.DescribeEBSVolume(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("get volume: %w", err)
	}

	if len(volume.Attachments) > 0 {
		return fmt.Errorf("volume %s is attached to instance(s), cannot delete", volumeID)
	}

	// 删除内部 Volume
	err = s.storageService.DeleteVolume(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("delete volume: %w", err)
	}

	// 从数据库软删除
	if err := s.volumeRepo.Delete(ctx, volumeID); err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete volume from database", err)
	}

	logger.Info().Str("volumeID", volumeID).Msg("EBS volume deleted successfully")
	return nil
}

// AttachEBSVolume 附加卷到实例
func (s *VolumeService) AttachEBSVolume(ctx context.Context, req *entity.AttachVolumeRequest) (*entity.VolumeAttachment, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("Attaching EBS volume to instance")

	// 获取卷信息
	volume, err := s.DescribeEBSVolume(ctx, req.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	if volume.State != "available" {
		return nil, fmt.Errorf("volume %s is not available (state: %s)", req.VolumeID, volume.State)
	}

	// 获取卷的路径
	internalVolume, err := s.storageService.GetVolume(ctx, req.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
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
	err = s.libvirtClient.AttachDiskToDomain(req.InstanceID, internalVolume.Path, device)
	if err != nil {
		return nil, fmt.Errorf("attach disk to domain: %w", err)
	}

	// 更新卷状态为 in-use
	volumeModel, err := s.volumeRepo.GetByID(ctx, req.VolumeID)
	if err == nil {
		volumeModel.State = "in-use"
		if err := s.volumeRepo.Update(ctx, volumeModel); err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to update volume state in database", err)
		}
	}

	attachment := &entity.VolumeAttachment{
		VolumeID:            req.VolumeID,
		InstanceID:          req.InstanceID,
		Device:              device,
		State:               "attached",
		AttachTime:          time.Now().Format(time.RFC3339),
		DeleteOnTermination: false,
	}

	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Str("device", device).
		Msg("EBS volume attached successfully")

	return attachment, nil
}

// DetachEBSVolume 从实例分离卷
func (s *VolumeService) DetachEBSVolume(ctx context.Context, req *entity.DetachVolumeRequest) (*entity.VolumeAttachment, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", req.InstanceID).
		Msg("Detaching EBS volume from instance")

	// 获取卷信息
	volume, err := s.DescribeEBSVolume(ctx, req.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	// 查找附加信息
	var attachment *entity.VolumeAttachment
	for i, att := range volume.Attachments {
		if req.InstanceID == "" || att.InstanceID == req.InstanceID {
			attachment = &volume.Attachments[i]
			break
		}
	}

	if attachment == nil {
		return nil, fmt.Errorf("volume %s is not attached to instance %s", req.VolumeID, req.InstanceID)
	}

	// 从 domain 分离磁盘
	err = s.libvirtClient.DetachDiskFromDomain(attachment.InstanceID, attachment.Device)
	if err != nil {
		return nil, fmt.Errorf("detach disk from domain: %w", err)
	}

	// 更新卷状态为 available
	volumeModel, err := s.volumeRepo.GetByID(ctx, req.VolumeID)
	if err == nil {
		volumeModel.State = "available"
		if err := s.volumeRepo.Update(ctx, volumeModel); err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to update volume state in database", err)
		}
	}

	attachment.State = "detached"

	logger.Info().
		Str("volumeID", req.VolumeID).
		Str("instanceID", attachment.InstanceID).
		Msg("EBS volume detached successfully")

	return attachment, nil
}

// DescribeEBSVolumes 描述 EBS 卷
func (s *VolumeService) DescribeEBSVolumes(ctx context.Context, req *entity.DescribeVolumesRequest) ([]entity.EBSVolume, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Describing EBS volumes")

	var volumes []entity.EBSVolume

	if len(req.VolumeIDs) > 0 {
		// 查询指定的卷
		for _, volumeID := range req.VolumeIDs {
			volume, err := s.DescribeEBSVolume(ctx, volumeID)
			if err != nil {
				// 如果卷不存在，跳过
				logger.Warn().Err(err).Str("volumeID", volumeID).Msg("Volume not found, skipping")
				continue
			}
			volumes = append(volumes, *volume)
		}
	} else {
		// 构建过滤器
		filters := make(map[string]interface{})
		if len(req.Filters) > 0 {
			for _, filter := range req.Filters {
				switch filter.Name {
				case "state":
					if len(filter.Values) > 0 {
						filters["state"] = filter.Values[0]
					}
				case "volume-type":
					if len(filter.Values) > 0 {
						filters["volume_type"] = filter.Values[0]
					}
				case "snapshot-id":
					if len(filter.Values) > 0 {
						filters["snapshot_id"] = filter.Values[0]
					}
				}
			}
		}

		// 优先从数据库查询
		volumeModels, err := s.volumeRepo.List(ctx, filters)
		if err != nil {
			return nil, fmt.Errorf("list volumes from database: %w", err)
		}

		for _, volumeModel := range volumeModels {
			volume, err := volumeModelToEntity(volumeModel)
			if err != nil {
				logger.Warn().Err(err).Str("volumeID", volumeModel.ID).Msg("Failed to convert volume model to entity")
				continue
			}
			// 补充附加信息
			volumeWithAttachments, err := s.enrichVolumeWithAttachments(ctx, volume)
			if err != nil {
				logger.Warn().Err(err).Str("volumeID", volumeModel.ID).Msg("Failed to enrich volume with attachments")
			} else {
				volumes = append(volumes, *volumeWithAttachments)
			}
		}
	}

	// 应用其他过滤器（不在数据库中的）
	if len(req.Filters) > 0 {
		volumes = s.applyFilters(volumes, req.Filters)
	}

	return volumes, nil
}

// DescribeEBSVolume 描述单个 EBS 卷
func (s *VolumeService) DescribeEBSVolume(ctx context.Context, volumeID string) (*entity.EBSVolume, error) {
	// 优先从数据库查询
	volumeModel, err := s.volumeRepo.GetByID(ctx, volumeID)
	if err == nil {
		volume, err := volumeModelToEntity(volumeModel)
		if err != nil {
			return nil, fmt.Errorf("convert volume model to entity: %w", err)
		}
		// 补充附加信息
		return s.enrichVolumeWithAttachments(ctx, volume)
	}

	// 如果数据库中没有，从 storage service 查询（兼容旧数据）
	volume, err := s.storageService.GetVolume(ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	// 获取文件创建时间
	var createTime string
	if fileInfo, err := os.Stat(volume.Path); err == nil {
		createTime = fileInfo.ModTime().Format(time.RFC3339)
	}

	ebsVolume := &entity.EBSVolume{
		VolumeID:         volume.ID,
		SizeGB:           volume.CapacityB / (1024 * 1024 * 1024),
		AvailabilityZone: "default",
		State:            "available",
		VolumeType:       "gp2",
		Attachments:      []entity.VolumeAttachment{},
		CreateTime:       createTime,
		Tags:             []entity.Tag{},
	}

	// 补充附加信息
	return s.enrichVolumeWithAttachments(ctx, ebsVolume)
}

// enrichVolumeWithAttachments 补充卷的附加信息
func (s *VolumeService) enrichVolumeWithAttachments(ctx context.Context, volume *entity.EBSVolume) (*entity.EBSVolume, error) {
	// 获取卷的路径
	internalVolume, err := s.storageService.GetVolume(ctx, volume.VolumeID)
	if err != nil {
		// 如果获取不到路径，返回原始卷信息
		return volume, nil
	}

	// 检查卷是否被附加到任何实例
	attachments := []entity.VolumeAttachment{}
	state := volume.State

	// 列出所有 domain，检查哪些 domain 使用了这个卷
	domains, err := s.libvirtClient.GetVMSummaries()
	if err == nil {
		for _, domain := range domains {
			disks, err := s.libvirtClient.GetDomainDisks(domain.Name)
			if err != nil {
				continue
			}
			for _, disk := range disks {
				if disk.Device == "disk" && disk.Source.File == internalVolume.Path {
					// 找到附加的实例
					attachment := entity.VolumeAttachment{
						VolumeID:            volume.VolumeID,
						InstanceID:          domain.Name,
						Device:              disk.Target.Dev,
						State:               "attached",
						AttachTime:          volume.CreateTime, // 使用卷创建时间作为附加时间
						DeleteOnTermination: false,
					}
					attachments = append(attachments, attachment)
					state = "in-use"
				}
			}
		}
	}

	volume.Attachments = attachments
	volume.State = state

	return volume, nil
}

// ModifyEBSVolume 修改 EBS 卷属性
func (s *VolumeService) ModifyEBSVolume(ctx context.Context, req *entity.ModifyVolumeRequest) (*entity.VolumeModification, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("volumeID", req.VolumeID).
		Uint64("sizeGB", req.SizeGB).
		Msg("Modifying EBS volume")

	// 获取卷信息
	volume, err := s.DescribeEBSVolume(ctx, req.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	modification := &entity.VolumeModification{
		VolumeID:          req.VolumeID,
		ModificationState: "modifying",
		StatusMessage:     "Modification in progress",
		StartTime:         time.Now().Format(time.RFC3339),
	}

	// 如果修改大小
	if req.SizeGB > 0 && req.SizeGB > volume.SizeGB {
		// 获取卷路径
		internalVolume, err := s.storageService.GetVolume(ctx, req.VolumeID)
		if err != nil {
			return nil, fmt.Errorf("get volume: %w", err)
		}

		// 使用 qemu-img resize 调整大小
		err = s.qemuImgClient.Resize(ctx, internalVolume.Path, req.SizeGB)
		if err != nil {
			return nil, fmt.Errorf("resize volume: %w", err)
		}

		modification.TargetSizeGB = req.SizeGB
		logger.Info().
			Str("volumeID", req.VolumeID).
			Uint64("newSizeGB", req.SizeGB).
			Msg("Volume resized successfully")

		// 更新数据库中的大小
		volumeModel, err := s.volumeRepo.GetByID(ctx, req.VolumeID)
		if err == nil {
			volumeModel.SizeGB = req.SizeGB
			if err := s.volumeRepo.Update(ctx, volumeModel); err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to update volume size in database", err)
			}
		}
	}

	// 如果修改类型或 IOPS
	if req.VolumeType != "" {
		modification.TargetVolumeType = req.VolumeType
		// 更新数据库
		volumeModel, err := s.volumeRepo.GetByID(ctx, req.VolumeID)
		if err == nil {
			volumeModel.VolumeType = req.VolumeType
			if err := s.volumeRepo.Update(ctx, volumeModel); err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to update volume type in database", err)
			}
		}
	}
	if req.Iops > 0 {
		modification.TargetIops = req.Iops
		// 更新数据库
		volumeModel, err := s.volumeRepo.GetByID(ctx, req.VolumeID)
		if err == nil {
			volumeModel.Iops = int(req.Iops)
			if err := s.volumeRepo.Update(ctx, volumeModel); err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to update volume IOPS in database", err)
			}
		}
	}

	modification.ModificationState = "completed"
	modification.EndTime = time.Now().Format(time.RFC3339)

	logger.Info().
		Str("volumeID", req.VolumeID).
		Msg("EBS volume modified successfully")

	return modification, nil
}

// extractTags 从 TagSpecifications 中提取标签
func extractTags(specs []entity.TagSpecification, resourceType string) []entity.Tag {
	var tags []entity.Tag
	for _, spec := range specs {
		if spec.ResourceType == resourceType {
			tags = append(tags, spec.Tags...)
		}
	}
	return tags
}

// applyFilters 应用过滤器
func (s *VolumeService) applyFilters(volumes []entity.EBSVolume, filters []entity.Filter) []entity.EBSVolume {
	var result []entity.EBSVolume

	for _, volume := range volumes {
		match := true
		for _, filter := range filters {
			if !s.matchesFilter(volume, filter) {
				match = false
				break
			}
		}
		if match {
			result = append(result, volume)
		}
	}

	return result
}

// matchesFilter 检查卷是否匹配过滤器
func (s *VolumeService) matchesFilter(volume entity.EBSVolume, filter entity.Filter) bool {
	for _, value := range filter.Values {
		switch filter.Name {
		case "volume-id":
			if volume.VolumeID == value {
				return true
			}
		case "state":
			if volume.State == value {
				return true
			}
		case "volume-type":
			if volume.VolumeType == value {
				return true
			}
		case "attachment.instance-id":
			for _, att := range volume.Attachments {
				if att.InstanceID == value {
					return true
				}
			}
		}
	}
	return false
}
