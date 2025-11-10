package jvp

import (
	"context"
	"fmt"
	"os"

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
	libvirtClient, err := libvirt.New()
	if err != nil {
		return nil, err
	}

	// 2. 创建 Storage Service
	storageService, err := service.NewStorageService(libvirtClient)
	if err != nil {
		return nil, err
	}

	// 3. 创建 Image Service
	imageService, err := service.NewImageService(storageService, libvirtClient)
	if err != nil {
		return nil, err
	}

	// 3.1. 确保默认镜像存在（如果不存在则下载）
	// 阻塞启动，等待镜像下载完成
	ctx := context.Background()
	logger.Info().Msg("Ensuring default images exist...")
	if err := imageService.EnsureDefaultImages(ctx); err != nil {
		return nil, fmt.Errorf("ensure default images: %w", err)
	}
	logger.Info().Msg("All default images are ready")

	// 4. 创建 Instance Service
	instanceService, err := service.NewInstanceService(storageService, imageService, libvirtClient)
	if err != nil {
		return nil, err
	}

	// 5. 创建 API
	apiInstance, err := api.New(instanceService)
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
	return s.api.Run(ctx)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.api.Shutdown(ctx)
}
