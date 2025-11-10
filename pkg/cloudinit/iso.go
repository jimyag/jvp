package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ISOBuilder cloud-init ISO 构建器
type ISOBuilder struct {
	generator *Generator
}

// NewISOBuilder 创建新的 ISO 构建器
func NewISOBuilder() *ISOBuilder {
	return &ISOBuilder{
		generator: NewGenerator(),
	}
}

// BuildOptions ISO 构建选项
type BuildOptions struct {
	VMName      string    // 虚拟机名称（用于生成 ISO 文件名）
	OutputDir   string    // 输出目录（默认：/var/lib/jvp/images）
	Config      *Config   // cloud-init 配置
	UserData    *UserData // cloud-init 用户数据
	KeepTempDir bool      // 是否保留临时目录（用于调试）
}

// BuildISO 生成 cloud-init ISO 镜像
// 返回生成的 ISO 文件路径
func (b *ISOBuilder) BuildISO(opts *BuildOptions) (string, error) {
	if opts.Config == nil {
		return "", fmt.Errorf("config is required")
	}

	if opts.VMName == "" {
		return "", fmt.Errorf("VM name is required")
	}

	// 设置默认输出目录
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "/var/lib/jvp/images"
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "cloudinit-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	if !opts.KeepTempDir {
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()
	}

	// 生成 meta-data
	metaData, err := b.generator.GenerateMetaData(opts.Config.Hostname)
	if err != nil {
		return "", fmt.Errorf("failed to generate meta-data: %v", err)
	}

	metaDataPath := filepath.Join(tmpDir, "meta-data")
	if err := os.WriteFile(metaDataPath, []byte(metaData), 0o600); err != nil {
		return "", fmt.Errorf("failed to write meta-data: %v", err)
	}

	// 生成 user-data
	userData, err := b.generator.GenerateUserData(opts.Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate user-data: %v", err)
	}

	if opts.UserData != nil {
		userData, err = b.generator.GenerateUserDataFromStruct(opts.UserData)
		if err != nil {
			return "", fmt.Errorf("failed to generate user-data: %v", err)
		}
	}

	userDataPath := filepath.Join(tmpDir, "user-data")
	if err := os.WriteFile(userDataPath, []byte(userData), 0o600); err != nil {
		return "", fmt.Errorf("failed to write user-data: %v", err)
	}

	// 生成 network-config（如果有网络配置）
	if opts.Config.Network != nil {
		networkConfig, err := b.generator.GenerateNetworkConfig(opts.Config.Network)
		if err != nil {
			return "", fmt.Errorf("failed to generate network-config: %v", err)
		}

		networkConfigPath := filepath.Join(tmpDir, "network-config")
		if err := os.WriteFile(networkConfigPath, []byte(networkConfig), 0o600); err != nil {
			return "", fmt.Errorf("failed to write network-config: %v", err)
		}
	}

	// 生成 ISO 文件
	isoPath := filepath.Join(outputDir, fmt.Sprintf("%s-cidata.iso", opts.VMName))

	// 使用 genisoimage 或 mkisofs 生成 ISO
	var cmd *exec.Cmd
	if _, err := exec.LookPath("genisoimage"); err == nil {
		cmd = exec.Command("genisoimage",
			"-output", isoPath,
			"-volid", "cidata",
			"-joliet",
			"-rock",
			tmpDir,
		)
	} else if _, err := exec.LookPath("mkisofs"); err == nil {
		cmd = exec.Command("mkisofs",
			"-output", isoPath,
			"-volid", "cidata",
			"-joliet",
			"-rock",
			tmpDir,
		)
	} else {
		return "", fmt.Errorf("neither genisoimage nor mkisofs found, please install one of them")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create ISO: %w, output: %s", err, string(output))
	}

	return isoPath, nil
}

// CleanupISO 清理 cloud-init ISO 文件
func (b *ISOBuilder) CleanupISO(vmName string, outputDir string) error {
	if outputDir == "" {
		outputDir = "/var/lib/jvp/images"
	}

	isoPath := filepath.Join(outputDir, fmt.Sprintf("%s-cidata.iso", vmName))
	if err := os.Remove(isoPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cloud-init ISO: %v", err)
	}
	return nil
}

// GetISOPath 获取 cloud-init ISO 路径
func (b *ISOBuilder) GetISOPath(vmName string, outputDir string) string {
	if outputDir == "" {
		outputDir = "/var/lib/jvp/images"
	}
	return filepath.Join(outputDir, fmt.Sprintf("%s-cidata.iso", vmName))
}
