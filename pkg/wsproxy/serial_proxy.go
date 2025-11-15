package wsproxy

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// SerialProxy Serial Console WebSocket 代理
type SerialProxy struct {
	serialDevice string
	wsConn       *websocket.Conn
	ptyFile      *os.File
	mu           sync.Mutex
	closed       bool
}

// NewSerialProxy 创建 Serial 代理
func NewSerialProxy(serialDevice string, wsConn *websocket.Conn) *SerialProxy {
	return &SerialProxy{
		serialDevice: serialDevice,
		wsConn:       wsConn,
	}
}

// Start 启动代理
func (p *SerialProxy) Start() error {
	// 打开 PTY 设备
	file, err := os.OpenFile(p.serialDevice, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open serial device %s: %w", p.serialDevice, err)
	}
	p.ptyFile = file

	log.Info().
		Str("serial_device", p.serialDevice).
		Msg("Serial proxy connected to PTY device")

	// 启动双向数据传输
	var wg sync.WaitGroup
	wg.Add(2)

	// PTY -> WebSocket
	go func() {
		defer wg.Done()
		p.forwardPTYToWS()
	}()

	// WebSocket -> PTY
	go func() {
		defer wg.Done()
		p.forwardWSToPTY()
	}()

	wg.Wait()
	return nil
}

// forwardPTYToWS 将 PTY 数据转发到 WebSocket
func (p *SerialProxy) forwardPTYToWS() {
	defer func() {
		p.Close()
	}()

	buffer := make([]byte, 4096) // 4KB buffer for serial data
	for {
		n, err := p.ptyFile.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Debug().Err(err).Msg("Error reading from PTY")
			}
			return
		}

		if n > 0 {
			// 发送文本数据到 WebSocket
			p.mu.Lock()
			if !p.closed {
				err = p.wsConn.WriteMessage(websocket.TextMessage, buffer[:n])
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

// forwardWSToPTY 将 WebSocket 数据转发到 PTY
func (p *SerialProxy) forwardWSToPTY() {
	defer func() {
		p.Close()
	}()

	for {
		messageType, data, err := p.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Debug().Err(err).Msg("WebSocket read error")
			}
			return
		}

		// 处理文本和二进制消息
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			_, err = p.ptyFile.Write(data)
			if err != nil {
				log.Debug().Err(err).Msg("Error writing to PTY")
				return
			}
		}
	}
}

// Close 关闭代理连接
func (p *SerialProxy) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	if p.ptyFile != nil {
		p.ptyFile.Close()
	}
	if p.wsConn != nil {
		p.wsConn.Close()
	}

	log.Info().Str("serial_device", p.serialDevice).Msg("Serial proxy closed")
}
