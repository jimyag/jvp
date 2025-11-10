package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/service"
)

type API struct {
	engine *gin.Engine
	server *http.Server

	instance *Instance
}

func New(instanceService *service.InstanceService) (*API, error) {
	engine := gin.Default()
	api := &API{
		engine:   engine,
		instance: NewInstance(instanceService),
	}
	api.instance.RegisterRoutes(engine.Group("/api"))
	api.server = &http.Server{
		Addr:    ":8080",
		Handler: engine,
	}
	return api, nil
}

func (a *API) Run(ctx context.Context) error {
	// TODO 添加 graceful shutdown
	return a.server.ListenAndServe()
}

func (a *API) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
