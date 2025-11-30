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
	if req.MetadataOnly {
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
