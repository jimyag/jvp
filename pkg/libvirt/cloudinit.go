package libvirt

import (
	"fmt"
	"os"

	"github.com/jimyag/jvp/pkg/cloudinit"
)

// generateCloudInitISO 生成 cloud-init 配置 ISO 镜像
// 返回生成的 ISO 文件路径
// 支持 CloudInit 或 CloudInitUserData 任一配置
// 注意：如果只提供 CloudInitUserData，会创建一个最小的 CloudInit 配置
func (c *Client) generateCloudInitISO(config *CreateVMConfig) (string, error) {
	// 至少需要 CloudInit 或 CloudInitUserData 之一
	if config.CloudInit == nil && config.CloudInitUserData == nil {
		return "", fmt.Errorf("either CloudInit or CloudInitUserData must be provided")
	}

	// BuildISO 需要 Config 不为 nil，如果只提供了 CloudInitUserData，创建一个最小的 Config
	cloudInitConfig := config.CloudInit
	if cloudInitConfig == nil {
		// 创建一个最小的 CloudInit 配置，使用 VM 名称作为 hostname
		cloudInitConfig = &cloudinit.Config{
			Hostname: config.Name,
		}
	} else {
		// 如果提供了 CloudInit，设置默认 hostname（如果未设置）
		if cloudInitConfig.Hostname == "" {
			cloudInitConfig.Hostname = config.Name
		}
	}

	// 使用 cloudinit 包生成 ISO
	builder := cloudinit.NewISOBuilder()
	isoPath, err := builder.BuildISO(&cloudinit.BuildOptions{
		VMName:   config.Name,
		Config:   cloudInitConfig,
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
