package qemuimg_test

import (
	"context"
	"fmt"
	"os"

	"github.com/jimyag/jvp/pkg/qemuimg"
)

func ExampleClient_CreateFromBackingFile() {
	client := qemuimg.New("")
	ctx := context.Background()

	// 从 backing file 创建新镜像（增量镜像）
	err := client.CreateFromBackingFile(ctx, "qcow2", "qcow2",
		"/path/to/base-image.qcow2",
		"/path/to/new-instance.qcow2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Image created successfully")
}

func ExampleClient_Resize() {
	client := qemuimg.New("")
	ctx := context.Background()

	// 调整镜像大小到 20GB
	err := client.Resize(ctx, "/path/to/image.qcow2", 20)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Image resized successfully")
}

func ExampleClient_Convert() {
	client := qemuimg.New("")
	ctx := context.Background()

	// 将 qcow2 格式转换为 raw 格式
	err := client.Convert(ctx, "qcow2", "raw",
		"/path/to/input.qcow2",
		"/path/to/output.raw")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Image converted successfully")
}

func ExampleClient_Info() {
	client := qemuimg.New("")
	ctx := context.Background()

	// 获取镜像信息
	info, err := client.Info(ctx, "/path/to/image.qcow2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Image info:")
	fmt.Println(info)
}

func ExampleClient_Check() {
	client := qemuimg.New("")
	ctx := context.Background()

	// 检查镜像完整性
	err := client.Check(ctx, "/path/to/image.qcow2", "qcow2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Image check passed")
}

func ExampleClient_CreateEmpty() {
	client := qemuimg.New("")
	ctx := context.Background()

	// 创建 10GB 的空 qcow2 镜像
	err := client.CreateEmpty(ctx, "qcow2", "/path/to/new.qcow2", 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Empty image created successfully")
}

func ExampleClient_WithTimeout() {
	// 创建 client 并设置自定义超时时间
	client := qemuimg.New("").WithTimeout(60 * 60) // 1 小时超时
	ctx := context.Background()

	// 对于大文件操作，使用更长的超时时间
	err := client.Convert(ctx, "qcow2", "qcow2",
		"/path/to/large-input.qcow2",
		"/path/to/large-output.qcow2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Large image converted successfully")
}

func ExampleNew() {
	// 使用默认的 qemu-img 路径
	client := qemuimg.New("")

	// 或者指定自定义路径
	customClient := qemuimg.New("/usr/local/bin/qemu-img")

	// 检查 qemu-img 是否可用
	ctx := context.Background()
	_, err := client.Info(ctx, "/dev/null")
	if err != nil {
		// qemu-img 可能不可用
		fmt.Printf("qemu-img may not be available: %v\n", err)
		return
	}

	fmt.Println("qemu-img is available")
	_ = customClient
}

// 这个示例展示了如何在实际场景中使用 qemuimg 包
// 从镜像创建实例磁盘的完整流程
func ExampleClient_createVolumeFromImage() {
	client := qemuimg.New("")
	ctx := context.Background()

	imagePath := "/var/lib/jvp/images/ubuntu-22.04.qcow2"
	volumePath := "/var/lib/jvp/images/vol-12345.qcow2"
	imageSizeGB := uint64(10)  // 镜像大小
	targetSizeGB := uint64(20) // 目标大小

	// 策略 1: 如果镜像大小 <= 目标大小，使用 backing file（节省空间）
	if imageSizeGB <= targetSizeGB {
		// 从 backing file 创建
		err := client.CreateFromBackingFile(ctx, "qcow2", "qcow2",
			imagePath, volumePath)
		if err != nil {
			fmt.Printf("Failed to create from backing file: %v\n", err)
			return
		}

		// 如果需要调整大小
		if imageSizeGB < targetSizeGB {
			err = client.Resize(ctx, volumePath, targetSizeGB)
			if err != nil {
				// 清理已创建的文件
				os.Remove(volumePath)
				fmt.Printf("Failed to resize: %v\n", err)
				return
			}
		}
	} else {
		// 策略 2: 如果镜像大小 > 目标大小，完整复制
		err := client.Convert(ctx, "qcow2", "qcow2",
			imagePath, volumePath)
		if err != nil {
			fmt.Printf("Failed to convert: %v\n", err)
			return
		}

		// 调整大小
		err = client.Resize(ctx, volumePath, targetSizeGB)
		if err != nil {
			os.Remove(volumePath)
			fmt.Printf("Failed to resize: %v\n", err)
			return
		}
	}

	fmt.Println("Volume created successfully from image")
}
