package wsproxy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// SerialProxy Serial Console WebSocket 代理
type SerialProxy struct {
	serialDevice string
	wsConn       *websocket.Conn
	ptyFile      *os.File
	sshCmd       *exec.Cmd // SSH 进程（用于远程连接）
	reader       io.ReadCloser
	writer       io.WriteCloser
	mu           sync.Mutex
	closed       bool
	isRemote     bool
	sshTarget    string // SSH 目标，格式: user@host
}

// NewSerialProxy 创建本地 Serial 代理
func NewSerialProxy(serialDevice string, wsConn *websocket.Conn) *SerialProxy {
	return &SerialProxy{
		serialDevice: serialDevice,
		wsConn:       wsConn,
		isRemote:     false,
	}
}

// NewRemoteSerialProxy 创建远程 Serial 代理（通过 SSH）
func NewRemoteSerialProxy(serialDevice string, wsConn *websocket.Conn, sshTarget string) *SerialProxy {
	return &SerialProxy{
		serialDevice: serialDevice,
		wsConn:       wsConn,
		isRemote:     true,
		sshTarget:    sshTarget,
	}
}

// Start 启动代理
func (p *SerialProxy) Start() error {
	if p.isRemote {
		// 远程连接：通过 SSH socat 转发
		if err := p.connectViaSSH(); err != nil {
			return err
		}
	} else {
		// 本地连接：直接打开 PTY 设备
		file, err := os.OpenFile(p.serialDevice, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("failed to open serial device %s: %w", p.serialDevice, err)
		}
		p.ptyFile = file
		p.reader = file
		p.writer = file
	}

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
		n, err := p.reader.Read(buffer)
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
			_, err = p.writer.Write(data)
			if err != nil {
				log.Debug().Err(err).Msg("Error writing to PTY")
				return
			}
		}
	}
}

// connectViaSSH 通过 SSH 连接到远程 Serial 设备
func (p *SerialProxy) connectViaSSH() error {
	// 使用 SSH socat 连接到远程 PTY 设备
	// ssh -o StrictHostKeyChecking=no user@host "socat - /dev/pts/X,raw,echo=0"
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		p.sshTarget,
		fmt.Sprintf("socat - %s,raw,echo=0", p.serialDevice),
	)

	// 获取 stdin 和 stdout 管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// 捕获 stderr 用于诊断
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 启动 SSH 进程
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("failed to start SSH: %w", err)
	}

	p.sshCmd = cmd
	p.reader = stdout
	p.writer = stdin

	// 异步读取 stderr 并记录错误
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				log.Error().
					Str("ssh_target", p.sshTarget).
					Str("serial_device", p.serialDevice).
					Str("stderr", string(buf[:n])).
					Msg("SSH Serial tunnel stderr")
			}
		}
	}()

	log.Info().
		Str("ssh_target", p.sshTarget).
		Str("serial_device", p.serialDevice).
		Msg("SSH tunnel established for Serial")

	return nil
}

// Close 关闭代理连接
func (p *SerialProxy) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	if p.reader != nil {
		p.reader.Close()
	}
	if p.writer != nil {
		p.writer.Close()
	}
	if p.ptyFile != nil {
		p.ptyFile.Close()
	}
	if p.wsConn != nil {
		p.wsConn.Close()
	}
	if p.sshCmd != nil && p.sshCmd.Process != nil {
		p.sshCmd.Process.Kill()
	}

	log.Info().Str("serial_device", p.serialDevice).Msg("Serial proxy closed")
}
