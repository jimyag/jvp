package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
	"gopkg.in/yaml.v3"
)

// TemplatesDirName 模板目录名称
const TemplatesDirName = "_templates_"

// TemplateStore 负责模板元数据的持久化
// 模板元数据保存在对应存储池的 _templates_ 目录中
type TemplateStore struct {
	nodeStorageFn NodeStorageGetter
	mu            sync.RWMutex
}

// NewTemplateStore 创建模板存储
func NewTemplateStore(nodeStorageFn NodeStorageGetter) *TemplateStore {
	return &TemplateStore{
		nodeStorageFn: nodeStorageFn,
	}
}

// getPoolPath 获取存储池的路径
func (s *TemplateStore) getPoolPath(ctx context.Context, nodeName, poolName string) (string, error) {
	client, err := s.nodeStorageFn(ctx, nodeName)
	if err != nil {
		return "", fmt.Errorf("get node client: %w", err)
	}

	pool, err := client.GetStoragePool(poolName)
	if err != nil {
		return "", fmt.Errorf("get storage pool: %w", err)
	}

	if pool.Path == "" {
		return "", fmt.Errorf("storage pool %s has no path", poolName)
	}

	return pool.Path, nil
}

// getTemplatesDir 获取模板目录路径
func (s *TemplateStore) getTemplatesDir(poolPath string) string {
	return filepath.Join(poolPath, TemplatesDirName)
}

// getMetadataPath 获取模板元数据文件路径
func (s *TemplateStore) getMetadataPath(poolPath, templateID string) string {
	return filepath.Join(s.getTemplatesDir(poolPath), templateID+".yaml")
}

// ensureTemplatesDir 确保模板目录存在
func (s *TemplateStore) ensureTemplatesDir(ctx context.Context, nodeName, poolPath string) error {
	templatesDir := s.getTemplatesDir(poolPath)

	// 判断是否是远程连接
	client, err := s.nodeStorageFn(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("get node client: %w", err)
	}

	if client.IsRemoteConnection() {
		// 远程连接：通过 SSH 创建目录
		return s.ensureTemplatesDirRemote(client, templatesDir)
	}

	// 本地连接：直接创建目录
	return os.MkdirAll(templatesDir, 0o755)
}

// ensureTemplatesDirRemote 通过 SSH 在远程创建目录
func (s *TemplateStore) ensureTemplatesDirRemote(client libvirt.LibvirtClient, templatesDir string) error {
	return client.ExecuteRemoteCommand(fmt.Sprintf("mkdir -p '%s'", templatesDir))
}

// Save 保存模板元数据
func (s *TemplateStore) Save(ctx context.Context, template *entity.Template) error {
	if template == nil {
		return errors.New("template is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	poolPath, err := s.getPoolPath(ctx, template.NodeName, template.PoolName)
	if err != nil {
		return err
	}

	// 确保模板目录存在
	if err := s.ensureTemplatesDir(ctx, template.NodeName, poolPath); err != nil {
		return fmt.Errorf("ensure templates directory: %w", err)
	}

	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("marshal template metadata: %w", err)
	}

	metadataPath := s.getMetadataPath(poolPath, template.ID)

	// 判断是否是远程连接
	client, err := s.nodeStorageFn(ctx, template.NodeName)
	if err != nil {
		return fmt.Errorf("get node client: %w", err)
	}

	if client.IsRemoteConnection() {
		// 远程连接：通过 SSH 写入文件
		return s.writeFileRemote(client, metadataPath, data)
	}

	// 本地连接：直接写入文件
	tmpPath := metadataPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write template metadata: %w", err)
	}

	if err := os.Rename(tmpPath, metadataPath); err != nil {
		return fmt.Errorf("finalize template metadata: %w", err)
	}

	return nil
}

// writeFileRemote 通过 SSH 写入文件
func (s *TemplateStore) writeFileRemote(client libvirt.LibvirtClient, path string, data []byte) error {
	// 使用 cat 和 heredoc 写入文件
	content := string(data)
	// 转义单引号
	content = strings.ReplaceAll(content, "'", "'\"'\"'")
	cmd := fmt.Sprintf("cat > '%s' << 'EOF'\n%s\nEOF", path, content)
	return client.ExecuteRemoteCommand(cmd)
}

