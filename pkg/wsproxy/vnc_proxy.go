package wsproxy

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// VNCProxy VNC WebSocket 代理
type VNCProxy struct {
	vncSocket  string
	wsConn     *websocket.Conn
	unixConn   net.Conn
	sshCmd     *exec.Cmd // SSH 进程（用于远程连接）
	mu         sync.Mutex
	closed     bool
	isRemote   bool
	sshTarget  string // SSH 目标，格式: user@host
}

// NewVNCProxy 创建本地 VNC 代理
func NewVNCProxy(vncSocket string, wsConn *websocket.Conn) *VNCProxy {
	return &VNCProxy{
		vncSocket: vncSocket,
		wsConn:    wsConn,
		isRemote:  false,
	}
}

// NewRemoteVNCProxy 创建远程 VNC 代理（通过 SSH）
func NewRemoteVNCProxy(vncSocket string, wsConn *websocket.Conn, sshTarget string) *VNCProxy {
	return &VNCProxy{
		vncSocket: vncSocket,
		wsConn:    wsConn,
		isRemote:  true,
		sshTarget: sshTarget,
	}
}

// Start 启动代理
func (p *VNCProxy) Start() error {
	var conn net.Conn
	var err error

	if p.isRemote {
		// 远程连接：通过 SSH socat 转发
		conn, err = p.connectViaSSH()
	} else {
		// 本地连接：直接连接 Unix Socket
		conn, err = net.Dial("unix", p.vncSocket)
	}

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
	for {
		n, err := p.unixConn.Read(buffer)
		if err != nil {
			return
		}

		if n > 0 {
			// 发送二进制数据到 WebSocket
			p.mu.Lock()
			if !p.closed {
				err = p.wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
				if err != nil {
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

	for {
		messageType, data, err := p.wsConn.ReadMessage()
		if err != nil {
			return
		}

		// 只处理二进制消息
		if messageType == websocket.BinaryMessage {
			_, err = p.unixConn.Write(data)
			if err != nil {
				return
			}
		}
	}
}

// connectViaSSH 通过 SSH 连接到远程 VNC socket
func (p *VNCProxy) connectViaSSH() (net.Conn, error) {
	// 使用 SSH 建立到远程 Unix socket 的连接
	// 方法：使用 ssh -W 配合 socat 转发
	// ssh -o StrictHostKeyChecking=no user@host "socat - UNIX-CONNECT:/path/to/socket"
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		p.sshTarget,
		fmt.Sprintf("socat - UNIX-CONNECT:%s", p.vncSocket),
	)

	// 获取 stdin 和 stdout 管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// 捕获 stderr 用于诊断
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 启动 SSH 进程
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start SSH: %w", err)
	}

	p.sshCmd = cmd

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
					Str("vnc_socket", p.vncSocket).
					Str("stderr", string(buf[:n])).
					Msg("SSH VNC tunnel stderr")
			}
		}
	}()

	log.Info().
		Str("ssh_target", p.sshTarget).
		Str("vnc_socket", p.vncSocket).
		Msg("SSH tunnel established for VNC")

	// 返回一个自定义的 Conn 封装 stdin/stdout
	return &sshConn{
		stdin:  stdin,
		stdout: stdout,
		cmd:    cmd,
	}, nil
}

// sshConn 封装 SSH 进程的 stdin/stdout 为 net.Conn 接口
type sshConn struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (c *sshConn) Read(b []byte) (n int, err error) {
	return c.stdout.Read(b)
}

func (c *sshConn) Write(b []byte) (n int, err error) {
	return c.stdin.Write(b)
}

func (c *sshConn) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	return nil
}

func (c *sshConn) LocalAddr() net.Addr                     { return nil }
func (c *sshConn) RemoteAddr() net.Addr                    { return nil }
func (c *sshConn) SetDeadline(_ time.Time) error           { return nil }
func (c *sshConn) SetReadDeadline(_ time.Time) error       { return nil }
func (c *sshConn) SetWriteDeadline(_ time.Time) error      { return nil }

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
	if p.sshCmd != nil && p.sshCmd.Process != nil {
		p.sshCmd.Process.Kill()
	}

	log.Info().Str("vnc_socket", p.vncSocket).Msg("VNC proxy closed")
}
