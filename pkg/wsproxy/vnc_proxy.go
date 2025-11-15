package wsproxy

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// VNCProxy VNC WebSocket 代理
type VNCProxy struct {
	vncSocket string
	wsConn    *websocket.Conn
	unixConn  net.Conn
	mu        sync.Mutex
	closed    bool
}

// NewVNCProxy 创建 VNC 代理
func NewVNCProxy(vncSocket string, wsConn *websocket.Conn) *VNCProxy {
	return &VNCProxy{
		vncSocket: vncSocket,
		wsConn:    wsConn,
	}
}

// Start 启动代理
func (p *VNCProxy) Start() error {
	// 连接到 VNC Unix Socket
	conn, err := net.Dial("unix", p.vncSocket)
	if err != nil {
		return fmt.Errorf("failed to connect to VNC socket %s: %w", p.vncSocket, err)
	}
	p.unixConn = conn

	log.Info().
		Str("vnc_socket", p.vncSocket).
		Msg("VNC proxy connected to Unix socket")

	// 启动双向数据传输
	var wg sync.WaitGroup
	wg.Add(2)

	// Unix Socket -> WebSocket
	go func() {
		defer wg.Done()
		p.forwardUnixToWS()
	}()

	// WebSocket -> Unix Socket
	go func() {
		defer wg.Done()
		p.forwardWSToUnix()
	}()

	wg.Wait()
	return nil
}

// forwardUnixToWS 将 Unix Socket 数据转发到 WebSocket
func (p *VNCProxy) forwardUnixToWS() {
	defer func() {
		p.Close()
	}()

	buffer := make([]byte, 32768) // 32KB buffer for VNC data
	totalBytes := 0
	for {
		n, err := p.unixConn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Debug().Err(err).Msg("Error reading from VNC socket")
			}
			log.Info().Int("total_bytes_forwarded", totalBytes).Msg("VNC->WS forwarding stopped")
			return
		}

		if n > 0 {
			totalBytes += n
			log.Debug().Int("bytes", n).Int("total", totalBytes).Msg("VNC->WS forwarding data")
			// 发送二进制数据到 WebSocket
			p.mu.Lock()
			if !p.closed {
				err = p.wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
				if err != nil {
					log.Debug().Err(err).Msg("Error writing to WebSocket")
					p.mu.Unlock()
					return
				}
			}
			p.mu.Unlock()
		}
	}
}

// forwardWSToUnix 将 WebSocket 数据转发到 Unix Socket
func (p *VNCProxy) forwardWSToUnix() {
	defer func() {
		p.Close()
	}()

	totalBytes := 0
	for {
		messageType, data, err := p.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Debug().Err(err).Msg("WebSocket read error")
			}
			log.Info().Int("total_bytes_forwarded", totalBytes).Msg("WS->VNC forwarding stopped")
			return
		}

		// 只处理二进制消息
		if messageType == websocket.BinaryMessage {
			totalBytes += len(data)
			log.Debug().Int("bytes", len(data)).Int("total", totalBytes).Msg("WS->VNC forwarding data")
			_, err = p.unixConn.Write(data)
			if err != nil {
				log.Debug().Err(err).Msg("Error writing to VNC socket")
				return
			}
		}
	}
}

// Close 关闭代理连接
func (p *VNCProxy) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	if p.unixConn != nil {
		p.unixConn.Close()
	}
	if p.wsConn != nil {
		p.wsConn.Close()
	}

	log.Info().Str("vnc_socket", p.vncSocket).Msg("VNC proxy closed")
}
