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
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/jimyag/jvp/pkg/idgen"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

// KeyPairService 密钥对服务
type KeyPairService struct {
	keyPairRepo repository.KeyPairRepository
	idGen       *idgen.Generator
}

// NewKeyPairService 创建密钥对服务
func NewKeyPairService(
	repo *repository.Repository,
) *KeyPairService {
	return &KeyPairService{
		keyPairRepo: repository.NewKeyPairRepository(repo.DB()),
		idGen:       idgen.New(),
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

	// 创建数据库模型
	keyPairModel := &model.KeyPair{
		ID:          keyPairID,
		Name:        req.Name,
		Algorithm:   algorithm,
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存到数据库
	if err := s.keyPairRepo.Create(ctx, keyPairModel); err != nil {
		logger.Error().Err(err).Msg("Failed to save keypair to database")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save keypair to database", err)
	}

	// 转换为 entity
	keyPair := &entity.KeyPair{
		ID:          keyPairModel.ID,
		Name:        keyPairModel.Name,
		Algorithm:   keyPairModel.Algorithm,
		PublicKey:   keyPairModel.PublicKey,
		Fingerprint: keyPairModel.Fingerprint,
		CreatedAt:   keyPairModel.CreatedAt.Format(time.RFC3339),
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

	// 创建数据库模型
	keyPairModel := &model.KeyPair{
		ID:          keyPairID,
		Name:        req.Name,
		Algorithm:   algorithm,
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存到数据库
	if err := s.keyPairRepo.Create(ctx, keyPairModel); err != nil {
		logger.Error().Err(err).Msg("Failed to save keypair to database")
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to save keypair to database", err)
	}

	// 转换为 entity
	keyPair := &entity.KeyPair{
		ID:          keyPairModel.ID,
		Name:        keyPairModel.Name,
		Algorithm:   keyPairModel.Algorithm,
		PublicKey:   keyPairModel.PublicKey,
		Fingerprint: keyPairModel.Fingerprint,
		CreatedAt:   keyPairModel.CreatedAt.Format(time.RFC3339),
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
	_, err := s.keyPairRepo.GetByID(ctx, keyPairID)
	if err != nil {
		logger.Error().Err(err).Str("keypair_id", keyPairID).Msg("Key pair not found")
		return apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("keypair %s not found", keyPairID),
			404,
		)
	}

	// 软删除
	if err := s.keyPairRepo.Delete(ctx, keyPairID); err != nil {
		logger.Error().Err(err).Str("keypair_id", keyPairID).Msg("Failed to delete keypair")
		return apierror.WrapError(apierror.ErrInternalError, "Failed to delete keypair", err)
	}

	logger.Info().Str("keypair_id", keyPairID).Msg("Key pair deleted successfully")
	return nil
}

// DescribeKeyPairs 描述密钥对
func (s *KeyPairService) DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]entity.KeyPair, error) {
	logger := zerolog.Ctx(ctx)

	// 构建过滤器
	filters := make(map[string]interface{})

	// 如果指定了 KeyPairIDs，需要特殊处理（因为 List 方法不支持 ID 列表）
	var keyPairs []*model.KeyPair
	if req.KeyPairIDs != nil {
		// 如果 KeyPairIDs 不为 nil，说明用户明确指定了要查询的 ID 列表
		if len(req.KeyPairIDs) > 0 {
			// 逐个查询
			for _, id := range req.KeyPairIDs {
				kp, err := s.keyPairRepo.GetByID(ctx, id)
				if err != nil {
					// 忽略不存在的密钥对
					continue
				}
				keyPairs = append(keyPairs, kp)
			}
		}
		// 如果 KeyPairIDs 为空数组，返回空结果（用户明确指定了空列表）
	} else {
		// KeyPairIDs 为 nil，表示查询所有密钥对
		// 应用其他过滤器
		// 注意：这里简化处理，实际可以根据 Filters 字段扩展
		var err error
		keyPairs, err = s.keyPairRepo.List(ctx, filters)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list keypairs")
			return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to list keypairs", err)
		}
	}

	// 转换为 entity
	result := make([]entity.KeyPair, 0, len(keyPairs))
	for _, kp := range keyPairs {
		result = append(result, entity.KeyPair{
			ID:          kp.ID,
			Name:        kp.Name,
			Algorithm:   kp.Algorithm,
			PublicKey:   kp.PublicKey,
			Fingerprint: kp.Fingerprint,
			CreatedAt:   kp.CreatedAt.Format(time.RFC3339),
		})
	}

	logger.Info().Int("count", len(result)).Msg("Key pairs described successfully")
	return result, nil
}

// GetKeyPairByID 根据 ID 获取密钥对（内部方法）
func (s *KeyPairService) GetKeyPairByID(ctx context.Context, keyPairID string) (*entity.KeyPair, error) {
	kp, err := s.keyPairRepo.GetByID(ctx, keyPairID)
	if err != nil {
		return nil, apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("keypair %s not found", keyPairID),
			404,
		)
	}

	return &entity.KeyPair{
		ID:          kp.ID,
		Name:        kp.Name,
		Algorithm:   kp.Algorithm,
		PublicKey:   kp.PublicKey,
		Fingerprint: kp.Fingerprint,
		CreatedAt:   kp.CreatedAt.Format(time.RFC3339),
	}, nil
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
