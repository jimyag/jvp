package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/metadata"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

// KeyPairService 密钥对服务
type KeyPairService struct {
	metadataStore metadata.KeyPairStore
	idGen         *idgen.Generator
}

// NewKeyPairService 创建密钥对服务
func NewKeyPairService(
	metadataStore metadata.KeyPairStore,
) *KeyPairService {
	return &KeyPairService{
		metadataStore: metadataStore,
		idGen:         idgen.New(),
	}
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

	// 保存到 metadata store
	if err := s.metadataStore.SaveKeyPair(ctx, keyPair); err != nil {
		logger.Error().Err(err).Msg("Failed to save keypair to metadata store")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save keypair to metadata store", err)
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

	// 保存到 metadata store
	if err := s.metadataStore.SaveKeyPair(ctx, keyPair); err != nil {
		logger.Error().Err(err).Msg("Failed to save keypair to metadata store")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save keypair to metadata store", err)
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
	_, err := s.metadataStore.GetKeyPair(ctx, keyPairID)
	if err != nil {
		logger.Error().Err(err).Str("keypair_id", keyPairID).Msg("Key pair not found")
		return apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("keypair %s not found", keyPairID),
			404,
		)
	}

	// 删除
	if err := s.metadataStore.DeleteKeyPair(ctx, keyPairID); err != nil {
		logger.Error().Err(err).Str("keypair_id", keyPairID).Msg("Failed to delete keypair")
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete keypair", err)
	}

	logger.Info().Str("keypair_id", keyPairID).Msg("Key pair deleted successfully")
	return nil
}

// DescribeKeyPairs 描述密钥对
func (s *KeyPairService) DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]entity.KeyPair, error) {
	logger := zerolog.Ctx(ctx)

	// 从 metadata store 查询
	keyPairPtrs, err := s.metadataStore.DescribeKeyPairs(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to describe keypairs from metadata store")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to describe keypairs from metadata store", err)
	}

	// 转换为值类型
	result := make([]entity.KeyPair, 0, len(keyPairPtrs))
	for _, kp := range keyPairPtrs {
		if kp != nil {
			result = append(result, *kp)
		}
	}

	logger.Info().Int("count", len(result)).Msg("Key pairs described successfully")
	return result, nil
}

// GetKeyPairByID 根据 ID 获取密钥对（内部方法）
func (s *KeyPairService) GetKeyPairByID(ctx context.Context, keyPairID string) (*entity.KeyPair, error) {
	kp, err := s.metadataStore.GetKeyPair(ctx, keyPairID)
	if err != nil {
		return nil, apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("keypair %s not found", keyPairID),
			404,
		)
	}

	return kp, nil
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
