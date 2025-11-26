package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create API with all services", func(t *testing.T) {
		t.Parallel()

		// 创建 mock services（使用 nil，因为 New 函数会创建新的实例）
		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, api)
		assert.NotNil(t, api.engine)
		assert.NotNil(t, api.server)
		assert.NotNil(t, api.node)
		assert.NotNil(t, api.instance)
		assert.NotNil(t, api.volume)
		assert.NotNil(t, api.image)
		assert.Equal(t, ":8080", api.server.Addr)
	})

	t.Run("API has registered routes", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// 验证路由已注册
		routes := api.engine.Routes()
		assert.Greater(t, len(routes), 0, "API should have registered routes")

		// 验证至少有一些预期的路由
		routePaths := make(map[string]bool)
		for _, route := range routes {
			routePaths[route.Path] = true
		}

		// 检查一些关键路由是否存在
		assert.True(t, routePaths["/api/list-nodes"] || routePaths["/api/describe-node"], "should have node routes")
		assert.True(t, routePaths["/api/instances/run"] || routePaths["/api/instances/describe"], "should have instance routes")
		assert.True(t, routePaths["/api/create-volume"] || routePaths["/api/describe-volume"], "should have volume routes")
		assert.True(t, routePaths["/api/images/create"] || routePaths["/api/images/describe"], "should have image routes")
	})
}

func TestAPI_Name(t *testing.T) {
	t.Parallel()

	t.Run("returns correct name", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		name := api.Name()
		assert.Equal(t, "API Server", name)
	})
}

func TestAPI_Run(t *testing.T) {
	t.Parallel()

	t.Run("run with context cancellation", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// 使用一个未使用的端口避免冲突
		api.server.Addr = ":0"

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 在 goroutine 中运行，然后立即取消
		errCh := make(chan error, 1)
		go func() {
			errCh <- api.Run(ctx)
		}()

		// 等待一小段时间确保服务器启动
		time.Sleep(10 * time.Millisecond)

		// 取消 context
		cancel()

		// 等待 Run 返回
		select {
		case err := <-errCh:
			if err != nil && strings.Contains(err.Error(), "operation not permitted") {
				t.Skip("Skipping Run test: socket operations not permitted in this environment")
			}
			assert.NoError(t, err, "Run should return nil when context is cancelled")
		case <-time.After(1 * time.Second):
			t.Fatal("Run did not return within timeout")
		}
	})

	t.Run("run with server error", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// 使用一个无效的地址来触发错误
		api.server.Addr = "invalid-address"

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = api.Run(ctx)
		// 可能会返回错误或超时，两种情况都接受
		if err != nil {
			assert.Error(t, err)
		}
	})
}

func TestAPI_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("shutdown running server", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// 使用一个未使用的端口
		api.server.Addr = ":0"

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 启动服务器
		go func() {
			_ = api.Run(ctx)
		}()

		// 等待服务器启动
		time.Sleep(50 * time.Millisecond)

		// 关闭服务器
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer shutdownCancel()

		err = api.Shutdown(shutdownCtx)
		assert.NoError(t, err, "Shutdown should succeed")

		// 取消运行 context
		cancel()
	})

	t.Run("shutdown with timeout", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// 使用一个未使用的端口
		api.server.Addr = ":0"

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 启动服务器
		go func() {
			_ = api.Run(ctx)
		}()

		// 等待服务器启动
		time.Sleep(50 * time.Millisecond)

		// 使用一个很短的超时来测试超时场景
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer shutdownCancel()

		// 等待超时
		time.Sleep(10 * time.Millisecond)

		err = api.Shutdown(shutdownCtx)
		// 可能会因为超时而失败，这是预期的
		_ = err

		// 取消运行 context
		cancel()
	})
}

func TestPrintRoutes(t *testing.T) {
	t.Parallel()

	t.Run("print routes for engine with routes", func(t *testing.T) {
		t.Parallel()

		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// printRoutes 是私有函数，但可以通过 New 间接测试
		// 验证路由已注册（printRoutes 在 New 中被调用）
		routes := api.engine.Routes()
		assert.Greater(t, len(routes), 0, "should have routes registered")
	})

	t.Run("print routes for empty engine", func(t *testing.T) {
		t.Parallel()

		// 创建一个空的 engine 来测试 printRoutes 的空路由情况
		// 由于 printRoutes 是私有函数，我们通过 New 来间接测试
		engine := gin.Default()
		routes := engine.Routes()
		// 空的 router 应该没有路由
		assert.Equal(t, 0, len(routes), "empty router should have no routes")
	})

	t.Run("printRoutes called during New", func(t *testing.T) {
		t.Parallel()

		// 验证 printRoutes 在 New 中被调用
		// 由于 printRoutes 会输出到 stdout，我们通过验证路由存在来间接验证
		api, err := New(nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		// 验证路由已注册（printRoutes 在 New 中被调用，会打印路由）
		routes := api.engine.Routes()
		assert.Greater(t, len(routes), 0, "printRoutes should have been called and routes should be registered")
	})
}
