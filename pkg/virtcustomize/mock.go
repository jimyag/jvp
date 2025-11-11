package virtcustomize

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockClient 是 virt-customize 客户端的 mock 实现
type MockClient struct {
	mock.Mock
}

// 确保 MockClient 实现了 VirtCustomizeClient 接口
var _ VirtCustomizeClient = (*MockClient)(nil)

// ResetPassword 重置单个用户的密码
func (m *MockClient) ResetPassword(ctx context.Context, diskPath string, username, password string) error {
	args := m.Called(ctx, diskPath, username, password)
	return args.Error(0)
}

// ResetMultiplePasswords 重置多个用户的密码
func (m *MockClient) ResetMultiplePasswords(ctx context.Context, diskPath string, users map[string]string) error {
	args := m.Called(ctx, diskPath, users)
	return args.Error(0)
}

// ValidateDiskPath 验证磁盘路径是否有效
func (m *MockClient) ValidateDiskPath(diskPath string) error {
	args := m.Called(diskPath)
	return args.Error(0)
}

// SetTimeout 设置命令超时时间
func (m *MockClient) SetTimeout(timeout time.Duration) {
	m.Called(timeout)
}
