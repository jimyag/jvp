package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/libvirt"
)

// NodeConfig 节点配置（用于持久化存储）
type NodeConfig struct {
	Name      string           `json:"name"`
	URI       string           `json:"uri"`
	Type      entity.NodeType  `json:"type"`
	State     entity.NodeState `json:"state"` // 节点状态（用于手动禁用/启用）
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// NodeStorage 节点存储
type NodeStorage struct {
	storageDir string
	mu         sync.RWMutex
	// 连接池：为每个 node 缓存 libvirt 连接
	connections map[string]*libvirt.Client
}

// NewNodeStorage 创建节点存储
func NewNodeStorage(dataDir string) (*NodeStorage, error) {
	storageDir := filepath.Join(dataDir, "nodes")
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create nodes directory: %w", err)
	}

	return &NodeStorage{
		storageDir:  storageDir,
		connections: make(map[string]*libvirt.Client),
	}, nil
}

// getConfigPath 获取节点配置文件路径
func (s *NodeStorage) getConfigPath(nodeName string) string {
	return filepath.Join(s.storageDir, nodeName+".json")
}

// Save 保存节点配置
func (s *NodeStorage) Save(config *NodeConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configPath := s.getConfigPath(config.Name)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal node config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write node config: %w", err)
	}

	return nil
}

// Get 获取节点配置（不加锁版本，内部使用）
func (s *NodeStorage) get(nodeName string) (*NodeConfig, error) {
	configPath := s.getConfigPath(nodeName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("node %s not found", nodeName)
		}
		return nil, fmt.Errorf("failed to read node config: %w", err)
	}

	var config NodeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node config: %w", err)
	}

	return &config, nil
}

// Get 获取节点配置
func (s *NodeStorage) Get(nodeName string) (*NodeConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.get(nodeName)
}

// List 列出所有节点配置
func (s *NodeStorage) List() ([]*NodeConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read nodes directory: %w", err)
	}

	configs := make([]*NodeConfig, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		nodeName := entry.Name()[:len(entry.Name())-5] // 去掉 .json
		config, err := s.get(nodeName)
		if err != nil {
			continue // 跳过无法读取的配置
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// EnsureDefaultLocal 确保存在本地节点配置（用于首次启动自动添加）
func (s *NodeStorage) EnsureDefaultLocal() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已有节点配置则不创建
	configs, err := s.listUnlocked()
	if err != nil {
		return err
	}
	if len(configs) > 0 {
		return nil
	}

	cfg := &NodeConfig{
		Name:      "local",
		URI:       "qemu:///system",
		Type:      entity.NodeTypeLocal,
		State:     entity.NodeStateOnline,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal default node config: %w", err)
	}

	configPath := s.getConfigPath(cfg.Name)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write default node config: %w", err)
	}
	return nil
}

func (s *NodeStorage) listUnlocked() ([]*NodeConfig, error) {
	entries, err := os.ReadDir(s.storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read nodes directory: %w", err)
	}

	configs := make([]*NodeConfig, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		nodeName := entry.Name()[:len(entry.Name())-5] // 去掉 .json
		config, err := s.get(nodeName)
		if err != nil {
			continue // 跳过无法读取的配置
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// getConnectionUnlocked assumes caller holds lock
func (s *NodeStorage) getConnectionUnlocked(cfg *NodeConfig) (*libvirt.Client, error) {
	if conn, ok := s.connections[cfg.Name]; ok {
		return conn, nil
	}
	conn, err := libvirt.NewWithURI(cfg.URI)
	if err != nil {
		return nil, err
	}
	s.connections[cfg.Name] = conn
	return conn, nil
}

// Delete 删除节点配置
func (s *NodeStorage) Delete(nodeName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 关闭并移除连接
	if conn, ok := s.connections[nodeName]; ok {
		// libvirt 连接不需要显式关闭
		_ = conn
		delete(s.connections, nodeName)
	}

	configPath := s.getConfigPath(nodeName)
	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("node %s not found", nodeName)
		}
		return fmt.Errorf("failed to delete node config: %w", err)
	}

	return nil
}

// GetConnection 获取或创建节点的 libvirt 连接
func (s *NodeStorage) GetConnection(nodeName string) (*libvirt.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查缓存
	if conn, ok := s.connections[nodeName]; ok {
		return conn, nil
	}

	// 获取配置（使用不加锁的版本）
	config, err := s.get(nodeName)
	if err != nil {
		return nil, err
	}

	// 创建新连接
	conn, err := libvirt.NewWithURI(config.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node %s: %w", nodeName, err)
	}

	// 缓存连接
	s.connections[nodeName] = conn

	return conn, nil
}

// Exists 检查节点是否存在
func (s *NodeStorage) Exists(nodeName string) bool {
	configPath := s.getConfigPath(nodeName)
	_, err := os.Stat(configPath)
	return err == nil
}
