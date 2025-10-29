package libvirt

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Cloud-Init ISO 生成相关方法
// ============================================================================

// generateCloudInitISO 生成 cloud-init配置 ISO 镜像
// 返回生成的 ISO 文件路径
func (c *Client) generateCloudInitISO(config *CreateVMConfig) (string, error) {
	if config.CloudInit == nil {
		return "", nil
	}

	// 创建临时目录
	tmpDir, err := ioutil.TempDir("", "cloudinit-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 生成 meta-data
	metaData, err := c.generateMetaData(config)
	if err != nil {
		return "", fmt.Errorf("failed to generate meta-data: %v", err)
	}

	metaDataPath := filepath.Join(tmpDir, "meta-data")
	if err := ioutil.WriteFile(metaDataPath, []byte(metaData), 0644); err != nil {
		return "", fmt.Errorf("failed to write meta-data: %v", err)
	}

	// 生成 user-data
	userData, err := c.generateUserData(config.CloudInit)
	if err != nil {
		return "", fmt.Errorf("failed to generate user-data: %v", err)
	}

	userDataPath := filepath.Join(tmpDir, "user-data")
	if err := ioutil.WriteFile(userDataPath, []byte(userData), 0644); err != nil {
		return "", fmt.Errorf("failed to write user-data: %v", err)
	}

	// 生成 network-config（如果有网络配置）
	if config.CloudInit.Network != nil {
		networkConfig, err := c.generateNetworkConfig(config.CloudInit.Network)
		if err != nil {
			return "", fmt.Errorf("failed to generate network-config: %v", err)
		}

		networkConfigPath := filepath.Join(tmpDir, "network-config")
		if err := ioutil.WriteFile(networkConfigPath, []byte(networkConfig), 0644); err != nil {
			return "", fmt.Errorf("failed to write network-config: %v", err)
		}
	}

	// 生成 ISO 文件
	isoPath := filepath.Join("/var/lib/libvirt/images", fmt.Sprintf("%s-cidata.iso", config.Name))

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
		return "", fmt.Errorf("failed to create ISO: %v, output: %s", err, output)
	}

	return isoPath, nil
}

// generateMetaData 生成 meta-data 文件内容
func (c *Client) generateMetaData(config *CreateVMConfig) (string, error) {
	hostname := config.CloudInit.Hostname
	if hostname == "" {
		hostname = config.Name
	}

	instanceID, err := generateInstanceID()
	if err != nil {
		return "", err
	}

	metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, instanceID, hostname)

	return metaData, nil
}

// generateUserData 生成 user-data 文件内容
func (c *Client) generateUserData(config *CloudInitConfig) (string, error) {
	// 如果提供了自定义 user-data，直接使用
	if config.CustomUserData != "" {
		return config.CustomUserData, nil
	}

	// 构建 user-data 结构
	userData := map[string]interface{}{
		"#cloud-config": nil,
	}

	// 设置用户
	username := config.Username
	if username == "" {
		username = "ubuntu"
	}

	users := []interface{}{
		"default",
		map[string]interface{}{
			"name":                username,
			"groups":              "sudo",
			"shell":               "/bin/bash",
			"sudo":                []string{"ALL=(ALL) NOPASSWD:ALL"},
			"lock_passwd":         false,
			"ssh_authorized_keys": config.SSHKeys,
		},
	}

	// 设置密码（如果提供）
	if config.Password != "" {
		hashedPassword, err := hashPassword(config.Password)
		if err != nil {
			return "", fmt.Errorf("failed to hash password: %v", err)
		}
		users[1].(map[string]interface{})["passwd"] = hashedPassword
	}

	userData["users"] = users

	// 禁用 root 登录
	if config.DisableRoot {
		userData["disable_root"] = true
	}

	// 设置时区
	if config.Timezone != "" {
		userData["timezone"] = config.Timezone
	}

	// 要安装的软件包
	if len(config.Packages) > 0 {
		userData["packages"] = config.Packages
	}

	// 启动后执行的命令
	if len(config.Commands) > 0 {
		userData["runcmd"] = config.Commands
	}

	// 要写入的文件
	if len(config.WriteFiles) > 0 {
		var writeFiles []map[string]interface{}
		for _, file := range config.WriteFiles {
			owner := file.Owner
			if owner == "" {
				owner = "root:root"
			}
			permissions := file.Permissions
			if permissions == "" {
				permissions = "0644"
			}

			writeFiles = append(writeFiles, map[string]interface{}{
				"path":        file.Path,
				"content":     file.Content,
				"owner":       owner,
				"permissions": permissions,
			})
		}
		userData["write_files"] = writeFiles
	}

	// 序列化为 YAML
	yamlData, err := yaml.Marshal(userData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal user-data to YAML: %v", err)
	}

	// 添加 cloud-config header
	result := "#cloud-config\n" + string(yamlData)
	return result, nil
}

// generateNetworkConfig 生成 network-config 文件内容
func (c *Client) generateNetworkConfig(network *CloudInitNetwork) (string, error) {
	version := network.Version
	if version == "" {
		version = "2"
	}

	networkConfig := map[string]interface{}{
		"version":   version,
		"ethernets": network.Ethernets,
	}

	yamlData, err := yaml.Marshal(networkConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal network-config to YAML: %v", err)
	}

	return string(yamlData), nil
}

// hashPassword 使用 bcrypt 加密密码
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// generateInstanceID 生成随机的 instance-id
func generateInstanceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("i-%x", b), nil
}

// cleanupCloudInitISO 清理 cloud-init ISO 文件
func (c *Client) cleanupCloudInitISO(vmName string) error {
	isoPath := filepath.Join("/var/lib/libvirt/images", fmt.Sprintf("%s-cidata.iso", vmName))
	if err := os.Remove(isoPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cloud-init ISO: %v", err)
	}
	return nil
}

// GetCloudInitISOPath 获取 cloud-init ISO 路径
func (c *Client) GetCloudInitISOPath(vmName string) string {
	return filepath.Join("/var/lib/libvirt/images", fmt.Sprintf("%s-cidata.iso", vmName))
}
