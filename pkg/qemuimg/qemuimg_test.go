package qemuimg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("default path", func(t *testing.T) {
		t.Parallel()
		client := New("")
		assert.Equal(t, "qemu-img", client.qemuImgPath)
		assert.Equal(t, 30*time.Minute, client.timeout)
	})

	t.Run("custom path", func(t *testing.T) {
		t.Parallel()
		client := New("/usr/local/bin/qemu-img")
		assert.Equal(t, "/usr/local/bin/qemu-img", client.qemuImgPath)
	})

	t.Run("with timeout", func(t *testing.T) {
		t.Parallel()
		client := New("").WithTimeout(60 * time.Minute)
		assert.Equal(t, 60*time.Minute, client.timeout)
	})
}

func TestClient_CreateEmpty(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	testcases := []struct {
		name     string
		format   string
		sizeGB   uint64
		wantErr  bool
		checkErr func(t *testing.T, err error)
	}{
		{
			name:    "create qcow2 image",
			format:  "qcow2",
			sizeGB:  1,
			wantErr: false,
		},
		{
			name:    "create raw image",
			format:  "raw",
			sizeGB:  1,
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "invalid",
			sizeGB:  1,
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := New("")
			ctx := context.Background()

			tmpDir := t.TempDir()
			outputFile := filepath.Join(tmpDir, "test."+tc.format)

			err := client.CreateEmpty(ctx, tc.format, outputFile, tc.sizeGB)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.checkErr != nil {
					tc.checkErr(t, err)
				}
			} else {
				assert.NoError(t, err)
				// 检查文件是否存在
				_, err := os.Stat(outputFile)
				assert.NoError(t, err, "output file should exist")
			}
		})
	}
}

func TestClient_Info(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	testcases := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "get info for existing image",
			setup: func(t *testing.T) string {
				client := New("")
				ctx := context.Background()
				tmpDir := t.TempDir()
				imagePath := filepath.Join(tmpDir, "test.qcow2")

				// 先创建一个镜像
				err := client.CreateEmpty(ctx, "qcow2", imagePath, 1)
				require.NoError(t, err)

				return imagePath
			},
			wantErr: false,
		},
		{
			name: "get info for non-existing image",
			setup: func(t *testing.T) string {
				return "/nonexistent/image.qcow2"
			},
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := New("")
			ctx := context.Background()

			imagePath := tc.setup(t)

			info, err := client.Info(ctx, imagePath)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Empty(t, info)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, info)
				assert.Contains(t, info, "file format")
			}
		})
	}
}

func TestClient_CreateFromBackingFile(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	client := New("")
	ctx := context.Background()

	tmpDir := t.TempDir()
	baseImage := filepath.Join(tmpDir, "base.qcow2")
	newImage := filepath.Join(tmpDir, "new.qcow2")

	// 先创建 base image
	err := client.CreateEmpty(ctx, "qcow2", baseImage, 1)
	require.NoError(t, err)

	// 从 backing file 创建新镜像
	err = client.CreateFromBackingFile(ctx, "qcow2", "qcow2", baseImage, newImage)
	assert.NoError(t, err)

	// 检查新镜像是否存在
	_, err = os.Stat(newImage)
	assert.NoError(t, err, "new image should exist")

	// 检查新镜像的信息，应该包含 backing file 信息
	info, err := client.Info(ctx, newImage)
	assert.NoError(t, err)
	assert.Contains(t, info, "backing file")
}

func TestClient_Resize(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	client := New("")
	ctx := context.Background()

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "test.qcow2")

	// 先创建 1GB 的镜像
	err := client.CreateEmpty(ctx, "qcow2", imagePath, 1)
	require.NoError(t, err)

	// 调整到 2GB
	err = client.Resize(ctx, imagePath, 2)
	assert.NoError(t, err)

	// 验证大小
	info, err := client.Info(ctx, imagePath)
	assert.NoError(t, err)
	assert.Contains(t, info, "virtual size")
}

func TestClient_Convert(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	client := New("")
	ctx := context.Background()

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.qcow2")
	outputFile := filepath.Join(tmpDir, "output.raw")

	// 先创建 qcow2 镜像
	err := client.CreateEmpty(ctx, "qcow2", inputFile, 1)
	require.NoError(t, err)

	// 转换为 raw 格式
	err = client.Convert(ctx, "qcow2", "raw", inputFile, outputFile)
	assert.NoError(t, err)

	// 检查输出文件是否存在
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "output file should exist")
}

func TestClient_Check(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	client := New("")
	ctx := context.Background()

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "test.qcow2")

	// 先创建镜像
	err := client.CreateEmpty(ctx, "qcow2", imagePath, 1)
	require.NoError(t, err)

	// 检查镜像完整性
	err = client.Check(ctx, imagePath, "qcow2")
	assert.NoError(t, err)
}

func TestClient_ContextTimeout(t *testing.T) {
	// 检查 qemu-img 是否可用
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not found in PATH, skipping test")
	}

	t.Parallel()

	client := New("").WithTimeout(1 * time.Nanosecond) // 极短的超时时间
	ctx := context.Background()

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "test.qcow2")

	// 这个操作应该会因为超时而失败
	err := client.CreateEmpty(ctx, "qcow2", imagePath, 1)
	assert.Error(t, err)
	// Go 的 context 超时错误是 "context deadline exceeded"
	assert.Contains(t, err.Error(), "deadline exceeded")
}
