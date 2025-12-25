package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/qemuimg"
	"github.com/rs/zerolog"
)

// SnapshotService 提供快照管理能力
type SnapshotService struct {
	nodeService *NodeService
	idGen       *idgen.Generator
}

// NewSnapshotService 创建快照服务
func NewSnapshotService(nodeService *NodeService) *SnapshotService {
	return &SnapshotService{
		nodeService: nodeService,
		idGen:       idgen.New(),
	}
}

// CreateSnapshot 创建外部快照（磁盘为外部增量，存储在 _snapshots_/vm/ 下）
func (s *SnapshotService) CreateSnapshot(ctx context.Context, req *entity.CreateSnapshotRequest) (*entity.Snapshot, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Bool("with_memory", req.WithMemory).
		Msg("Creating snapshot")

	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	domain, err := client.GetDomainByName(req.VMName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to find domain", err)
	}

	disks, err := client.GetDomainDisks(domain.Name)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain disks", err)
	}

	if req.SnapshotName == "" {
		id, genErr := s.idGen.GenerateID()
		if genErr != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate snapshot id", genErr)
		}
		req.SnapshotName = fmt.Sprintf("snap-%d", id)
	}

	now := time.Now().UTC()
	safeSnapshotName := sanitizeName(req.SnapshotName)

	snapshotXML := libvirt.DomainSnapshotXML{
		Name:        safeSnapshotName,
		Description: req.Description,
	}
	if req.WithMemory {
		snapshotXML.Memory = &libvirt.DomainSnapshotMemoryXML{
			Snapshot: "internal",
		}
	}

	for _, disk := range disks {
		if disk.Device != "disk" || disk.Source.File == "" || disk.Target.Dev == "" {
			continue
		}

		destDir := filepath.Join(filepath.Dir(disk.Source.File), SnapshotsDirName, req.VMName)
		if err := ensureDir(client, destDir); err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to prepare snapshot directory", err)
		}

		fileName := fmt.Sprintf("%s-%s-%s.qcow2", disk.Target.Dev, safeSnapshotName, now.Format("20060102-150405"))
		targetPath := filepath.Join(destDir, fileName)

		snapshotXML.Disks = append(snapshotXML.Disks, libvirt.DomainSnapshotDiskXML{
			Name:     disk.Target.Dev,
			Snapshot: "external",
			Driver: &libvirt.DomainSnapshotDiskDriverXML{
				Type: "qcow2",
			},
			Source: &libvirt.DomainSnapshotDiskSourceXML{
				File: targetPath,
			},
		})
	}

	if len(snapshotXML.Disks) == 0 {
		return nil, apierror.WrapError(apierror.ErrInternalError, "No valid disks found for snapshot", nil)
	}

	xmlBytes, err := xml.Marshal(snapshotXML)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to marshal snapshot XML", err)
	}

	flags := libvirtlib.DomainSnapshotCreateFlags(libvirtlib.DomainSnapshotCreateAtomic)
	if !req.WithMemory {
		flags |= libvirtlib.DomainSnapshotCreateDiskOnly
	}

	if err := client.CreateSnapshot(domain.Name, string(xmlBytes), flags); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to create snapshot", err)
	}

	// 读取最新的快照信息
	created, err := client.GetSnapshotXML(domain.Name, safeSnapshotName)
	if err != nil {
		logger.Warn().Err(err).Msg("Snapshot created but failed to read XML; returning basic info")
		return &entity.Snapshot{
			ID:       safeSnapshotName,
			Name:     safeSnapshotName,
			VMName:   domain.Name,
			NodeName: req.NodeName,
			DiskOnly: !req.WithMemory,
			Memory:   req.WithMemory,
		}, nil
	}

	return convertSnapshot(req.NodeName, domain.Name, created), nil
}

// ListSnapshots 列举快照
func (s *SnapshotService) ListSnapshots(ctx context.Context, req *entity.ListSnapshotsRequest) ([]entity.Snapshot, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Msg("Listing snapshots")

	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	snapXMLs, err := client.ListSnapshotXML(req.VMName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to list snapshots", err)
	}

	sort.SliceStable(snapXMLs, func(i, j int) bool {
		ci, cj := snapXMLs[i].CreationTime, snapXMLs[j].CreationTime
		switch {
		case ci == 0 && cj == 0:
			return snapXMLs[i].Name < snapXMLs[j].Name
		case ci == 0:
			return false
		case cj == 0:
			return true
		default:
			return ci < cj
		}
	})

	result := make([]entity.Snapshot, 0, len(snapXMLs))
	for _, snap := range snapXMLs {
		result = append(result, *convertSnapshot(req.NodeName, req.VMName, &snap))
	}

	return result, nil
}

// DescribeSnapshot 查询快照详情
func (s *SnapshotService) DescribeSnapshot(ctx context.Context, req *entity.DescribeSnapshotRequest) (*entity.Snapshot, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Msg("Describing snapshot")

	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	snap, err := client.GetSnapshotXML(req.VMName, req.SnapshotName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to describe snapshot", err)
	}

	return convertSnapshot(req.NodeName, req.VMName, snap), nil
}

