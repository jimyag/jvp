package jvp

import (
	"context"
	"os"

	"github.com/jimyag/jvp/internal/jvp/api"
	"github.com/jimyag/jvp/internal/jvp/config"
	"github.com/rs/zerolog"
)

type Server struct {
	cfg *config.Config
	api *api.API
}

func New(cfg *config.Config) (*Server, error) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger

	api, err := api.New()
	if err != nil {
		return nil, err
	}
	server := &Server{
		cfg: cfg,
		api: api,
	}
	return server, nil
}
func (s *Server) Run(ctx context.Context) error {
	return s.api.Run(ctx)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.api.Shutdown(ctx)
}
