package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/rs/zerolog"
)

// NodeStorageGetter 定义获取节点存储连接的函数类型
type NodeStorageGetter func(ctx context.Context, nodeName string) (libvirt.LibvirtClient, error)

// TemplateService 管理模板元数据以及与存储层的交互
type TemplateService struct {
	nodeStorageFn NodeStorageGetter
	store         *TemplateStore
	idGen         *idgen.Generator
}

// NewTemplateService 创建新的 TemplateService
func NewTemplateService(nodeStorageFn NodeStorageGetter, store *TemplateStore) *TemplateService {
	return &TemplateService{
		nodeStorageFn: nodeStorageFn,
		store:         store,
		idGen:         idgen.New(),
	}
}

// RegisterTemplate 基于现有卷注册模板
func (s *TemplateService) RegisterTemplate(ctx context.Context, req *entity.RegisterTemplateRequest) (*entity.Template, error) {
	logger := zerolog.Ctx(ctx)
	if req == nil {
		return nil, apierror.NewErrorWithStatus("InvalidParameter", "request body is required", http.StatusBadRequest)
	}

	if req.PoolName == "" {
		return nil, invalidParameterError("pool_name")
	}
	if req.VolumeName == "" {
		return nil, invalidParameterError("volume_name")
	}
	if req.Name == "" {
		return nil, invalidParameterError("name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	client, err := s.getNodeClient(ctx, nodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node storage", err)
	}

	volumeInfo, err := s.lookupVolume(client, req.PoolName, req.VolumeName)
	if err != nil {
		return nil, err
	}

	templateID, err := s.idGen.GenerateTemplateID()
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate template ID", err)
	}

	now := time.Now().UTC()
	template := &entity.Template{
		ID:          templateID,
		Name:        req.Name,
		Description: req.Description,
		NodeName:    nodeName,
		PoolName:    req.PoolName,
		VolumeName:  volumeInfo.Name,
		Path:        volumeInfo.Path,
		Format:      volumeInfo.Format,
		SizeBytes:   volumeInfo.CapacityB,
		SizeGB:      volumeInfo.CapacityB / (1024 * 1024 * 1024),
		Source:      cloneTemplateSource(req.Source),
		OS:          req.OS,
		Features:    req.Features,
		Usage:       entity.TemplateUsage{},
		Tags:        cloneTags(req.Tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Save(template); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to persist template metadata", err)
	}

	logger.Info().
		Str("template_id", template.ID).
		Str("node_name", template.NodeName).
		Str("pool_name", template.PoolName).
		Msg("Template registered")

	return template, nil
}

// ListTemplates 列举模板
func (s *TemplateService) ListTemplates(ctx context.Context, req *entity.ListTemplatesRequest) ([]entity.Template, error) {
	if req == nil {
		req = &entity.ListTemplatesRequest{}
	}
	nodeName := req.NodeName
	if nodeName != "" {
		nodeName = normalizeNodeName(nodeName)
	}

	templates, err := s.store.List(nodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to list templates", err)
	}

	if req.PoolName == "" {
		return templates, nil
	}

	filtered := make([]entity.Template, 0, len(templates))
	for _, tpl := range templates {
		if tpl.PoolName == req.PoolName {
			filtered = append(filtered, tpl)
		}
	}
	return filtered, nil
}

// DescribeTemplate 查询模板详情
func (s *TemplateService) DescribeTemplate(ctx context.Context, req *entity.DescribeTemplateRequest) (*entity.Template, error) {
	if req == nil {
		return nil, apierror.NewErrorWithStatus("InvalidParameter", "request body is required", http.StatusBadRequest)
	}
	if req.TemplateID == "" {
		return nil, invalidParameterError("template_id")
	}
	if req.NodeName == "" {
		return nil, invalidParameterError("node_name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	template, err := s.store.Get(nodeName, req.TemplateID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apierror.NewErrorWithStatus(
				"Template.NotFound",
				fmt.Sprintf("template %s not found on node %s", req.TemplateID, nodeName),
				http.StatusNotFound,
			)
		}
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to load template metadata", err)
	}
	return template, nil
}