// DeleteSnapshot 删除快照
// 注意：对于外部快照（external snapshot），libvirt 无法自动合并磁盘链，
// 因此默认只删除快照元数据。快照的磁盘文件需要手动清理或使用 blockcommit/blockpull 操作。
func (s *SnapshotService) DeleteSnapshot(ctx context.Context, req *entity.DeleteSnapshotRequest) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Bool("delete_children", req.DeleteChildren).
		Bool("metadata_only", req.MetadataOnly).
		Bool("disks_only", req.DisksOnly).
		Msg("Deleting snapshot")

	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	var flags libvirtlib.DomainSnapshotDeleteFlags
	if req.DeleteChildren {
		flags |= libvirtlib.DomainSnapshotDeleteChildren
	}

	// 对于外部快照，强制使用 MetadataOnly，因为 libvirt 无法自动合并外部快照的磁盘链
	// 如果用户明确设置了 MetadataOnly 或没有设置任何特殊选项，都使用 MetadataOnly
	if req.MetadataOnly || !req.DisksOnly {
		flags |= libvirtlib.DomainSnapshotDeleteMetadataOnly
	}

	if err := client.DeleteSnapshot(req.VMName, req.SnapshotName, flags); err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete snapshot", err)
	}
	return nil
}

// RevertSnapshot 回滚到快照
func (s *SnapshotService) RevertSnapshot(ctx context.Context, req *entity.RevertSnapshotRequest) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("vm_name", req.VMName).
		Str("snapshot_name", req.SnapshotName).
		Bool("start_after_revert", req.StartAfterRevert).
		Bool("force", req.Force).
		Msg("Reverting snapshot")

	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	var flags libvirtlib.DomainSnapshotRevertFlags
	if req.StartAfterRevert {
		flags |= libvirtlib.DomainSnapshotRevertRunning
	}
	if req.Force {
		flags |= libvirtlib.DomainSnapshotRevertForce
	}

	if err := client.RevertToSnapshot(req.VMName, req.SnapshotName, flags); err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to revert snapshot", err)
	}
	return nil
}

func ensureDir(client libvirt.LibvirtClient, dir string) error {
	if client.IsRemoteConnection() {
		return client.ExecuteRemoteCommand(fmt.Sprintf("mkdir -p '%s'", dir))
	}
	return os.MkdirAll(dir, 0o755)
}

func convertSnapshot(nodeName, vmName string, snap *libvirt.DomainSnapshotXML) *entity.Snapshot {
	result := &entity.Snapshot{
		ID:       snap.Name,
		Name:     snap.Name,
		VMName:   vmName,
		NodeName: nodeName,
		State:    snap.State,
		DiskOnly: snap.Memory == nil || strings.EqualFold(snap.Memory.Snapshot, "no"),
		Memory:   snap.Memory != nil && !strings.EqualFold(snap.Memory.Snapshot, "no"),
	}

	if snap.CreationTime > 0 {
		result.CreatedAt = time.Unix(snap.CreationTime, 0).UTC().Format(time.RFC3339)
	}
	if snap.Parent != nil {
		result.Parent = snap.Parent.Name
	}
	result.Description = snap.Description

	for _, disk := range snap.Disks {
		var path string
		if disk.Source != nil {
			path = disk.Source.File
		}
		var format string
		if disk.Driver != nil {
			format = disk.Driver.Type
		}
		result.Disks = append(result.Disks, entity.SnapshotDisk{
			Target: disk.Name,
			Path:   path,
			Format: format,
		})
	}

	return result
}

func sanitizeName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "snapshot"
	}
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return s
}

