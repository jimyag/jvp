package metadata

import (
	"context"
	"fmt"
	"os"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/rs/zerolog/log"
)

// ==================== KeyPair 管理器 ====================

// KeyPairManager 密钥对管理器
type KeyPairManager struct {
	keypairsDir string
}

// NewKeyPairManager 创建密钥对管理器
func NewKeyPairManager(keypairsDir string) *KeyPairManager {
	return &KeyPairManager{
		keypairsDir: keypairsDir,
	}
}

// SaveKeyPair 保存密钥对元数据
func (s *LibvirtMetadataStore) SaveKeyPair(ctx context.Context, keyPair *entity.KeyPair) error {
	log.Debug().
		Str("keypair_id", keyPair.ID).
		Str("name", keyPair.Name).
		Msg("Saving keypair metadata")

	// 1. 构建密钥对元数据
	meta := KeyPairMetadata{
		Version:       "1.0",
		SchemaVersion: "1.0",
		ResourceType:  "keypair",
		ID:            keyPair.ID,
		Name:          keyPair.Name,
		Algorithm:     keyPair.Algorithm,
		Fingerprint:   keyPair.Fingerprint,
		PublicKey:     keyPair.PublicKey,
		CreatedAt:     keyPair.CreatedAt,
	}

	// 2. 保存元数据文件
	metaPath := getKeyPairMetadataPath(s.config.BasePath, keyPair.ID)
	if err := saveJSONFile(metaPath, meta); err != nil {
		return fmt.Errorf("save keypair metadata: %w", err)
	}

	// 3. 保存公钥文件(单独存储,方便注入实例)
	pubKeyPath := getKeyPairPublicKeyPath(s.config.BasePath, keyPair.ID)
	if err := os.WriteFile(pubKeyPath, []byte(keyPair.PublicKey), 0644); err != nil {
		return fmt.Errorf("save public key: %w", err)
	}

	// 4. 更新内存索引
	s.updateKeyPairIndex(keyPair)

	log.Info().
		Str("keypair_id", keyPair.ID).
		Str("meta_path", metaPath).
		Msg("KeyPair metadata saved successfully")

	return nil
}

// GetKeyPair 获取单个密钥对
func (s *LibvirtMetadataStore) GetKeyPair(ctx context.Context, keyPairID string) (*entity.KeyPair, error) {
	log.Debug().Str("keypair_id", keyPairID).Msg("Getting keypair")

	// 1. 读取元数据文件
	metaPath := getKeyPairMetadataPath(s.config.BasePath, keyPairID)
	if !fileExists(metaPath) {
		return nil, fmt.Errorf("keypair not found: %s", keyPairID)
	}

	var meta KeyPairMetadata
	if err := loadJSONFile(metaPath, &meta); err != nil {
		return nil, fmt.Errorf("load keypair metadata: %w", err)
	}

	// 2. 构建密钥对对象
	keyPair := &entity.KeyPair{
		ID:          meta.ID,
		Name:        meta.Name,
		Algorithm:   meta.Algorithm,
		Fingerprint: meta.Fingerprint,
		PublicKey:   meta.PublicKey,
		CreatedAt:   meta.CreatedAt,
	}

	return keyPair, nil
}

// ListKeyPairs 列出所有密钥对
func (s *LibvirtMetadataStore) ListKeyPairs(ctx context.Context) ([]*entity.KeyPair, error) {
	log.Debug().Msg("Listing all keypairs")

	s.index.RLock()
	keyPairIDs := make([]string, 0, len(s.index.KeyPairs))
	for id := range s.index.KeyPairs {
		keyPairIDs = append(keyPairIDs, id)
	}
	s.index.RUnlock()

	keyPairs := make([]*entity.KeyPair, 0, len(keyPairIDs))
	for _, id := range keyPairIDs {
		keyPair, err := s.GetKeyPair(ctx, id)
		if err != nil {
			log.Warn().Str("keypair_id", id).Err(err).Msg("Failed to get keypair")
			continue
		}
		keyPairs = append(keyPairs, keyPair)
	}

	log.Debug().Int("count", len(keyPairs)).Msg("Listed keypairs")
	return keyPairs, nil
}

