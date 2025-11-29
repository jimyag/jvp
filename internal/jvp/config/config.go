package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	// LibvirtURI 是 libvirt 连接 URI
	// 支持以下格式：
	// - qemu:///system (本地系统连接，默认)
	// - qemu+ssh://user@host/system (SSH 远程连接)
	// - qemu+tcp://host/system (TCP 远程连接)
	// 可以通过环境变量 LIBVIRT_URI 配置
	LibvirtURI string

	// DataDir 是 JVP 数据目录
	// 用于存储镜像、卷、元数据等
	// 可以通过环境变量 JVP_DATA_DIR 配置
	// 默认：~/.local/share/jvp
	DataDir string

	Address string
}

func New() (*Config, error) {
	cfg := &Config{
		LibvirtURI: getLibvirtURI(),
		DataDir:    getDataDir(),
		Address:    getAddress(),
	}
	return cfg, nil
}

// getLibvirtURI 获取 libvirt URI，优先使用环境变量
func getLibvirtURI() string {
	// 1. 优先使用环境变量 LIBVIRT_URI
	if uri := os.Getenv("LIBVIRT_URI"); uri != "" {
		return uri
	}

	// 2. 尝试使用 JVP_LIBVIRT_URI
	if uri := os.Getenv("JVP_LIBVIRT_URI"); uri != "" {
		return uri
	}

	// 3. 默认使用本地系统连接
	return "qemu:///system"
}

// getDataDir 获取数据目录，优先使用环境变量
func getDataDir() string {
	// 1. 优先使用环境变量 JVP_DATA_DIR
	if dir := os.Getenv("JVP_DATA_DIR"); dir != "" {
		return dir
	}

	// 2. 使用用户主目录下的 .local/share/jvp
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "jvp")
	}

	// 3. 如果无法获取主目录，使用当前目录下的 data
	return filepath.Join(".", "data")
}

// getAddress 获取绑定地址，优先使用环境变量 JVP_ADDRESS
func getAddress() string {
	if addr := os.Getenv("JVP_ADDRESS"); addr != "" {
		return addr
	}

	return "0.0.0.0:7777"
}