// CloneFromSnapshot 基于快照克隆创建新实例
func (s *SnapshotService) CloneFromSnapshot(ctx context.Context, req *entity.CloneFromSnapshotRequest) (*entity.Instance, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("source_vm", req.SourceVMName).
		Str("snapshot_name", req.SnapshotName).
		Str("new_vm_name", req.NewVMName).
		Bool("flatten", req.Flatten).
		Msg("Cloning instance from snapshot")

	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 1. 获取快照信息
	snap, err := client.GetSnapshotXML(req.SourceVMName, req.SnapshotName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get snapshot info", err)
	}

	// 2. 获取源 VM 信息
	sourceDomain, err := client.GetDomainByName(req.SourceVMName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get source VM", err)
	}

	sourceInfo, err := client.GetDomainInfo(sourceDomain.UUID)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get source VM info", err)
	}

	// 3. 生成新 VM 名称
	newVMName := req.NewVMName
	if newVMName == "" {
		id, genErr := s.idGen.GenerateID()
		if genErr != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate VM name", genErr)
		}
		newVMName = fmt.Sprintf("%s-clone-%d", req.SourceVMName, id)
	}

	// 4. 获取存储池路径
	poolInfo, err := client.GetStoragePool(req.PoolName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get storage pool info", err)
	}
	poolPath := poolInfo.Path

	// 5. 准备 qemu-img 客户端
	qemuClient := s.createQemuImgClient(client)

	// 6. 处理快照磁盘 - 为新 VM 创建磁盘
	// 关键点：快照磁盘（snap-xxx.qcow2）是 VM 当前使用的增量文件，
	// 它的 backing file 才是快照时刻的状态。
	// 所以我们需要获取 backing file 路径作为克隆源。
	var newDiskPath string
	for _, disk := range snap.Disks {
		if disk.Source == nil || disk.Source.File == "" {
			continue
		}

		snapshotDiskPath := disk.Source.File
		diskFormat := "qcow2"
		if disk.Driver != nil && disk.Driver.Type != "" {
			diskFormat = disk.Driver.Type
		}

		// 获取 backing file 路径（这才是快照时刻的状态）
		backingFile, err := qemuClient.GetBackingFile(ctx, snapshotDiskPath)
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get backing file path", err)
		}

		logger.Info().
			Str("snapshot_disk", snapshotDiskPath).
			Str("backing_file", backingFile).
			Msg("Found backing file info for snapshot disk")

		// 如果快照磁盘没有 backing file，说明它本身就是完整的快照状态
		// 这种情况应该直接使用 snapshotDiskPath
		if backingFile == "" {
			logger.Info().
				Str("snapshot_disk", snapshotDiskPath).
				Msg("Snapshot disk has no backing file, using snapshot disk directly as source")
			backingFile = snapshotDiskPath
		}

		// 新磁盘路径
		newDiskPath = filepath.Join(poolPath, fmt.Sprintf("%s.qcow2", newVMName))

		logger.Info().
			Str("src_disk", backingFile).
			Str("new_disk", newDiskPath).
			Bool("flatten", req.Flatten).
			Msg("Creating disk for cloned VM from backing file (snapshot state)")

		if req.Flatten {
			// 合并增量链 - 使用 qemu-img convert
			// backing file 是只读的，不需要 -U 参数
			if err := qemuClient.Convert(ctx, diskFormat, "qcow2", backingFile, newDiskPath); err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to convert/flatten disk", err)
			}
		} else {
			// 保留增量链 - 使用 qemu-img create -b
			// 基于 backing file 创建新的增量文件
			if err := qemuClient.CreateFromBackingFile(ctx, "qcow2", diskFormat, backingFile, newDiskPath); err != nil {
				return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to create disk from backing file", err)
			}
		}

		// 只处理第一个有效磁盘（主磁盘）
		break
	}

	if newDiskPath == "" {
		return nil, apierror.WrapError(apierror.ErrInternalError, "No valid disk found in snapshot", nil)
	}

	// 7. 确定新 VM 配置 - 继承源 VM 或使用用户覆盖值
	vcpus := uint16(sourceInfo.VCPUs)
	if req.VCPUs > 0 {
		vcpus = uint16(req.VCPUs)
	}

	memoryKB := sourceInfo.Memory
	if req.MemoryMB > 0 {
		memoryKB = uint64(req.MemoryMB) * 1024
	}

	networkType := req.NetworkType
	networkSource := req.NetworkSource
	if networkType == "" {
		networkType = "bridge"
	}
	if networkSource == "" {
		networkSource = "br0"
	}

	// 8. 创建新 VM
	vmConfig := &libvirt.CreateVMConfig{
		Name:          newVMName,
		Memory:        memoryKB,
		VCPUs:         vcpus,
		DiskPath:      newDiskPath,
		DiskBus:       "virtio",
		NetworkType:   networkType,
		NetworkSource: networkSource,
		OSType:        "hvm",
		Architecture:  "x86_64",
		VNCSocket:     fmt.Sprintf("/var/lib/jvp/qemu/%s.vnc", newVMName),
	}

	domain, err := client.CreateDomain(vmConfig, req.StartAfterClone)
	if err != nil {
		// 清理已创建的磁盘
		s.cleanupDisk(client, newDiskPath)
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to create cloned VM", err)
	}

	// 9. 获取新 VM 状态
	state := "stopped"
	if req.StartAfterClone {
		state = "running"
	}

	logger.Info().
		Str("new_vm", newVMName).
		Str("state", state).
		Msg("Successfully cloned instance from snapshot")

	return &entity.Instance{
		ID:         newVMName,
		Name:       newVMName,
		State:      state,
		NodeName:   req.NodeName,
		MemoryMB:   memoryKB / 1024,
		VCPUs:      vcpus,
		DomainUUID: fmt.Sprintf("%x", domain.UUID),
		DomainName: domain.Name,
	}, nil
}

// createQemuImgClient 创建 qemu-img 客户端，支持本地和远程
func (s *SnapshotService) createQemuImgClient(client libvirt.LibvirtClient) *qemuimg.Client {
	qemuClient := qemuimg.New("")
	if client.IsRemoteConnection() {
		sshTarget, err := client.GetSSHTarget()
		if err == nil {
			qemuClient = qemuClient.WithSSHTarget(sshTarget)
		}
	}
	return qemuClient
}

// cleanupDisk 清理磁盘文件
func (s *SnapshotService) cleanupDisk(client libvirt.LibvirtClient, diskPath string) {
	if client.IsRemoteConnection() {
		_ = client.ExecuteRemoteCommand(fmt.Sprintf("rm -f '%s'", diskPath))
	} else {
		_ = os.Remove(diskPath)
	}
}
