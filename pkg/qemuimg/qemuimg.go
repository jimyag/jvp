package qemuimg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client 封装 qemu-img 命令行工具的操作
type Client struct {
	qemuImgPath string
	timeout     time.Duration
}

// New 创建新的 qemuimg client
// qemuImgPath 是 qemu-img 的路径，如果为空则使用默认的 "qemu-img"
func New(qemuImgPath string) *Client {
	if qemuImgPath == "" {
		qemuImgPath = "qemu-img"
	}
	return &Client{
		qemuImgPath: qemuImgPath,
		timeout:     30 * time.Minute, // 默认超时 30 分钟（大文件操作可能需要较长时间）
	}
}

// WithTimeout 设置操作超时时间
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.timeout = timeout
	return c
}

// CreateFromBackingFile 从 backing file 创建新镜像
// 这是创建增量镜像的常用方式，可以节省存储空间
//
// 参数：
//   - format: 输出镜像格式（如 "qcow2"）
//   - backingFormat: backing file 的格式（如 "qcow2"）
//   - backingFile: backing file 的路径
//   - outputFile: 输出文件路径
//
// 示例：
//
//	err := client.CreateFromBackingFile("qcow2", "qcow2", "/path/to/base.qcow2", "/path/to/new.qcow2")
func (c *Client) CreateFromBackingFile(ctx context.Context, format, backingFormat, backingFile, outputFile string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "create",
		"-f", format,
		"-F", backingFormat,
		"-b", backingFile,
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create image from backing file %s: %w, output: %s", backingFile, err, string(output))
	}

	return nil
}

// Resize 调整镜像大小
// 只能扩大，不能缩小（除非使用 --shrink 参数，但不推荐）
//
// 参数：
//   - imagePath: 镜像文件路径
//   - sizeGB: 目标大小（GB）
//
// 示例：
//
//	err := client.Resize(ctx, "/path/to/image.qcow2", 20)
func (c *Client) Resize(ctx context.Context, imagePath string, sizeGB uint64) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "resize",
		imagePath,
		fmt.Sprintf("%dG", sizeGB),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to resize image %s to %dG: %w, output: %s", imagePath, sizeGB, err, string(output))
	}

	return nil
}

// Convert 转换镜像格式或复制镜像
//
// 参数：
//   - inputFormat: 输入镜像格式（如 "qcow2", "raw"）
//   - outputFormat: 输出镜像格式（如 "qcow2", "raw"）
//   - inputFile: 输入文件路径
//   - outputFile: 输出文件路径
//
// 示例：
//
//	// 将 qcow2 转换为 raw
//	err := client.Convert(ctx, "qcow2", "raw", "/path/to/input.qcow2", "/path/to/output.raw")
//
//	// 复制 qcow2 镜像
//	err := client.Convert(ctx, "qcow2", "qcow2", "/path/to/input.qcow2", "/path/to/output.qcow2")
func (c *Client) Convert(ctx context.Context, inputFormat, outputFormat, inputFile, outputFile string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "convert",
		"-f", inputFormat,
		"-O", outputFormat,
		inputFile,
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to convert image from %s to %s: %w, output: %s", inputFile, outputFile, err, string(output))
	}

	return nil
}

// Info 获取镜像信息
// 返回 qemu-img info 的原始输出
//
// 参数：
//   - imagePath: 镜像文件路径
//
// 返回：
//   - info 字符串（qemu-img info 的输出）
//
// 示例：
//
//	info, err := client.Info(ctx, "/path/to/image.qcow2")
func (c *Client) Info(ctx context.Context, imagePath string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // info 操作通常很快
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "info", imagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get image info for %s: %w, output: %s", imagePath, err, string(output))
	}

	return string(output), nil
}

// GetFormat 获取镜像的实际格式
// 通过解析 qemu-img info 输出来获取格式
//
// 参数：
//   - imagePath: 镜像文件路径
//
// 返回：
//   - format 字符串（如 "qcow2", "raw"）
//
// 示例：
//
//	format, err := client.GetFormat(ctx, "/path/to/image.img")
func (c *Client) GetFormat(ctx context.Context, imagePath string) (string, error) {
	info, err := c.Info(ctx, imagePath)
	if err != nil {
		return "", err
	}

	// 解析输出，查找 "file format: xxx" 行
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "file format:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("failed to parse format from qemu-img info output for %s", imagePath)
}

// Check 检查镜像完整性
//
// 参数：
//   - imagePath: 镜像文件路径
//   - format: 镜像格式（如 "qcow2"）
//
// 返回：
//   - 如果镜像完整，返回 nil；否则返回错误
//
// 示例：
//
//	err := client.Check(ctx, "/path/to/image.qcow2", "qcow2")
func (c *Client) Check(ctx context.Context, imagePath, format string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute) // check 操作可能需要较长时间
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "check",
		"-f", format,
		imagePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check image %s: %w, output: %s", imagePath, err, string(output))
	}

	return nil
}

// CreateEmpty 创建空镜像
//
// 参数：
//   - format: 镜像格式（如 "qcow2"）
//   - outputFile: 输出文件路径
//   - sizeGB: 镜像大小（GB）
//
// 示例：
//
//	err := client.CreateEmpty(ctx, "qcow2", "/path/to/new.qcow2", 10)
func (c *Client) CreateEmpty(ctx context.Context, format, outputFile string, sizeGB uint64) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "create",
		"-f", format,
		outputFile,
		fmt.Sprintf("%dG", sizeGB),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create empty image %s: %w, output: %s", outputFile, err, string(output))
	}

	return nil
}

// Snapshot 创建快照
// 注意：qemu-img snapshot 只支持 qcow2 格式
//
// 参数：
//   - imagePath: 镜像文件路径
//   - snapshotName: 快照名称
//
// 示例：
//
//	err := client.Snapshot(ctx, "/path/to/image.qcow2", "snapshot1")
func (c *Client) Snapshot(ctx context.Context, imagePath, snapshotName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "snapshot",
		"-c", snapshotName,
		imagePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create snapshot %s for %s: %w, output: %s", snapshotName, imagePath, err, string(output))
	}

	return nil
}

// DeleteSnapshot 删除快照
//
// 参数：
//   - imagePath: 镜像文件路径
//   - snapshotName: 快照名称
//
// 示例：
//
//	err := client.DeleteSnapshot(ctx, "/path/to/image.qcow2", "snapshot1")
func (c *Client) DeleteSnapshot(ctx context.Context, imagePath, snapshotName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "snapshot",
		"-d", snapshotName,
		imagePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete snapshot %s from %s: %w, output: %s", snapshotName, imagePath, err, string(output))
	}

	return nil
}

// ListSnapshots 列出镜像的所有快照
//
// 参数：
//   - imagePath: 镜像文件路径
//
// 返回：
//   - 快照名称列表
//
// 示例：
//
//	snapshots, err := client.ListSnapshots(ctx, "/path/to/image.qcow2")
func (c *Client) ListSnapshots(ctx context.Context, imagePath string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.qemuImgPath, "snapshot",
		"-l", imagePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots for %s: %w, output: %s", imagePath, err, string(output))
	}

	// 解析输出，提取快照名称
	// qemu-img snapshot -l 输出格式：
	// Snapshot list:
	// ID        TAG               VM SIZE                DATE       VM CLOCK
	// 1         snapshot1         0 B 2024-01-01 12:00:00   00:00:00.000
	lines := strings.Split(string(output), "\n")
	var snapshots []string
	for i, line := range lines {
		if i < 2 {
			continue // 跳过标题行
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			snapshots = append(snapshots, fields[1]) // TAG 列
		}
	}

	return snapshots, nil
}
