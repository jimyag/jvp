package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	libvirtlib "github.com/digitalocean/go-libvirt"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/repository"
	"github.com/jimyag/jvp/internal/jvp/repository/model"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstanceService_ResetPassword(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		req           *entity.ResetPasswordRequest
		setupInstance func(*testing.T, *TestServices) *entity.Instance
		mockSetup     func(*testing.T, *TestServices, *entity.Instance)
		expectError   bool
		errorContains string
		validate      func(*testing.T, *entity.ResetPasswordResponse, string) // 添加 strategy 参数
	}{
		{
			name: "reset password for stopped instance",
			req: &entity.ResetPasswordRequest{
				InstanceID: "i-123",
				Users: []entity.PasswordReset{
					{
						Username:    "root",
						NewPassword: "NewPassword123!",
					},
				},
				AutoStart: false,
			},
			setupInstance: func(t *testing.T, services *TestServices) *entity.Instance {
				// 创建测试实例（停止状态）
				instance := &entity.Instance{
					ID:         "i-123",
					Name:       "test-instance",
					State:      "stopped",
					ImageID:    "ami-123",
					VolumeID:   "vol-123",
					MemoryMB:   2048,
					VCPUs:      2,
					CreatedAt:  "2024-01-01T00:00:00Z",
					DomainUUID: "uuid-123",
					DomainName: "i-123",
				}
				return instance
			},
			mockSetup: func(t *testing.T, services *TestServices, instance *entity.Instance) {
				// 在数据库中创建实例
				instanceRepo := repository.NewInstanceRepository(services.Repo.DB())
				instanceModel := &model.Instance{
					ID:        "i-123",
					Name:      "test-instance",
					State:     "stopped",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(context.Background(), instanceModel)
				require.NoError(t, err)

				// Mock GetDomainDisks
				diskPath := filepath.Join(services.TempDir, "vol-123.qcow2")
				file, err := os.Create(diskPath)
				require.NoError(t, err)
				file.Close()

				services.MockLibvirt.On("GetDomainDisks", "i-123").Return([]libvirt.DomainDisk{
					{
						Source: libvirt.DomainDiskSource{
							File: diskPath,
						},
					},
				}, nil).Once()

				// Mock virt-customize
				// 注意：ValidateDiskPath 现在在 VirtCustomizeStrategy.ResetPassword 中调用，所以需要 mock
				mockVirtCustomize := new(virtcustomize.MockClient)
				mockVirtCustomize.On("ValidateDiskPath", mock.AnythingOfType("string")).Return(nil).Once()
				mockVirtCustomize.On("ResetMultiplePasswords", mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(users map[string]string) bool {
					return users["root"] == "NewPassword123!"
				})).Return(nil).Once()
				services.InstanceService.virtCustomizeClient = mockVirtCustomize
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.ResetPasswordResponse, strategy string) {
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Equal(t, "i-123", resp.InstanceID)
				assert.Contains(t, resp.Users, "root")
				// 停止状态的实例应该使用 virt-customize
				assert.Equal(t, "virt-customize", strategy)
			},
		},
		{
			name: "reset password for running instance with qemu-guest-agent",
			req: &entity.ResetPasswordRequest{
				InstanceID: "i-123",
				Users: []entity.PasswordReset{
					{
						Username:    "root",
						NewPassword: "NewPassword123!",
					},
				},
				AutoStart: false,
			},
			setupInstance: func(t *testing.T, services *TestServices) *entity.Instance {
				return &entity.Instance{
					ID:         "i-123",
					Name:       "test-instance",
					State:      "running",
					ImageID:    "ami-123",
					VolumeID:   "vol-123",
					MemoryMB:   2048,
					VCPUs:      2,
					CreatedAt:  "2024-01-01T00:00:00Z",
					DomainUUID: "uuid-123",
					DomainName: "i-123",
				}
			},
			mockSetup: func(t *testing.T, services *TestServices, instance *entity.Instance) {
				// 在数据库中创建实例（运行状态）
				instanceRepo := repository.NewInstanceRepository(services.Repo.DB())
				instanceModel := &model.Instance{
					ID:        "i-123",
					Name:      "test-instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(context.Background(), instanceModel)
				require.NoError(t, err)

				// Mock qemu-guest-agent 策略
				domain := libvirtlib.Domain{
					Name: "i-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				}
				services.MockLibvirt.On("GetDomainByName", "i-123").Return(domain, nil).Maybe()
				services.MockLibvirt.On("CheckGuestAgentAvailable", domain).Return(true, nil).Once()
				services.MockLibvirt.On("QemuAgentCommand", domain, mock.AnythingOfType("string"), uint32(30), uint32(0)).Return(`{"return":{"pid":12345}}`, nil).Once()
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.ResetPasswordResponse, strategy string) {
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Equal(t, "qemu-guest-agent", strategy)
			},
		},
		{
			name: "reset password for running instance fallback to virt-customize",
			req: &entity.ResetPasswordRequest{
				InstanceID: "i-123",
				Users: []entity.PasswordReset{
					{
						Username:    "root",
						NewPassword: "NewPassword123!",
					},
				},
				AutoStart: true,
			},
			setupInstance: func(t *testing.T, services *TestServices) *entity.Instance {
				return &entity.Instance{
					ID:         "i-123",
					Name:       "test-instance",
					State:      "running",
					ImageID:    "ami-123",
					VolumeID:   "vol-123",
					MemoryMB:   2048,
					VCPUs:      2,
					CreatedAt:  "2024-01-01T00:00:00Z",
					DomainUUID: "uuid-123",
					DomainName: "i-123",
				}
			},
			mockSetup: func(t *testing.T, services *TestServices, instance *entity.Instance) {
				// 在数据库中创建实例（运行状态）
				instanceRepo := repository.NewInstanceRepository(services.Repo.DB())
				instanceModel := &model.Instance{
					ID:        "i-123",
					Name:      "test-instance",
					State:     "running",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(context.Background(), instanceModel)
				require.NoError(t, err)

				// Mock qemu-guest-agent 不可用
				domain := libvirtlib.Domain{
					Name: "i-123",
					UUID: libvirtlib.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				}
				services.MockLibvirt.On("GetDomainByName", "i-123").Return(domain, nil).Maybe()
				services.MockLibvirt.On("CheckGuestAgentAvailable", domain).Return(false, nil).Once()

				// Mock StopInstances（ResetPassword 内部会调用 StopInstances）
				services.MockLibvirt.On("StopDomain", domain).Return(nil).Maybe()

				// Mock GetDomainDisks（会在停止后调用）
				diskPath := filepath.Join(services.TempDir, "vol-123.qcow2")
				file, err := os.Create(diskPath)
				require.NoError(t, err)
				file.Close()

				services.MockLibvirt.On("GetDomainDisks", "i-123").Return([]libvirt.DomainDisk{
					{
						Source: libvirt.DomainDiskSource{
							File: diskPath,
						},
					},
				}, nil).Maybe()

				// Mock StartInstances（如果 AutoStart=true）
				services.MockLibvirt.On("StartDomain", domain).Return(nil).Maybe()

				// Mock virt-customize
				// 注意：ValidateDiskPath 现在在 VirtCustomizeStrategy.ResetPassword 中调用，所以需要 mock
				mockVirtCustomize := new(virtcustomize.MockClient)
				mockVirtCustomize.On("ValidateDiskPath", mock.AnythingOfType("string")).Return(nil).Once()
				mockVirtCustomize.On("ResetMultiplePasswords", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Once()
				services.InstanceService.virtCustomizeClient = mockVirtCustomize
			},
			expectError: false,
			validate: func(t *testing.T, resp *entity.ResetPasswordResponse, strategy string) {
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Equal(t, "virt-customize", strategy)
			},
		},
		{
			name: "instance not found",
			req: &entity.ResetPasswordRequest{
				InstanceID: "i-not-found",
				Users: []entity.PasswordReset{
					{
						Username:    "root",
						NewPassword: "NewPassword123!",
					},
				},
			},
			setupInstance: nil,
			mockSetup: func(t *testing.T, services *TestServices, instance *entity.Instance) {
				// GetInstance 会先尝试从数据库查询（会失败），然后尝试从 libvirt 查询
				// Mock libvirt 查询也失败
				services.MockLibvirt.On("GetDomainByName", "i-not-found").Return(libvirtlib.Domain{}, fmt.Errorf("domain not found")).Maybe()
			},
			expectError:   true,
			errorContains: "not found",
		},
		{
			name: "virt-customize client not available",
			req: &entity.ResetPasswordRequest{
				InstanceID: "i-123",
				Users: []entity.PasswordReset{
					{
						Username:    "root",
						NewPassword: "NewPassword123!",
					},
				},
			},
			setupInstance: func(t *testing.T, services *TestServices) *entity.Instance {
				return &entity.Instance{
					ID:    "i-123",
					State: "stopped",
				}
			},
			mockSetup: func(t *testing.T, services *TestServices, instance *entity.Instance) {
				// 在数据库中创建实例
				instanceRepo := repository.NewInstanceRepository(services.Repo.DB())
				instanceModel := &model.Instance{
					ID:        "i-123",
					Name:      "test-instance",
					State:     "stopped",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(context.Background(), instanceModel)
				require.NoError(t, err)

				// GetInstance 可能会调用 GetDomainByName（如果数据库中没有），所以需要 mock
				services.MockLibvirt.On("GetDomainByName", "i-123").Return(libvirtlib.Domain{}, nil).Maybe()

				// 设置 virtCustomizeClient 为 nil
				services.InstanceService.virtCustomizeClient = nil
			},
			expectError:   true,
			errorContains: "virt-customize command not found",
		},
		{
			name: "instance has no disk",
			req: &entity.ResetPasswordRequest{
				InstanceID: "i-123",
				Users: []entity.PasswordReset{
					{
						Username:    "root",
						NewPassword: "NewPassword123!",
					},
				},
			},
			setupInstance: func(t *testing.T, services *TestServices) *entity.Instance {
				return &entity.Instance{
					ID:    "i-123",
					State: "stopped",
				}
			},
			mockSetup: func(t *testing.T, services *TestServices, instance *entity.Instance) {
				// 在数据库中创建实例
				instanceRepo := repository.NewInstanceRepository(services.Repo.DB())
				instanceModel := &model.Instance{
					ID:        "i-123",
					Name:      "test-instance",
					State:     "stopped",
					ImageID:   "ami-123",
					VolumeID:  "vol-123",
					MemoryMB:  2048,
					VCPUs:     2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := instanceRepo.Create(context.Background(), instanceModel)
				require.NoError(t, err)

				// GetInstance 可能会调用 GetDomainByName（如果数据库中没有），所以需要 mock
				services.MockLibvirt.On("GetDomainByName", "i-123").Return(libvirtlib.Domain{}, nil).Maybe()

				// Mock GetDomainDisks 返回空列表（没有磁盘）
				services.MockLibvirt.On("GetDomainDisks", "i-123").Return([]libvirt.DomainDisk{}, nil).Once()
				mockVirtCustomize := new(virtcustomize.MockClient)
				services.InstanceService.virtCustomizeClient = mockVirtCustomize
			},
			expectError:   true,
			errorContains: "has no disk",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			services := setupTestServices(t)
			ctx := context.Background()

			var instance *entity.Instance
			if tc.setupInstance != nil {
				instance = tc.setupInstance(t, services)
			}

			if tc.mockSetup != nil {
				tc.mockSetup(t, services, instance)
			}

			resp, err := services.InstanceService.ResetPassword(ctx, tc.req)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.validate != nil {
				// 从响应消息中提取策略名称
				strategy := "virt-customize" // 默认
				if resp.Message != "" {
					// 消息格式：Password reset successfully via {strategy} 或 Password reset successfully via {strategy} (instance restart required)
					prefix := "Password reset successfully via "
					if strings.HasPrefix(resp.Message, prefix) {
						strategy = strings.TrimPrefix(resp.Message, prefix)
						// 移除可能的后缀，如 " (instance restart required)"
						if idx := strings.Index(strategy, " ("); idx >= 0 {
							strategy = strategy[:idx]
						}
						// 移除末尾空格
						strategy = strings.TrimSpace(strategy)
					}
				}
				tc.validate(t, resp, strategy)
			}
		})
	}
}
