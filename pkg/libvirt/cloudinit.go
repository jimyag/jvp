package libvirt

import (
	"os"

	"github.com/jimyag/jvp/pkg/cloudinit"
)

// generateCloudInitISO 生成 cloud-init 配置 ISO 镜像
// 返回生成的 ISO 文件路径
func (c *Client) generateCloudInitISO(config *CreateVMConfig) (string, error) {
	if config.CloudInit == nil {
		return "", nil
	}

	// 设置默认 hostname
	if config.CloudInit.Hostname == "" {
		config.CloudInit.Hostname = config.Name
	}

	// 使用 cloudinit 包生成 ISO
	builder := cloudinit.NewISOBuilder()
	isoPath, err := builder.BuildISO(&cloudinit.BuildOptions{
		VMName:   config.Name,
		Config:   config.CloudInit,
		UserData: config.CloudInitUserData,
	})
	if err != nil {
		return "", err
	}

	return isoPath, nil
}

// CleanupCloudInitISO 清理 cloud-init ISO 文件
func (c *Client) CleanupCloudInitISO(vmName string) error {
	builder := cloudinit.NewISOBuilder()
	return builder.CleanupISO(vmName, "")
}

// GetCloudInitISOPath 获取 cloud-init ISO 路径
func (c *Client) GetCloudInitISOPath(vmName string) string {
	builder := cloudinit.NewISOBuilder()
	return builder.GetISOPath(vmName, "")
}

// cleanupCloudInitISOOnError 错误时清理 cloud-init ISO
func (c *Client) cleanupCloudInitISOOnError(cloudInitISOPath string) {
	if cloudInitISOPath != "" {
		_ = os.Remove(cloudInitISOPath)
	}
}
