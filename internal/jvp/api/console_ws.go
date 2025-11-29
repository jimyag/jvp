package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/wsproxy"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32768,
	WriteBufferSize: 32768,
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源 (生产环境应该检查 Origin)
		return true
	},
}

type ConsoleWS struct {
	instanceService *service.InstanceService
}

func NewConsoleWS(instanceService *service.InstanceService) *ConsoleWS {
	return &ConsoleWS{
		instanceService: instanceService,
	}
}

func (c *ConsoleWS) RegisterRoutes(router *gin.RouterGroup) {
	consoleRouter := router.Group("/console")
	// 路由包含 node_name 和 instance_id
	consoleRouter.GET("/vnc/:node_name/:instance_id", c.HandleVNCWebSocket)
	consoleRouter.GET("/serial/:node_name/:instance_id", c.HandleSerialWebSocket)
}

// HandleVNCWebSocket 处理 VNC WebSocket 连接
func (c *ConsoleWS) HandleVNCWebSocket(ctx *gin.Context) {
	logger := zerolog.Ctx(ctx.Request.Context())
	nodeName := ctx.Param("node_name")
	instanceID := ctx.Param("instance_id")

	logger.Info().
		Str("node_name", nodeName).
		Str("instance_id", instanceID).
		Msg("VNC WebSocket connection request")

	// 升级为 WebSocket 连接
	wsConn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to upgrade WebSocket")
		return
	}
	defer wsConn.Close()

	// 获取节点的 libvirt 客户端
	client, err := c.instanceService.GetLibvirtClient(ctx.Request.Context(), nodeName)
	if err != nil {
		logger.Error().
			Err(err).
			Str("node_name", nodeName).
			Msg("Failed to get node connection")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to connect to node"))
		return
	}

	// 获取实例信息
	instance, err := c.instanceService.GetInstance(ctx.Request.Context(), nodeName, instanceID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Failed to get instance")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Instance not found"))
		return
	}

	// 检查实例状态
	if instance.State != "running" {
		logger.Warn().
			Str("instance_id", instanceID).
			Str("state", instance.State).
			Msg("Instance is not running")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Instance is not running"))
		return
	}

	// 获取 domain
	domain, err := client.GetDomainByName(instanceID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Failed to get domain")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to get domain"))
		return
	}

	// 获取控制台信息
	consoleInfo, err := client.GetDomainConsoleInfo(domain)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Failed to get console info")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to get console info"))
		return
	}

	if consoleInfo.VNCSocket == "" {
		logger.Warn().
			Str("instance_id", instanceID).
			Msg("VNC socket not configured")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("VNC not configured"))
		return
	}

	// 判断是远程连接还是本地连接
	isRemote := client.IsRemoteConnection()
	var sshTarget string
	if isRemote {
		sshTarget, err = client.GetSSHTarget()
		if err != nil {
			logger.Error().
				Err(err).
				Str("node_name", nodeName).
				Msg("Failed to get SSH target")
			wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to get SSH target"))
			return
		}
	}

	logger.Info().
		Str("instance_id", instanceID).
		Str("vnc_socket", consoleInfo.VNCSocket).
		Str("ws_remote_addr", ctx.Request.RemoteAddr).
		Bool("is_remote", isRemote).
		Str("ssh_target", sshTarget).
		Msg("Starting VNC proxy")

	// 对于本地连接，验证 VNC socket 文件是否存在
	if !isRemote {
		if _, err := os.Stat(consoleInfo.VNCSocket); err != nil {
			logger.Error().
				Err(err).
				Str("instance_id", instanceID).
				Str("vnc_socket", consoleInfo.VNCSocket).
				Msg("VNC socket file not accessible")
			wsConn.WriteMessage(websocket.CloseMessage, []byte("VNC socket file not accessible"))
			return
		}
	}

	// 创建并启动 VNC 代理
	var proxy *wsproxy.VNCProxy
	if isRemote {
		proxy = wsproxy.NewRemoteVNCProxy(consoleInfo.VNCSocket, wsConn, sshTarget)
	} else {
		proxy = wsproxy.NewVNCProxy(consoleInfo.VNCSocket, wsConn)
	}
	if err := proxy.Start(); err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("VNC proxy failed")
	}

	logger.Info().
		Str("instance_id", instanceID).
		Msg("VNC proxy session ended")
}

// HandleSerialWebSocket 处理 Serial WebSocket 连接
func (c *ConsoleWS) HandleSerialWebSocket(ctx *gin.Context) {
	logger := zerolog.Ctx(ctx.Request.Context())
	nodeName := ctx.Param("node_name")
	instanceID := ctx.Param("instance_id")

	logger.Info().
		Str("node_name", nodeName).
		Str("instance_id", instanceID).
		Msg("Serial WebSocket connection request")

	// 升级为 WebSocket 连接
	wsConn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to upgrade WebSocket")
		return
	}
	defer wsConn.Close()

	// 获取节点的 libvirt 客户端
	client, err := c.instanceService.GetLibvirtClient(ctx.Request.Context(), nodeName)
	if err != nil {
		logger.Error().
			Err(err).
			Str("node_name", nodeName).
			Msg("Failed to get node connection")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to connect to node"))
		return
	}

	// 获取实例信息
	instance, err := c.instanceService.GetInstance(ctx.Request.Context(), nodeName, instanceID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Failed to get instance")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Instance not found"))
		return
	}

	// 检查实例状态
	if instance.State != "running" {
		logger.Warn().
			Str("instance_id", instanceID).
			Str("state", instance.State).
			Msg("Instance is not running")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Instance is not running"))
		return
	}

	// 获取 domain
	domain, err := client.GetDomainByName(instanceID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Failed to get domain")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to get domain"))
		return
	}

	// 获取控制台信息
	consoleInfo, err := client.GetDomainConsoleInfo(domain)
	if err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Failed to get console info")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to get console info"))
		return
	}

	if consoleInfo.SerialDevice == "" {
		logger.Warn().
			Str("instance_id", instanceID).
			Msg("Serial device not available")
		wsConn.WriteMessage(websocket.CloseMessage, []byte("Serial console not available"))
		return
	}

	// 判断是远程连接还是本地连接
	isRemote := client.IsRemoteConnection()
	var sshTarget string
	if isRemote {
		sshTarget, err = client.GetSSHTarget()
		if err != nil {
			logger.Error().
				Err(err).
				Str("node_name", nodeName).
				Msg("Failed to get SSH target")
			wsConn.WriteMessage(websocket.CloseMessage, []byte("Failed to get SSH target"))
			return
		}
	}

	logger.Info().
		Str("instance_id", instanceID).
		Str("serial_device", consoleInfo.SerialDevice).
		Bool("is_remote", isRemote).
		Str("ssh_target", sshTarget).
		Msg("Starting Serial proxy")

	// 创建并启动 Serial 代理
	var serialProxy *wsproxy.SerialProxy
	if isRemote {
		serialProxy = wsproxy.NewRemoteSerialProxy(consoleInfo.SerialDevice, wsConn, sshTarget)
	} else {
		serialProxy = wsproxy.NewSerialProxy(consoleInfo.SerialDevice, wsConn)
	}
	if err := serialProxy.Start(); err != nil {
		logger.Error().
			Err(err).
			Str("instance_id", instanceID).
			Msg("Serial proxy failed")
	}

	logger.Info().
		Str("instance_id", instanceID).
		Msg("Serial proxy session ended")
}
