package qemuimg

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockClient 是 QemuImgClient 的 mock 实现
// 用于测试，不需要真实的 qemu-img 命令
type MockClient struct {
	mock.Mock
}

// NewMockClient 创建新的 MockClient
func NewMockClient() *MockClient {
	return &MockClient{}
}

// CreateFromBackingFile 实现 QemuImgClient 接口
func (m *MockClient) CreateFromBackingFile(ctx context.Context, format, backingFormat, backingFile, outputFile string) error {
	args := m.Called(ctx, format, backingFormat, backingFile, outputFile)
	return args.Error(0)
}

// Resize 实现 QemuImgClient 接口
func (m *MockClient) Resize(ctx context.Context, imagePath string, sizeGB uint64) error {
	args := m.Called(ctx, imagePath, sizeGB)
	return args.Error(0)
}

// Convert 实现 QemuImgClient 接口
func (m *MockClient) Convert(ctx context.Context, inputFormat, outputFormat, inputFile, outputFile string) error {
	args := m.Called(ctx, inputFormat, outputFormat, inputFile, outputFile)
	return args.Error(0)
}

// Info 实现 QemuImgClient 接口
func (m *MockClient) Info(ctx context.Context, imagePath string) (string, error) {
	args := m.Called(ctx, imagePath)
	return args.String(0), args.Error(1)
}

// Check 实现 QemuImgClient 接口
func (m *MockClient) Check(ctx context.Context, imagePath, format string) error {
	args := m.Called(ctx, imagePath, format)
	return args.Error(0)
}

// CreateEmpty 实现 QemuImgClient 接口
func (m *MockClient) CreateEmpty(ctx context.Context, format, outputFile string, sizeGB uint64) error {
	args := m.Called(ctx, format, outputFile, sizeGB)
	return args.Error(0)
}

// Snapshot 实现 QemuImgClient 接口
func (m *MockClient) Snapshot(ctx context.Context, imagePath, snapshotName string) error {
	args := m.Called(ctx, imagePath, snapshotName)
	return args.Error(0)
}

// DeleteSnapshot 实现 QemuImgClient 接口
func (m *MockClient) DeleteSnapshot(ctx context.Context, imagePath, snapshotName string) error {
	args := m.Called(ctx, imagePath, snapshotName)
	return args.Error(0)
}

// ListSnapshots 实现 QemuImgClient 接口
func (m *MockClient) ListSnapshots(ctx context.Context, imagePath string) ([]string, error) {
	args := m.Called(ctx, imagePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