// UpdateTemplate 更新模板元数据
func (s *TemplateService) UpdateTemplate(ctx context.Context, req *entity.UpdateTemplateRequest) (*entity.Template, error) {
	if req == nil {
		return nil, apierror.NewErrorWithStatus("InvalidParameter", "request body is required", http.StatusBadRequest)
	}
	if req.TemplateID == "" {
		return nil, invalidParameterError("template_id")
	}
	if req.NodeName == "" {
		return nil, invalidParameterError("node_name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	template, err := s.store.Get(nodeName, req.TemplateID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apierror.NewErrorWithStatus(
				"Template.NotFound",
				fmt.Sprintf("template %s not found on node %s", req.TemplateID, nodeName),
				http.StatusNotFound,
			)
		}
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to load template metadata", err)
	}

	modified := false
	if req.Description != nil && template.Description != *req.Description {
		template.Description = *req.Description
		modified = true
	}
	if req.Tags != nil {
		template.Tags = cloneTags(*req.Tags)
		modified = true
	}
	if req.Features != nil {
		template.Features = *req.Features
		modified = true
	}
	if req.OS != nil {
		template.OS = *req.OS
		modified = true
	}

	if modified {
		template.UpdatedAt = time.Now().UTC()
		if err := s.store.Save(template); err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to update template metadata", err)
		}
	}

	return template, nil
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(ctx context.Context, req *entity.DeleteTemplateRequest) error {
	if req == nil {
		return apierror.NewErrorWithStatus("InvalidParameter", "request body is required", http.StatusBadRequest)
	}
	if req.TemplateID == "" {
		return invalidParameterError("template_id")
	}
	if req.NodeName == "" {
		return invalidParameterError("node_name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	template, err := s.store.Get(nodeName, req.TemplateID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apierror.NewErrorWithStatus(
				"Template.NotFound",
				fmt.Sprintf("template %s not found on node %s", req.TemplateID, nodeName),
				http.StatusNotFound,
			)
		}
		return apierror.WrapError(apierror.ErrInternalError, "Failed to load template metadata", err)
	}

	if req.DeleteVolume {
		client, err := s.getNodeClient(ctx, nodeName)
		if err != nil {
			return apierror.WrapError(apierror.ErrInternalError, "Failed to get node storage", err)
		}
		if err := client.DeleteVolume(template.PoolName, template.VolumeName); err != nil {
			return apierror.WrapError(apierror.ErrInternalError, "Failed to delete backing volume", err)
		}
	}

	if err := s.store.Delete(nodeName, req.TemplateID); err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete template metadata", err)
	}

	zerolog.Ctx(ctx).Info().
		Str("template_id", req.TemplateID).
		Str("node_name", nodeName).
		Msg("Template deleted")

	return nil
}

func (s *TemplateService) lookupVolume(client libvirt.LibvirtClient, poolName, volumeName string) (*libvirt.VolumeInfo, error) {
	candidates := buildVolumeCandidates(volumeName)
	for _, candidate := range candidates {
		vol, err := client.GetVolume(poolName, candidate)
		if err == nil {
			return vol, nil
		}
	}
	return nil, apierror.NewErrorWithStatus(
		"Template.VolumeNotFound",
		fmt.Sprintf("volume %s not found in pool %s", volumeName, poolName),
		http.StatusBadRequest,
	)
}

func buildVolumeCandidates(volumeName string) []string {
	base := strings.TrimSpace(volumeName)
	if base == "" {
		return []string{}
	}

	extensions := []string{".qcow2", ".raw", ".img"}
	candidates := []string{base}
	hasExt := strings.Contains(base, ".")
	if !hasExt {
		for _, ext := range extensions {
			candidates = append(candidates, base+ext)
		}
	}
	return candidates
}

func normalizeNodeName(nodeName string) string {
	if strings.TrimSpace(nodeName) == "" {
		return "local"
	}
	return nodeName
}

func cloneTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	cp := make([]string, len(tags))
	copy(cp, tags)
	return cp
}

func cloneTemplateSource(src *entity.TemplateSource) *entity.TemplateSource {
	if src == nil {
		return nil
	}
	copied := *src
	return &copied
}

func invalidParameterError(name string) *apierror.Error {
	return apierror.NewErrorWithStatus(
		"InvalidParameter",
		fmt.Sprintf("%s is required", name),
		http.StatusBadRequest,
	)
}

func (s *TemplateService) getNodeClient(ctx context.Context, nodeName string) (libvirt.LibvirtClient, error) {
	if s.nodeStorageFn == nil {
		return nil, fmt.Errorf("node storage provider is not configured")
	}
	return s.nodeStorageFn(ctx, nodeName)
}
