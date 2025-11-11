package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jimyag/jvp/pkg/cloudinit"
	"github.com/jimyag/jvp/pkg/libvirt"
	"github.com/jimyag/jvp/pkg/virtcustomize"
	"github.com/rs/zerolog"
)

// PasswordResetStrategy 密码重置策略接口
type PasswordResetStrategy interface {
	ResetPassword(ctx context.Context, instanceID string, users map[string]string) error
	Name() string
}

// QemuGuestAgentStrategy qemu-guest-agent 密码重置策略
type QemuGuestAgentStrategy struct {
	libvirtClient libvirt.LibvirtClient
}

func NewQemuGuestAgentStrategy(libvirtClient libvirt.LibvirtClient) *QemuGuestAgentStrategy {
	return &QemuGuestAgentStrategy{
		libvirtClient: libvirtClient,
	}
}

func (s *QemuGuestAgentStrategy) Name() string {
	return "qemu-guest-agent"
}

// ResetPassword 使用 qemu-guest-agent 重置密码
func (s *QemuGuestAgentStrategy) ResetPassword(ctx context.Context, instanceID string, users map[string]string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instance_id", instanceID).
		Str("strategy", s.Name()).
		Msg("Attempting password reset via qemu-guest-agent")

	// 获取 domain
	domain, err := s.libvirtClient.GetDomainByName(instanceID)
	if err != nil {
		return fmt.Errorf("get domain: %w", err)
	}

	// 检查 guest agent 是否可用
	available, err := s.libvirtClient.CheckGuestAgentAvailable(domain)
	if err != nil {
		return fmt.Errorf("check guest agent: %w", err)
	}
	if !available {
		return fmt.Errorf("guest agent not available")
	}

	// 为每个用户重置密码
	for username, password := range users {
		// 使用 guest-exec 执行 passwd 命令
		// 注意：passwd 需要交互式输入，我们使用 echo 和管道
		// 或者使用 chpasswd 命令（如果可用）
		cmd := fmt.Sprintf(`echo "%s:%s" | chpasswd`, username, password)

		// 构建 guest-exec 命令
		execCmd := map[string]interface{}{
			"execute": "guest-exec",
			"arguments": map[string]interface{}{
				"path": "/bin/sh",
				"arg":  []string{"-c", cmd},
			},
		}

		cmdJSON, err := json.Marshal(execCmd)
		if err != nil {
			return fmt.Errorf("marshal command: %w", err)
		}

		// 执行命令
		result, err := s.libvirtClient.QemuAgentCommand(domain, string(cmdJSON), 30, 0)
		if err != nil {
			logger.Error().
				Err(err).
				Str("username", username).
				Msg("Failed to reset password via guest agent")
			return fmt.Errorf("execute guest command: %w", err)
		}

		// 解析结果，检查是否有错误
		var execResult map[string]interface{}
		if err := json.Unmarshal([]byte(result), &execResult); err != nil {
			logger.Warn().
				Err(err).
				Str("result", result).
				Msg("Failed to parse guest agent result")
			// 继续执行，因为命令可能已经成功
		}

		// 检查是否有错误
		if errorObj, ok := execResult["error"].(map[string]interface{}); ok {
			errorMsg, _ := errorObj["desc"].(string)
			return fmt.Errorf("guest agent error: %s", errorMsg)
		}

		logger.Info().
			Str("username", username).
			Msg("Password reset via guest agent successful")
	}

	return nil
}

// CloudInitStrategy cloud-init 密码重置策略
type CloudInitStrategy struct {
	libvirtClient libvirt.LibvirtClient
	tempDir       string
}

func NewCloudInitStrategy(libvirtClient libvirt.LibvirtClient, tempDir string) *CloudInitStrategy {
	return &CloudInitStrategy{
		libvirtClient: libvirtClient,
		tempDir:       tempDir,
	}
}

func (s *CloudInitStrategy) Name() string {
	return "cloud-init"
}

// ResetPassword 使用 cloud-init 重置密码
func (s *CloudInitStrategy) ResetPassword(ctx context.Context, instanceID string, users map[string]string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("instance_id", instanceID).
		Str("strategy", s.Name()).
		Msg("Attempting password reset via cloud-init")

	// 构建 chpasswd 列表
	var chpasswdList []string
	for username, password := range users {
		chpasswdList = append(chpasswdList, fmt.Sprintf("%s:%s", username, password))
	}

	// 创建 cloud-init user-data
	userData := &cloudinit.UserData{
		ChPasswd: &cloudinit.ChPasswd{
			List:   chpasswdList,
			Expire: false,
		},
	}

	// 使用 ISOBuilder 生成 cloud-init ISO
	builder := cloudinit.NewISOBuilder()

	tempDir := s.tempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// 生成 ISO
	isoPath, err := builder.BuildISO(&cloudinit.BuildOptions{
		VMName:    instanceID,
		OutputDir: tempDir,
		Config: &cloudinit.Config{
			Hostname: instanceID,
		},
		UserData: userData,
	})
	if err != nil {
		return fmt.Errorf("build cloud-init ISO: %w", err)
	}

	logger.Info().
		Str("iso_path", isoPath).
		Msg("Cloud-init ISO generated")

	// 注意：cloud-init 方案需要将 ISO 注入到实例并重启
	// 这需要修改 domain XML，添加 CDROM 设备
	// 由于实现复杂度较高，这里返回错误，让调用方回退到 virt-customize
	return fmt.Errorf("cloud-init injection requires domain XML modification and instance restart, falling back to virt-customize")
}

// VirtCustomizeStrategy virt-customize 密码重置策略
type VirtCustomizeStrategy struct {
	virtCustomizeClient virtcustomize.VirtCustomizeClient
	libvirtClient       libvirt.LibvirtClient
}

func NewVirtCustomizeStrategy(virtCustomizeClient virtcustomize.VirtCustomizeClient, libvirtClient libvirt.LibvirtClient) *VirtCustomizeStrategy {
	return &VirtCustomizeStrategy{
		virtCustomizeClient: virtCustomizeClient,
		libvirtClient:       libvirtClient,
	}
}

func (s *VirtCustomizeStrategy) Name() string {
	return "virt-customize"
}

// ResetPassword 使用 virt-customize 重置密码
// diskPath 参数由调用方传入，避免重复调用 GetDomainDisks
func (s *VirtCustomizeStrategy) ResetPassword(ctx context.Context, diskPath string, users map[string]string) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("disk_path", diskPath).
		Str("strategy", s.Name()).
		Msg("Attempting password reset via virt-customize")

	if s.virtCustomizeClient == nil {
		return fmt.Errorf("virt-customize client not available")
	}

	// 验证磁盘路径
	if err := s.virtCustomizeClient.ValidateDiskPath(diskPath); err != nil {
		return fmt.Errorf("validate disk path: %w", err)
	}

	// 调用 virt-customize 重置密码
	err := s.virtCustomizeClient.ResetMultiplePasswords(ctx, diskPath, users)
	if err != nil {
		return fmt.Errorf("virt-customize reset passwords: %w", err)
	}

	logger.Info().
		Str("disk_path", diskPath).
		Msg("Password reset via virt-customize successful")

	return nil
}
