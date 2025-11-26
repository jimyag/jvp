package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

// KeyPairService 密钥对服务（使用文件系统存储）
type KeyPairService struct {
	idGen      *idgen.Generator
	storageDir string // 存储目录，默认 ~/.jvp/keypairs
}

// NewKeyPairService 创建密钥对服务
func NewKeyPairService() (*KeyPairService, error) {
	// 默认存储目录为 ~/.jvp/keypairs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home directory: %w", err)
	}

	storageDir := filepath.Join(homeDir, ".jvp", "keypairs")

	// 确保目录存在
	if err := os.MkdirAll(storageDir, 0o700); err != nil {
		return nil, fmt.Errorf("create keypair storage directory: %w", err)
	}

	return &KeyPairService{
		idGen:      idgen.New(),
		storageDir: storageDir,
	}, nil
}

// CreateKeyPair 创建密钥对
func (s *KeyPairService) CreateKeyPair(ctx context.Context, req *entity.CreateKeyPairRequest) (*entity.CreateKeyPairResponse, error) {
	logger := zerolog.Ctx(ctx)

	// 设置默认算法
	algorithm := req.Algorithm
	if algorithm == "" {
		algorithm = "ed25519"
	}
	if algorithm != "rsa" && algorithm != "ed25519" {
		return nil, apierror.NewErrorWithStatus(
			"InvalidParameter",
			fmt.Sprintf("unsupported algorithm: %s, supported: rsa, ed25519", algorithm),
			400,
		)
	}

	// 设置 RSA 密钥长度默认值
	keySize := req.KeySize
	if algorithm == "rsa" && keySize == 0 {
		keySize = 2048
	}
	if algorithm == "rsa" && keySize < 2048 {
		return nil, apierror.NewErrorWithStatus(
			"InvalidParameter",
			"RSA key size must be at least 2048 bits",
			400,
		)
	}

	// 生成密钥对
	var publicKeyStr string
	var privateKeyStr string
	var fingerprint string
	var err error

	if algorithm == "ed25519" {
		publicKeyStr, privateKeyStr, fingerprint, err = s.generateED25519KeyPair()
	} else {
		publicKeyStr, privateKeyStr, fingerprint, err = s.generateRSAKeyPair(keySize)
	}

	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate key pair")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate key pair", err)
	}

	// 生成密钥对 ID
	keyPairID, err := s.idGen.GenerateKeyPairID()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate keypair ID")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate keypair ID", err)
	}

	// 创建 entity
	keyPair := &entity.KeyPair{
		ID:          keyPairID,
		Name:        req.Name,
		Algorithm:   algorithm,
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// 保存到文件系统
	if err := s.saveKeyPairToFile(keyPair); err != nil {
		logger.Error().Err(err).Msg("Failed to save keypair to file")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save keypair to file", err)
	}

	logger.Info().
		Str("keypair_id", keyPair.ID).
		Str("name", keyPair.Name).
		Str("algorithm", keyPair.Algorithm).
		Msg("Key pair created successfully")

	return &entity.CreateKeyPairResponse{
		KeyPair:    keyPair,
		PrivateKey: privateKeyStr,
	}, nil
}

// ImportKeyPair 导入密钥对
func (s *KeyPairService) ImportKeyPair(ctx context.Context, req *entity.ImportKeyPairRequest) (*entity.ImportKeyPairResponse, error) {
	logger := zerolog.Ctx(ctx)

	// 解析和验证公钥格式
	publicKey, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse public key")
		return nil, apierror.NewErrorWithStatus(
			"InvalidParameter",
			fmt.Sprintf("invalid public key format: %v", err),
			400,
		)
	}

	// 检查是否有未解析的内容
	if len(rest) > 0 {
		return nil, apierror.NewErrorWithStatus(
			"InvalidParameter",
			"public key contains extra data",
			400,
		)
	}

	// 忽略 comment 和 options，只使用公钥本身
	_ = comment
	_ = options

	// 格式化公钥为 OpenSSH 格式
	publicKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))

	// 计算公钥指纹
	fingerprint := s.calculateFingerprint(publicKey)

	// 确定算法类型
	algorithm := s.determineAlgorithm(publicKey.Type())

	// 生成密钥对 ID
	keyPairID, err := s.idGen.GenerateKeyPairID()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate keypair ID")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to generate keypair ID", err)
	}

	// 创建 entity
	keyPair := &entity.KeyPair{
		ID:          keyPairID,
		Name:        req.Name,
		Algorithm:   algorithm,
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// 保存到文件系统
	if err := s.saveKeyPairToFile(keyPair); err != nil {
		logger.Error().Err(err).Msg("Failed to save keypair to file")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save keypair to file", err)
	}

	logger.Info().
		Str("keypair_id", keyPair.ID).
		Str("name", keyPair.Name).
		Str("algorithm", keyPair.Algorithm).
		Msg("Key pair imported successfully")

	return &entity.ImportKeyPairResponse{
		KeyPair: keyPair,
	}, nil
}

// DeleteKeyPair 删除密钥对
func (s *KeyPairService) DeleteKeyPair(ctx context.Context, keyPairID string) error {
	logger := zerolog.Ctx(ctx)

	// 检查密钥对是否存在
	_, err := s.loadKeyPairFromFile(keyPairID)
	if err != nil {
		logger.Error().Err(err).Str("keypair_id", keyPairID).Msg("Key pair not found")
		return apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("keypair %s not found", keyPairID),
			404,
		)
	}

	// 删除文件
	filePath := filepath.Join(s.storageDir, keyPairID+".json")
	if err := os.Remove(filePath); err != nil {
		logger.Error().Err(err).Str("keypair_id", keyPairID).Msg("Failed to delete keypair file")
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete keypair file", err)
	}

	logger.Info().Str("keypair_id", keyPairID).Msg("Key pair deleted successfully")
	return nil
}

