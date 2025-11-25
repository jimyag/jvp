// Package jvp 提供 JVP 服务器的主入口和初始化逻辑
package jvp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jimmicro/grace"
	"github.com/jimyag/jvp/internal/jvp/api"
	"github.com/jimyag/jvp/internal/jvp/config"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/rs/zerolog"
)

type Server struct {
	cfg *config.Config
	api *api.API
}

func New(cfg *config.Config) (*Server, error) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger

	// 1. 创建 Libvirt Client
	logger.Info().Str("uri", cfg.LibvirtURI).Msg("Connecting to libvirt")
	libvirtClient, err := libvirt.NewWithURI(cfg.LibvirtURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt at %s: %w", cfg.LibvirtURI, err)
	}
	logger.Info().Msg("Successfully connected to libvirt")
	logger.Info().Str("data_dir", cfg.DataDir).Msg("Using data directory")

	// 2. 创建 Node Storage
	nodeStorage, err := service.NewNodeStorage(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("create node storage: %w", err)
	}

	// 3. 创建 Node Service
	nodeService, err := service.NewNodeService(nodeStorage)
	if err != nil {
		return nil, err
	}

	// 4. 创建 Storage Service
	storageService, err := service.NewStorageService(libvirtClient, cfg.DataDir)
	if err != nil {
		return nil, err
	}

	// 3. 创建 Image Service（不再使用 metadataStore）
	imageService, err := service.NewImageService(storageService, libvirtClient)
	if err != nil {
		return nil, err
	}

	// 注意：不再在启动时自动下载默认镜像
	// 用户可以通过 API 手动下载需要的镜像
	logger.Info().Msg("Image service initialized. Use API to download images as needed.")

	// 4. 创建 KeyPair Service（使用文件存储）
	keyPairService, err := service.NewKeyPairService()
	if err != nil {
		return nil, fmt.Errorf("create keypair service: %w", err)
	}

	// 5. 创建 Instance Service（不再使用 metadataStore）
	instanceService, err := service.NewInstanceService(storageService, imageService, keyPairService, libvirtClient)
	if err != nil {
		return nil, err
	}

	// 6. 创建 Volume Service（不再使用 metadataStore）
	volumeService := service.NewVolumeService(storageService, instanceService, libvirtClient)

	// 7. 创建 API（添加 nodeService）
	apiInstance, err := api.New(nodeService, instanceService, volumeService, imageService, keyPairService, storageService)
	if err != nil {
		return nil, err
	}

	server := &Server{
		cfg: cfg,
		api: apiInstance,
	}
	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	// 使用 grace.Shepherd 管理服务生命周期
	services := []grace.Grace{
		s.api,
	}

	shepherd := grace.NewShepherd(
		services,
		grace.WithTimeout(30*time.Second),
		grace.WithLogger(&zerologLogger{}),
	)

	shepherd.Start(ctx)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.api.Shutdown(ctx)
}

// Name 实现 grace.Grace 接口
func (s *Server) Name() string {
	return "JVP Server"
}

// zerologLogger 实现 grace.Logger 接口
type zerologLogger struct{}

func (l *zerologLogger) Info(msg string, args ...interface{}) {
	logger := zerolog.DefaultContextLogger.Info()
	// 如果有参数，使用 Msgf 格式化消息
	if len(args) > 0 {
		logger.Msgf(msg, args...)
	} else {
		logger.Msg(msg)
	}
}

func (l *zerologLogger) Error(msg string, args ...interface{}) {
	logger := zerolog.DefaultContextLogger.Error()
	// 如果有参数，使用 Msgf 格式化消息
	if len(args) > 0 {
		logger.Msgf(msg, args...)
	} else {
		logger.Msg(msg)
	}
}
