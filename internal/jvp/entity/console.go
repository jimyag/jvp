package entity

// GetConsoleRequest 获取控制台连接信息请求
type GetConsoleRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Type       string `json:"type"` // vnc, serial, 为空则返回两种都支持的信息
}

// GetConsoleResponse 获取控制台连接信息响应
type GetConsoleResponse struct {
	InstanceID   string `json:"instance_id"`
	VNCSocket    string `json:"vnc_socket,omitempty"`    // VNC Unix Socket 路径
	VNCPort      int    `json:"vnc_port,omitempty"`      // VNC WebSocket 代理端口 (由前端连接)
	VNCToken     string `json:"vnc_token,omitempty"`     // VNC 连接认证 token (可选)
	SerialDevice string `json:"serial_device,omitempty"` // Serial PTY 设备路径
	SerialPort   int    `json:"serial_port,omitempty"`   // Serial WebSocket 代理端口
	SerialToken  string `json:"serial_token,omitempty"`  // Serial 连接认证 token (可选)
	Type         string `json:"type"`                    // 返回的控制台类型: vnc, serial, both
}
