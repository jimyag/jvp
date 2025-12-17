package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// KeyPairServiceInterface 定义密钥对服务的接口
type KeyPairServiceInterface interface {
	CreateKeyPair(ctx context.Context, req *entity.CreateKeyPairRequest) (*entity.CreateKeyPairResponse, error)
	ImportKeyPair(ctx context.Context, req *entity.ImportKeyPairRequest) (*entity.ImportKeyPairResponse, error)
	DeleteKeyPair(ctx context.Context, keyPairID string) error
	DescribeKeyPairs(ctx context.Context, req *entity.DescribeKeyPairsRequest) ([]entity.KeyPair, error)
}

type KeyPair struct {
	keyPairService KeyPairServiceInterface
}

func NewKeyPair(keyPairService *service.KeyPairService) *KeyPair {
	return &KeyPair{
		keyPairService: keyPairService,
	}
}

func (k *KeyPair) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create-keypair", ginx.Adapt5(k.CreateKeyPair))
	router.POST("/import-keypair", ginx.Adapt5(k.ImportKeyPair))
	router.POST("/delete-keypair", ginx.Adapt5(k.DeleteKeyPair))
	router.POST("/describe-keypairs", ginx.Adapt5(k.DescribeKeyPairs))
}

func (k *KeyPair) CreateKeyPair(ctx *gin.Context, req *entity.CreateKeyPairRequest) (*entity.CreateKeyPairResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("name", req.Name).
		Str("algorithm", req.Algorithm).
		Msg("CreateKeyPair called")

	response, err := k.keyPairService.CreateKeyPair(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create keypair")
		return nil, err
	}

	logger.Info().
		Str("keypair_id", response.KeyPair.ID).
		Str("name", response.KeyPair.Name).
		Msg("Key pair created successfully")

	return response, nil
}

func (k *KeyPair) ImportKeyPair(ctx *gin.Context, req *entity.ImportKeyPairRequest) (*entity.ImportKeyPairResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("name", req.Name).
		Msg("ImportKeyPair called")

	response, err := k.keyPairService.ImportKeyPair(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to import keypair")
		return nil, err
	}

	logger.Info().
		Str("keypair_id", response.KeyPair.ID).
		Str("name", response.KeyPair.Name).
		Msg("Key pair imported successfully")

	return response, nil
}

func (k *KeyPair) DeleteKeyPair(ctx *gin.Context, req *entity.DeleteKeyPairRequest) (*entity.DeleteKeyPairResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("keypair_id", req.KeyPairID).
		Msg("DeleteKeyPair called")

	err := k.keyPairService.DeleteKeyPair(ctx, req.KeyPairID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("keypair_id", req.KeyPairID).
			Msg("Failed to delete keypair")
		return nil, err
	}

	logger.Info().
		Str("keypair_id", req.KeyPairID).
		Msg("Key pair deleted successfully")

	return &entity.DeleteKeyPairResponse{
		Return: true,
	}, nil
}

func (k *KeyPair) DescribeKeyPairs(ctx *gin.Context, req *entity.DescribeKeyPairsRequest) (*entity.DescribeKeyPairsResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Interface("request", req).
		Msg("DescribeKeyPairs called")

	keyPairs, err := k.keyPairService.DescribeKeyPairs(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to describe keypairs")
		return nil, err
	}

	logger.Info().
		Int("count", len(keyPairs)).
		Msg("Key pairs described successfully")

	return &entity.DescribeKeyPairsResponse{
		KeyPairs: keyPairs,
	}, nil
}
