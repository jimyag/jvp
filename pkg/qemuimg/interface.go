package qemuimg

import "context"

// QemuImgClient 定义了 qemu-img 客户端的接口
// 用于抽象 qemu-img 操作，便于测试和 mock
type QemuImgClient interface {
	// CreateFromBackingFile 从 backing file 创建新镜像
	CreateFromBackingFile(ctx context.Context, format, backingFormat, backingFile, outputFile string) error
	// Resize 调整镜像大小
	Resize(ctx context.Context, imagePath string, sizeGB uint64) error
	// Convert 转换镜像格式或复制镜像
	Convert(ctx context.Context, inputFormat, outputFormat, inputFile, outputFile string) error
	// Info 获取镜像信息
	Info(ctx context.Context, imagePath string) (string, error)
	// Check 检查镜像完整性
	Check(ctx context.Context, imagePath, format string) error
	// CreateEmpty 创建空镜像
	CreateEmpty(ctx context.Context, format, outputFile string, sizeGB uint64) error
	// Snapshot 创建快照
	Snapshot(ctx context.Context, imagePath, snapshotName string) error
	// DeleteSnapshot 删除快照
	DeleteSnapshot(ctx context.Context, imagePath, snapshotName string) error
	// ListSnapshots 列出镜像的所有快照
	ListSnapshots(ctx context.Context, imagePath string) ([]string, error)
}
