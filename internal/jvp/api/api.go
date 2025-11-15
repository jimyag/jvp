// Package api 提供 HTTP API 路由和处理逻辑
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/service"
)

type API struct {
	engine *gin.Engine
	server *http.Server

	instance  *Instance
	volume    *Volume
	snapshot  *Snapshot
	image     *Image
	keypair   *KeyPair
	consoleWS *ConsoleWS
}

func New(
	instanceService *service.InstanceService,
	volumeService *service.VolumeService,
	snapshotService *service.SnapshotService,
	imageService *service.ImageService,
	keyPairService *service.KeyPairService,
) (*API, error) {
	// 先禁用 Gin 的 debug 路由输出（避免打印带函数名的路由信息）
	// 注意：这需要在创建 engine 之前设置
	gin.SetMode(gin.ReleaseMode)

	engine := gin.Default()
	api := &API{
		engine:    engine,
		instance:  NewInstance(instanceService),
		volume:    NewVolume(volumeService),
		snapshot:  NewSnapshot(snapshotService),
		image:     NewImage(imageService),
		keypair:   NewKeyPair(keyPairService),
		consoleWS: NewConsoleWS(instanceService),
	}

	apiGroup := engine.Group("/api")
	api.instance.RegisterRoutes(apiGroup)
	api.volume.RegisterRoutes(apiGroup)
	api.snapshot.RegisterRoutes(apiGroup)
	api.image.RegisterRoutes(apiGroup)
	api.keypair.RegisterRoutes(apiGroup)
	api.consoleWS.RegisterRoutes(apiGroup)

	// 打印路由信息（只显示方法和路径，不显示处理函数）
	printRoutes(engine)

	api.server = &http.Server{
		Addr:    ":8080",
		Handler: engine,
	}
	return api, nil
}

// printRoutes 打印所有注册的路由（只显示方法和路径）
func printRoutes(engine *gin.Engine) {
	routes := engine.Routes()
	if len(routes) == 0 {
		return
	}

	// 使用 fmt 直接打印到标准输出，避免使用 gin 的 debug 输出
	fmt.Fprintf(os.Stdout, "\n[API Routes]\n")
	fmt.Fprintf(os.Stdout, "Method   Path\n")
	fmt.Fprintf(os.Stdout, "----------------------------\n")

	// 打印每个路由（只显示方法和路径）
	for _, route := range routes {
		fmt.Fprintf(os.Stdout, "%-8s %s\n", route.Method, route.Path)
	}
	fmt.Fprintf(os.Stdout, "\n")
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
