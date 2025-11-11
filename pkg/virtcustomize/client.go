package virtcustomize

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// VirtCustomizeClient 定义 virt-customize 客户端接口
type VirtCustomizeClient interface {
	ResetPassword(ctx context.Context, diskPath string, username, password string) error
	ResetMultiplePasswords(ctx context.Context, diskPath string, users map[string]string) error
	ValidateDiskPath(diskPath string) error
	SetTimeout(timeout time.Duration)
}

// Client virt-customize 客户端
type Client struct {
	virtCustomizePath string // virt-customize 命令路径
	timeout           time.Duration
}

// 确保 Client 实现了 VirtCustomizeClient 接口
var _ VirtCustomizeClient = (*Client)(nil)

// NewClient 创建 virt-customize 客户端
func NewClient() (*Client, error) {
	// 查找 virt-customize 命令路径
	path, err := exec.LookPath("virt-customize")
	if err != nil {
		return nil, fmt.Errorf("virt-customize command not found: %w", err)
	}

	return &Client{
		virtCustomizePath: path,
		timeout:           5 * time.Minute, // 默认超时 5 分钟
	}, nil
}

// NewClientWithPath 使用指定的路径创建客户端
func NewClientWithPath(path string) *Client {
	return &Client{
		virtCustomizePath: path,
		timeout:           5 * time.Minute,
	}
}

// ResetPassword 重置单个用户的密码
func (c *Client) ResetPassword(ctx context.Context, diskPath string, username, password string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("disk_path", diskPath).
		Str("username", username).
		Msg("Resetting password")

	// 验证磁盘文件存在
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return fmt.Errorf("disk file not found: %s", diskPath)
	}

	// 创建带超时的 context
	cmdCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 构建命令
	// 使用环境变量传递密码，避免命令行参数泄露
	cmd := exec.CommandContext(cmdCtx, c.virtCustomizePath,
		"-a", diskPath,
		"--password", fmt.Sprintf("%s:password:%s", username, password),
	)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error().
			Err(err).
			Str("output", string(output)).
			Msg("Failed to reset password")
		return fmt.Errorf("virt-customize failed: %w, output: %s", err, string(output))
	}

	logger.Info().
		Str("username", username).
		Msg("Password reset successfully")

	return nil
}

// ResetMultiplePasswords 重置多个用户的密码
func (c *Client) ResetMultiplePasswords(ctx context.Context, diskPath string, users map[string]string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("disk_path", diskPath).
		Int("user_count", len(users)).
		Msg("Resetting multiple passwords")

	// 验证磁盘文件存在
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return fmt.Errorf("disk file not found: %s", diskPath)
	}

	if len(users) == 0 {
		return fmt.Errorf("no users specified")
	}

	// 构建命令参数
	args := []string{"-a", diskPath}
	for username, password := range users {
		args = append(args, "--password", fmt.Sprintf("%s:password:%s", username, password))
	}

	// 创建带超时的 context
	cmdCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(cmdCtx, c.virtCustomizePath, args...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error().
			Err(err).
			Str("output", string(output)).
			Msg("Failed to reset multiple passwords")
		return fmt.Errorf("virt-customize failed: %w, output: %s", err, string(output))
	}

	logger.Info().
		Int("user_count", len(users)).
		Msg("Multiple passwords reset successfully")

	return nil
}

// ValidateDiskPath 验证磁盘路径是否有效
func (c *Client) ValidateDiskPath(diskPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return fmt.Errorf("disk file not found: %s", diskPath)
	}

	// 检查文件扩展名（仅支持 qcow2）
	ext := strings.ToLower(filepath.Ext(diskPath))
	if ext != ".qcow2" {
		return fmt.Errorf("unsupported disk format: %s (only qcow2 is supported)", ext)
	}

	return nil
}

// SetTimeout 设置命令超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}
