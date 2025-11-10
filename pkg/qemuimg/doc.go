// Package qemuimg 封装 qemu-img 命令行工具的操作
//
// 该包提供了对 qemu-img 常用操作的封装，包括：
//   - 从 backing file 创建镜像（CreateFromBackingFile）
//   - 调整镜像大小（Resize）
//   - 转换镜像格式（Convert）
//   - 获取镜像信息（Info）
//   - 检查镜像完整性（Check）
//   - 创建空镜像（CreateEmpty）
//
// 所有操作都支持 context 超时控制，适合长时间运行的操作。
//
// 示例：
//
//	// 创建 client
//	client := qemuimg.New("")
//
//	// 从 backing file 创建新镜像
//	err := client.CreateFromBackingFile(ctx, "qcow2", "qcow2",
//		"/path/to/base.qcow2", "/path/to/new.qcow2")
//
//	// 调整镜像大小
//	err = client.Resize(ctx, "/path/to/image.qcow2", 20)
//
//	// 转换镜像格式
//	err = client.Convert(ctx, "qcow2", "raw",
//		"/path/to/input.qcow2", "/path/to/output.raw")
//
//	// 获取镜像信息
//	info, err := client.Info(ctx, "/path/to/image.qcow2")
//
//	// 检查镜像完整性
//	err = client.Check(ctx, "/path/to/image.qcow2", "qcow2")
//
//	// 创建空镜像
//	err = client.CreateEmpty(ctx, "qcow2", "/path/to/new.qcow2", 10)
package qemuimg
