package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"gopkg.in/yaml.v3"
)

// TemplateStore 负责模板元数据的本地持久化
type TemplateStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewTemplateStore 创建模板存储
func NewTemplateStore(dataDir string) (*TemplateStore, error) {
	baseDir := filepath.Join(dataDir, "metadata", "templates")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create template metadata dir: %w", err)
	}
	return &TemplateStore{baseDir: baseDir}, nil
}

func (s *TemplateStore) nodeDir(nodeName string) string {
	return filepath.Join(s.baseDir, nodeName)
}

func (s *TemplateStore) templateDir(nodeName, templateID string) string {
	return filepath.Join(s.nodeDir(nodeName), templateID)
}

func (s *TemplateStore) metadataPath(nodeName, templateID string) string {
	return filepath.Join(s.templateDir(nodeName, templateID), "metadata.yaml")
}

// Save 保存模板元数据
func (s *TemplateStore) Save(template *entity.Template) error {
	if template == nil {
		return errors.New("template is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.templateDir(template.NodeName, template.ID), 0o755); err != nil {
		return fmt.Errorf("ensure template directory: %w", err)
	}

	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("marshal template metadata: %w", err)
	}

	tmpPath := s.metadataPath(template.NodeName, template.ID) + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write template metadata: %w", err)
	}

	if err := os.Rename(tmpPath, s.metadataPath(template.NodeName, template.ID)); err != nil {
		return fmt.Errorf("finalize template metadata: %w", err)
	}
	return nil
}

// List 列举模板
func (s *TemplateStore) List(nodeName string) ([]entity.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := []string{}
	if nodeName != "" {
		nodes = append(nodes, nodeName)
	} else {
		entries, err := os.ReadDir(s.baseDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []entity.Template{}, nil
			}
			return nil, fmt.Errorf("read template metadata dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				nodes = append(nodes, entry.Name())
			}
		}
	}

	results := make([]entity.Template, 0)
	for _, node := range nodes {
		nodeTemplates, err := s.listNodeTemplatesLocked(node)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		results = append(results, nodeTemplates...)
	}
	return results, nil
}

func (s *TemplateStore) listNodeTemplatesLocked(nodeName string) ([]entity.Template, error) {
	dir := s.nodeDir(nodeName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []entity.Template{}, nil
		}
		return nil, err
	}

	templates := make([]entity.Template, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		template, err := s.loadTemplateLocked(nodeName, entry.Name())
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}
	return templates, nil
}

// Get 获取模板
func (s *TemplateStore) Get(nodeName, templateID string) (*entity.Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadTemplateLocked(nodeName, templateID)
}

func (s *TemplateStore) loadTemplateLocked(nodeName, templateID string) (*entity.Template, error) {
	data, err := os.ReadFile(s.metadataPath(nodeName, templateID))
	if err != nil {
		return nil, err
	}

	var template entity.Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("unmarshal template metadata: %w", err)
	}
	return &template, nil
}

// Delete 删除模板元数据
func (s *TemplateStore) Delete(nodeName, templateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.templateDir(nodeName, templateID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete template metadata: %w", err)
	}
	return nil
}
