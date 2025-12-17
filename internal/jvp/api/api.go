// Package api 提供 HTTP API 路由和处理逻辑
package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/config"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/rs/zerolog/log"
)

type API struct {
	engine *gin.Engine
	server *http.Server

	node        *NodeAPI
	instance    *Instance
	volume      *Volume
	keypair     *KeyPair
	consoleWS   *ConsoleWS
	storagePool *StoragePoolAPI
	template    *Template
	snapshot    *Snapshot
	network     *NetworkAPI
	bridge      *BridgeAPI
	frontendFS  http.FileSystem
}

func New(
	nodeService *service.NodeService,
	instanceService *service.InstanceService,
	volumeService *service.VolumeService,
	keyPairService *service.KeyPairService,
	storagePoolService *service.StoragePoolService,
	templateService *service.TemplateService,
	snapshotService *service.SnapshotService,
	networkService *service.NetworkService,
	bridgeService *service.BridgeService,
	cfg *config.Config,
) (*API, error) {
	// 先禁用 Gin 的 debug 路由输出（避免打印带函数名的路由信息）
	// 注意：这需要在创建 engine 之前设置
	gin.SetMode(gin.ReleaseMode)

	engine := gin.Default()
	api := &API{
		engine:      engine,
		node:        NewNodeAPI(nodeService),
		instance:    NewInstance(instanceService),
		volume:      NewVolume(volumeService),
		keypair:     NewKeyPair(keyPairService),
		consoleWS:   NewConsoleWS(instanceService),
		storagePool: NewStoragePoolAPI(storagePoolService),
		template:    NewTemplate(templateService),
		snapshot:    NewSnapshot(snapshotService),
		network:     NewNetworkAPI(networkService),
		bridge:      NewBridgeAPI(bridgeService),
	}

	apiGroup := engine.Group("/api")
	api.node.RegisterRoutes(apiGroup)
	api.instance.RegisterRoutes(apiGroup)
	api.volume.RegisterRoutes(apiGroup)
	api.keypair.RegisterRoutes(apiGroup)
	api.consoleWS.RegisterRoutes(apiGroup)
	api.storagePool.RegisterRoutes(apiGroup)
	api.template.RegisterRoutes(apiGroup)
	api.snapshot.RegisterRoutes(apiGroup)
	api.network.RegisterRoutes(apiGroup)
	api.bridge.RegisterRoutes(apiGroup)
	api.mountFrontend()

	api.server = &http.Server{
		Addr:    cfg.Address,
		Handler: engine,
	}
	log.Info().Str("address", cfg.Address).Msg("API server configured")
	return api, nil
}

func (a *API) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *API) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

// Name 实现 grace.Grace 接口
func (a *API) Name() string {
	return "API Server"
}
