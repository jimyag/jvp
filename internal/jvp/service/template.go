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
	nodeStorageFn   NodeStorageGetter
	store           *TemplateStore
	idGen           *idgen.Generator
	downloadManager *DownloadTaskManager
}

// NewTemplateService 创建新的 TemplateService
func NewTemplateService(nodeStorageFn NodeStorageGetter, store *TemplateStore) *TemplateService {
	return &TemplateService{
		nodeStorageFn:   nodeStorageFn,
		store:           store,
		idGen:           idgen.New(),
		downloadManager: NewDownloadTaskManager(),
	}
}

// RegisterTemplateResult 注册模板结果
type RegisterTemplateResult struct {
	Template     *entity.Template
	DownloadTask *DownloadTask
	IsAsync      bool // 是否是异步下载
}

// RegisterTemplate 基于现有卷注册模板
// 如果 Source.Type 为 "url"，则异步下载文件到存储池
func (s *TemplateService) RegisterTemplate(ctx context.Context, req *entity.RegisterTemplateRequest) (*RegisterTemplateResult, error) {
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

	// 如果 Source.Type 为 "url"，启动异步下载
	if req.Source != nil && req.Source.Type == "url" && req.Source.URL != "" {
		// 检查是否已有下载任务
		existingTask := s.downloadManager.GetTaskByVolume(nodeName, req.PoolName, req.VolumeName)
		if existingTask != nil && (existingTask.Status == DownloadTaskStatusPending || existingTask.Status == DownloadTaskStatusRunning) {
			logger.Info().
				Str("task_id", existingTask.ID).
				Str("status", string(existingTask.Status)).
				Msg("Download task already in progress")

			return &RegisterTemplateResult{
				DownloadTask: existingTask,
				IsAsync:      true,
			}, nil
		}

		// 生成任务 ID
		taskIDNum, err := s.idGen.GenerateID()
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate task ID", err)
		}
		taskID := fmt.Sprintf("task-%d", taskIDNum)

		// 创建下载任务
		task, isNew := s.downloadManager.CreateTask(taskID, nodeName, req.PoolName, req.VolumeName, req.Source.URL)

		if !isNew {
			// 任务已存在
			logger.Info().
				Str("task_id", task.ID).
				Str("status", string(task.Status)).
				Msg("Download task already exists")

			return &RegisterTemplateResult{
				DownloadTask: task,
				IsAsync:      true,
			}, nil
		}

		logger.Info().
			Str("task_id", task.ID).
			Str("url", req.Source.URL).
			Str("volume_name", req.VolumeName).
			Msg("Starting async download")

		// 保存请求信息以便下载完成后注册模板
		reqCopy := *req
		s.downloadManager.StartDownload(ctx, task, client, func(completedTask *DownloadTask, downloadErr error) {
			if downloadErr != nil {
				logger.Error().
					Err(downloadErr).
					Str("task_id", completedTask.ID).
					Msg("Download failed, template not registered")
				return
			}

			// 下载成功，自动注册模板
			logger.Info().
				Str("task_id", completedTask.ID).
				Msg("Download completed, registering template")

			template, err := s.registerTemplateFromVolume(ctx, &reqCopy, client, nodeName)
			if err != nil {
				logger.Error().
					Err(err).
					Str("task_id", completedTask.ID).
					Msg("Failed to register template after download")
				return
			}

			logger.Info().
				Str("task_id", completedTask.ID).
				Str("template_id", template.ID).
				Msg("Template registered successfully after download")
		})

		return &RegisterTemplateResult{
			DownloadTask: task,
			IsAsync:      true,
		}, nil
	}

	// 同步注册（卷已存在）
	template, err := s.registerTemplateFromVolume(ctx, req, client, nodeName)
	if err != nil {
		return nil, err
	}

	return &RegisterTemplateResult{
		Template: template,
		IsAsync:  false,
	}, nil
}

// registerTemplateFromVolume 从已存在的卷注册模板
func (s *TemplateService) registerTemplateFromVolume(ctx context.Context, req *entity.RegisterTemplateRequest, client libvirt.LibvirtClient, nodeName string) (*entity.Template, error) {
	logger := zerolog.Ctx(ctx)

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
		SizeGB:      float64(volumeInfo.CapacityB) / (1024 * 1024 * 1024),
		Source:      cloneTemplateSource(req.Source),
		OS:          req.OS,
		Features:    req.Features,
		Usage:       entity.TemplateUsage{},
		Tags:        cloneTags(req.Tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Save(ctx, template); err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to persist template metadata", err)
	}

	logger.Info().
		Str("template_id", template.ID).
		Str("node_name", template.NodeName).
		Str("pool_name", template.PoolName).
		Msg("Template registered")

	return template, nil
}