// DescribeKeyPairs 查询密钥对(支持过滤)
func (s *LibvirtMetadataStore) DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]*entity.KeyPair, error) {
	log.Debug().
		Strs("keypair_ids", req.KeyPairIDs).
		Interface("filters", req.Filters).
		Msg("Describing keypairs")

	// 如果指定了密钥对 ID,直接查询
	if len(req.KeyPairIDs) > 0 {
		return s.getKeyPairsByIDs(ctx, req.KeyPairIDs)
	}

	// 否则,获取所有密钥对并过滤
	allKeyPairs, err := s.ListKeyPairs(ctx)
	if err != nil {
		return nil, err
	}

	// 应用过滤条件
	keyPairs := make([]*entity.KeyPair, 0)
	for _, kp := range allKeyPairs {
		if s.matchesKeyPairFilters(kp, req.Filters) {
			keyPairs = append(keyPairs, kp)
		}
	}

	log.Debug().Int("count", len(keyPairs)).Msg("Described keypairs")
	return keyPairs, nil
}

// DeleteKeyPair 删除密钥对元数据
func (s *LibvirtMetadataStore) DeleteKeyPair(ctx context.Context, keyPairID string) error {
	log.Debug().Str("keypair_id", keyPairID).Msg("Deleting keypair metadata")

	// 1. 删除元数据文件
	metaPath := getKeyPairMetadataPath(s.config.BasePath, keyPairID)
	if err := deleteJSONFile(metaPath); err != nil {
		log.Warn().Err(err).Msg("Failed to delete keypair metadata file")
	}

	// 2. 删除公钥文件
	pubKeyPath := getKeyPairPublicKeyPath(s.config.BasePath, keyPairID)
	if err := os.Remove(pubKeyPath); err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msg("Failed to delete public key file")
	}

	// 3. 从索引中删除
	s.removeKeyPairFromIndex(keyPairID)

	log.Info().Str("keypair_id", keyPairID).Msg("KeyPair metadata deleted")
	return nil
}

// GetKeyPairPublicKey 获取密钥对的公钥(用于注入实例)
func (s *LibvirtMetadataStore) GetKeyPairPublicKey(ctx context.Context, keyPairID string) (string, error) {
	log.Debug().Str("keypair_id", keyPairID).Msg("Getting keypair public key")

	// 直接读取公钥文件
	pubKeyPath := getKeyPairPublicKeyPath(s.config.BasePath, keyPairID)
	if !fileExists(pubKeyPath) {
		return "", fmt.Errorf("public key not found for keypair: %s", keyPairID)
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("read public key: %w", err)
	}

	return string(data), nil
}

// ==================== 辅助函数 ====================

// getKeyPairsByIDs 根据密钥对 ID 列表获取密钥对
func (s *LibvirtMetadataStore) getKeyPairsByIDs(ctx context.Context, keyPairIDs []string) ([]*entity.KeyPair, error) {
	keyPairs := make([]*entity.KeyPair, 0, len(keyPairIDs))

	for _, id := range keyPairIDs {
		keyPair, err := s.GetKeyPair(ctx, id)
		if err != nil {
			log.Warn().Str("keypair_id", id).Err(err).Msg("Failed to get keypair")
			continue
		}
		keyPairs = append(keyPairs, keyPair)
	}

	return keyPairs, nil
}

// matchesKeyPairFilters 检查密钥对是否匹配过滤条件
func (s *LibvirtMetadataStore) matchesKeyPairFilters(keyPair *entity.KeyPair, filters []entity.Filter) bool {
	for _, filter := range filters {
		switch filter.Name {
		case "keypair-id":
			matched := false
			for _, value := range filter.Values {
				if keyPair.ID == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "key-name":
			matched := false
			for _, value := range filter.Values {
				if keyPair.Name == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		case "fingerprint":
			matched := false
			for _, value := range filter.Values {
				if keyPair.Fingerprint == value {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}

		// tag filter removed - entity.KeyPair doesn't support Tags
		}
	}

	return true
}

// updateKeyPairIndex 更新密钥对索引
func (s *LibvirtMetadataStore) updateKeyPairIndex(keyPair *entity.KeyPair) {
	s.index.Lock()
	defer s.index.Unlock()

	idx := &KeyPairIndexEntry{
		ID:          keyPair.ID,
		Name:        keyPair.Name,
		Fingerprint: keyPair.Fingerprint,
	}

	s.index.KeyPairs[keyPair.ID] = idx
}

// removeKeyPairFromIndex 从索引中删除密钥对
func (s *LibvirtMetadataStore) removeKeyPairFromIndex(keyPairID string) {
	s.index.Lock()
	defer s.index.Unlock()

	delete(s.index.KeyPairs, keyPairID)
}
