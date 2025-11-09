package main

import (
	"context"

	_ "github.com/jimmicro/version"
	"github.com/jimyag/jvp/internal/jvp"
	"github.com/jimyag/jvp/internal/jvp/config"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create config")
	}
	server, err := jvp.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}
	if err := server.Run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to run server")
	}
}