// GetDownloadTask 获取下载任务状态
func (s *TemplateService) GetDownloadTask(ctx context.Context, taskID string) (*DownloadTask, error) {
	task := s.downloadManager.GetTask(taskID)
	if task == nil {
		return nil, apierror.NewErrorWithStatus("NotFound", "Download task not found", http.StatusNotFound)
	}
	return task, nil
}

// ListDownloadTasks 列出所有活跃的下载任务
func (s *TemplateService) ListDownloadTasks(ctx context.Context) []*DownloadTask {
	return s.downloadManager.ListActiveTasks()
}

// ListTemplates 列举模板
func (s *TemplateService) ListTemplates(ctx context.Context, req *entity.ListTemplatesRequest) ([]entity.Template, error) {
	if req == nil {
		req = &entity.ListTemplatesRequest{}
	}
	nodeName := normalizeNodeName(req.NodeName)

	// 如果指定了存储池，直接查询
	if req.PoolName != "" {
		templates, err := s.store.List(ctx, nodeName, req.PoolName)
		if err != nil {
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to list templates", err)
		}
		return templates, nil
	}

	// 否则需要遍历存储池
	// 暂时返回空列表，因为需要提供存储池
	// TODO: 遍历所有存储池
	return []entity.Template{}, nil
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
	if req.PoolName == "" {
		return nil, invalidParameterError("pool_name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	template, err := s.store.Get(ctx, nodeName, req.PoolName, req.TemplateID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apierror.NewErrorWithStatus(
				"Template.NotFound",
				fmt.Sprintf("template %s not found on node %s pool %s", req.TemplateID, nodeName, req.PoolName),
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
	if req.PoolName == "" {
		return nil, invalidParameterError("pool_name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	template, err := s.store.Get(ctx, nodeName, req.PoolName, req.TemplateID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apierror.NewErrorWithStatus(
				"Template.NotFound",
				fmt.Sprintf("template %s not found on node %s pool %s", req.TemplateID, nodeName, req.PoolName),
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
		if err := s.store.Save(ctx, template); err != nil {
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
	if req.PoolName == "" {
		return invalidParameterError("pool_name")
	}

	nodeName := normalizeNodeName(req.NodeName)
	template, err := s.store.Get(ctx, nodeName, req.PoolName, req.TemplateID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apierror.NewErrorWithStatus(
				"Template.NotFound",
				fmt.Sprintf("template %s not found on node %s pool %s", req.TemplateID, nodeName, req.PoolName),
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

	if err := s.store.Delete(ctx, nodeName, req.PoolName, req.TemplateID); err != nil {
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete template metadata", err)
	}

	zerolog.Ctx(ctx).Info().
		Str("template_id", req.TemplateID).
		Str("node_name", nodeName).
		Msg("Template deleted")

	return nil
}

func (s *TemplateService) lookupVolume(client libvirt.LibvirtClient, poolName, volumeName string) (*libvirt.VolumeInfo, error) {
	// 获取存储池信息
	poolInfo, err := client.GetStoragePool(poolName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get storage pool", err)
	}

	if poolInfo.Path == "" {
		return nil, apierror.NewErrorWithStatus(
			"Template.InvalidPool",
			fmt.Sprintf("storage pool %s has no path", poolName),
			http.StatusBadRequest,
		)
	}

	// 在 _templates_ 目录中查找文件
	templatesDir := poolInfo.Path + "/" + TemplatesDirName
	candidates := buildVolumeCandidates(volumeName)

	for _, candidate := range candidates {
		filePath := templatesDir + "/" + candidate
		var fileExists bool
		var fileSize int64

		if client.IsRemoteConnection() {
			// 远程：通过 SSH 检查文件是否存在
			checkCmd := fmt.Sprintf("test -f '%s'", filePath)
			if err := client.ExecuteRemoteCommand(checkCmd); err == nil {
				fileExists = true
				// 对于远程文件，暂时不获取文件大小（需要读取整个文件太慢）
				// 文件大小会在需要时从卷信息中获取
				fileSize = 0
			}
		} else {
			// 本地：直接检查文件
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				fileExists = true
				fileSize = info.Size()
			}
		}

		if fileExists {
			// 根据文件扩展名判断格式
			format := "raw"
			if strings.HasSuffix(candidate, ".qcow2") {
				format = "qcow2"
			} else if strings.HasSuffix(candidate, ".img") {
				format = "raw"
			}

			return &libvirt.VolumeInfo{
				Name:        candidate,
				Path:        filePath,
				CapacityB:   uint64(fileSize),
				AllocationB: uint64(fileSize),
				Format:      format,
			}, nil
		}
	}

	return nil, apierror.NewErrorWithStatus(
		"Template.VolumeNotFound",
		fmt.Sprintf("volume %s not found in pool %s/_templates_", volumeName, poolName),
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
