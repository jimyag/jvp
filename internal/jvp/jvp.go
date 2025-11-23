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
	"github.com/jimyag/jvp/internal/jvp/metadata"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/rs/zerolog"
)

type Server struct {
	cfg           *config.Config
	api           *api.API
	metadataStore metadata.MetadataStore
}

func New(cfg *config.Config) (*Server, error) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger

	// 0. 创建 Metadata Store（基于 Libvirt API，替代 SQLite）
	metadataStore, err := metadata.NewLibvirtMetadataStore(&metadata.StoreConfig{
		BasePath:             "/var/lib/jvp",
		LibvirtURI:           "qemu:///system",
		EnableIndexCache:     true,
		IndexRefreshInterval: 5 * time.Minute,
		LockTimeout:          30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create metadata store: %w", err)
	}

	// 初始化 Metadata Store
	ctx := context.Background()
	if err := metadataStore.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize metadata store: %w", err)
	}
	logger.Info().Msg("Metadata store initialized")

	// 1. 创建 Libvirt Client
	libvirtClient, err := libvirt.New()
	if err != nil {
		return nil, err
	}

	// 2. 创建 Storage Service
	storageService, err := service.NewStorageService(libvirtClient)
	if err != nil {
		return nil, err
	}

	// 3. 创建 Image Service（使用 metadataStore）
	imageService, err := service.NewImageService(storageService, libvirtClient, metadataStore)
	if err != nil {
		return nil, err
	}

	// 3.1. 确保默认镜像存在（如果不存在则下载）
	// 阻塞启动，等待镜像下载完成
	logger.Info().Msg("Ensuring default images exist...")
	if err := imageService.EnsureDefaultImages(ctx); err != nil {
		return nil, fmt.Errorf("ensure default images: %w", err)
	}
	logger.Info().Msg("All default images are ready")

	// 4. 创建 KeyPair Service（使用 metadataStore）
	keyPairService := service.NewKeyPairService(metadataStore)

	// 5. 创建 Instance Service（使用 metadataStore）
	instanceService, err := service.NewInstanceService(storageService, imageService, keyPairService, libvirtClient, metadataStore)
	if err != nil {
		return nil, err
	}

	// 6. 创建 Volume Service（使用 metadataStore）
	volumeService := service.NewVolumeService(storageService, instanceService, libvirtClient, metadataStore)

	// 7. 创建 Snapshot Service（使用 metadataStore）
	snapshotService := service.NewSnapshotService(storageService, libvirtClient, metadataStore)

	// 8. 创建 API
	apiInstance, err := api.New(instanceService, volumeService, snapshotService, imageService, keyPairService)
	if err != nil {
		return nil, err
	}

	server := &Server{
		cfg:           cfg,
		api:           apiInstance,
		metadataStore: metadataStore,
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