// DescribeKeyPairs 描述密钥对
func (s *KeyPairService) DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]entity.KeyPair, error) {
	logger := zerolog.Ctx(ctx)

	// 读取所有 JSON 文件
	files, err := filepath.Glob(filepath.Join(s.storageDir, "*.json"))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list keypair files")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to list keypair files", err)
	}

	var result []entity.KeyPair
	for _, file := range files {
		keyPair, err := s.loadKeyPairFromFileByPath(file)
		if err != nil {
			logger.Warn().Err(err).Str("file", file).Msg("Failed to load keypair file, skipping")
			continue
		}

		// 应用过滤器
		if req != nil && len(req.KeyPairIDs) > 0 {
			found := false
			for _, id := range req.KeyPairIDs {
				if keyPair.ID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		result = append(result, *keyPair)
	}

	logger.Info().Int("count", len(result)).Msg("Key pairs described successfully")
	return result, nil
}

// GetKeyPairByID 根据 ID 获取密钥对（内部方法）
func (s *KeyPairService) GetKeyPairByID(ctx context.Context, keyPairID string) (*entity.KeyPair, error) {
	kp, err := s.loadKeyPairFromFile(keyPairID)
	if err != nil {
		return nil, apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("keypair %s not found", keyPairID),
			404,
		)
	}

	return kp, nil
}

// saveKeyPairToFile 保存密钥对到文件
func (s *KeyPairService) saveKeyPairToFile(keyPair *entity.KeyPair) error {
	filePath := filepath.Join(s.storageDir, keyPair.ID+".json")

	data, err := json.MarshalIndent(keyPair, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal keypair: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("write keypair file: %w", err)
	}

	return nil
}

// loadKeyPairFromFile 从文件加载密钥对
func (s *KeyPairService) loadKeyPairFromFile(keyPairID string) (*entity.KeyPair, error) {
	filePath := filepath.Join(s.storageDir, keyPairID+".json")
	return s.loadKeyPairFromFileByPath(filePath)
}

// loadKeyPairFromFileByPath 从文件路径加载密钥对
func (s *KeyPairService) loadKeyPairFromFileByPath(filePath string) (*entity.KeyPair, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read keypair file: %w", err)
	}

	var keyPair entity.KeyPair
	if err := json.Unmarshal(data, &keyPair); err != nil {
		return nil, fmt.Errorf("unmarshal keypair: %w", err)
	}

	return &keyPair, nil
}

// generateED25519KeyPair 生成 ED25519 密钥对
func (s *KeyPairService) generateED25519KeyPair() (publicKeyStr, privateKeyStr, fingerprint string, err error) {
	// 生成密钥对
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// 转换为 SSH 公钥格式
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}
	publicKeyStr = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey)))

	// 计算指纹
	fingerprint = s.calculateFingerprint(sshPublicKey)

	// 转换为 OpenSSH 私钥格式
	privateKeyBytes, err := s.marshalED25519PrivateKey(privateKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}
	privateKeyStr = string(privateKeyBytes)

	return publicKeyStr, privateKeyStr, fingerprint, nil
}

// generateRSAKeyPair 生成 RSA 密钥对
func (s *KeyPairService) generateRSAKeyPair(keySize int) (publicKeyStr, privateKeyStr, fingerprint string, err error) {
	// 生成 RSA 密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// 转换为 SSH 公钥格式
	sshPublicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}
	publicKeyStr = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey)))

	// 计算指纹
	fingerprint = s.calculateFingerprint(sshPublicKey)

	// 转换为 OpenSSH 私钥格式
	privateKeyBytes, err := s.marshalRSAPrivateKey(privateKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}
	privateKeyStr = string(privateKeyBytes)

	return publicKeyStr, privateKeyStr, fingerprint, nil
}

// calculateFingerprint 计算公钥指纹（SHA256）
func (s *KeyPairService) calculateFingerprint(publicKey ssh.PublicKey) string {
	hash := sha256.Sum256(publicKey.Marshal())
	return fmt.Sprintf("SHA256:%s", base64.RawStdEncoding.EncodeToString(hash[:]))
}

// determineAlgorithm 根据 SSH 密钥类型确定算法名称
func (s *KeyPairService) determineAlgorithm(keyType string) string {
	switch keyType {
	case ssh.KeyAlgoRSA, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512:
		return "rsa"
	case ssh.KeyAlgoED25519:
		return "ed25519"
	default:
		return strings.ToLower(keyType)
	}
}

// marshalED25519PrivateKey 将 ED25519 私钥转换为 OpenSSH 格式
func (s *KeyPairService) marshalED25519PrivateKey(privateKey ed25519.PrivateKey) ([]byte, error) {
	// 使用标准的 PEM 格式存储完整的私钥（64 字节）
	// ED25519 私钥是 64 字节（32 字节种子 + 32 字节公钥）
	privateKeyBytes := make([]byte, len(privateKey))
	copy(privateKeyBytes, privateKey)

	block := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	return pem.EncodeToMemory(block), nil
}

// marshalRSAPrivateKey 将 RSA 私钥转换为 OpenSSH 格式
func (s *KeyPairService) marshalRSAPrivateKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	// 使用 PKCS#1 格式
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	}

	return pem.EncodeToMemory(block), nil
}