// List 列举模板
func (s *TemplateStore) List(ctx context.Context, nodeName, poolName string) ([]entity.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 如果指定了 nodeName 和 poolName，只查询该存储池
	if nodeName != "" && poolName != "" {
		return s.listPoolTemplates(ctx, nodeName, poolName)
	}

	// 否则需要遍历所有节点和存储池 - 这需要外部调用者提供节点和池列表
	// 这里暂时返回空列表，由 TemplateService 负责聚合
	return []entity.Template{}, nil
}

// listPoolTemplates 列举指定存储池的模板
func (s *TemplateStore) listPoolTemplates(ctx context.Context, nodeName, poolName string) ([]entity.Template, error) {
	poolPath, err := s.getPoolPath(ctx, nodeName, poolName)
	if err != nil {
		return nil, err
	}

	templatesDir := s.getTemplatesDir(poolPath)

	client, err := s.nodeStorageFn(ctx, nodeName)
	if err != nil {
		return nil, fmt.Errorf("get node client: %w", err)
	}

	if client.IsRemoteConnection() {
		return s.listPoolTemplatesRemote(client, templatesDir)
	}

	// 本地连接：直接读取目录
	return s.listPoolTemplatesLocal(templatesDir)
}

// listPoolTemplatesLocal 本地列举模板
func (s *TemplateStore) listPoolTemplatesLocal(templatesDir string) ([]entity.Template, error) {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []entity.Template{}, nil
		}
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	templates := make([]entity.Template, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(templatesDir, entry.Name()))
		if err != nil {
			continue
		}

		var template entity.Template
		if err := yaml.Unmarshal(data, &template); err != nil {
			continue
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// listPoolTemplatesRemote 远程列举模板
func (s *TemplateStore) listPoolTemplatesRemote(client libvirt.LibvirtClient, templatesDir string) ([]entity.Template, error) {
	// 获取目录中的所有 yaml 文件
	files, err := client.ListRemoteFiles(templatesDir, "*.yaml")
	if err != nil {
		// 目录不存在返回空列表
		return []entity.Template{}, nil
	}

	templates := make([]entity.Template, 0, len(files))
	for _, file := range files {
		data, err := client.ReadRemoteFile(filepath.Join(templatesDir, file))
		if err != nil {
			continue
		}

		var template entity.Template
		if err := yaml.Unmarshal(data, &template); err != nil {
			continue
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// Get 获取模板
func (s *TemplateStore) Get(ctx context.Context, nodeName, poolName, templateID string) (*entity.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	poolPath, err := s.getPoolPath(ctx, nodeName, poolName)
	if err != nil {
		return nil, err
	}

	metadataPath := s.getMetadataPath(poolPath, templateID)

	client, err := s.nodeStorageFn(ctx, nodeName)
	if err != nil {
		return nil, fmt.Errorf("get node client: %w", err)
	}

	var data []byte
	if client.IsRemoteConnection() {
		data, err = client.ReadRemoteFile(metadataPath)
	} else {
		data, err = os.ReadFile(metadataPath)
	}

	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("read template metadata: %w", err)
	}

	var template entity.Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("unmarshal template metadata: %w", err)
	}

	return &template, nil
}

// Delete 删除模板元数据
func (s *TemplateStore) Delete(ctx context.Context, nodeName, poolName, templateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	poolPath, err := s.getPoolPath(ctx, nodeName, poolName)
	if err != nil {
		return err
	}

	metadataPath := s.getMetadataPath(poolPath, templateID)

	client, err := s.nodeStorageFn(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("get node client: %w", err)
	}

	if client.IsRemoteConnection() {
		return client.ExecuteRemoteCommand(fmt.Sprintf("rm -f '%s'", metadataPath))
	}

	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete template metadata: %w", err)
	}

	return nil
}
