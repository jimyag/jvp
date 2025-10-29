package cloudinit

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// Generator cloud-init 配置生成器
type Generator struct{}

// NewGenerator 创建新的 cloud-init 生成器
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateMetaData 生成 meta-data 文件内容
func (g *Generator) GenerateMetaData(hostname string) (string, error) {
	if hostname == "" {
		hostname = "localhost"
	}

	instanceID, err := generateInstanceID()
	if err != nil {
		return "", err
	}

	metaData := &MetaData{
		InstanceID:    instanceID,
		LocalHostname: hostname,
	}

	yamlData, err := yaml.Marshal(metaData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal meta-data to YAML: %v", err)
	}

	return string(yamlData), nil
}

// GenerateUserDataFromStruct 直接从 UserData 结构生成 user-data 文件内容
// 这个方法提供最大的灵活性，允许用户完全控制输出
//
// 示例：
//
//	userData := &cloudinit.UserData{
//	    Groups: map[string][]string{"developers": {"john"}},
//	    Users: []any{"default", cloudinit.User{Name: "admin"}},
//	}
//	content, _ := gen.GenerateUserDataFromStruct(userData)
func (g *Generator) GenerateUserDataFromStruct(userData *UserData) (string, error) {
	if userData == nil {
		return "", fmt.Errorf("userData is required")
	}

	yamlData, err := yaml.Marshal(userData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal user-data to YAML: %v", err)
	}

	// 添加 cloud-config header
	result := "#cloud-config\n" + string(yamlData)
	return result, nil
}

// GenerateNetworkConfigFromStruct 直接从 NetworkData 结构生成 network-config 文件内容
// 这个方法提供最大的灵活性，允许用户完全控制输出
//
// 示例：
//
//	networkData := &cloudinit.NetworkData{
//	    Version: "2",
//	    Ethernets: map[string]cloudinit.Ethernet{
//	        "eth0": {DHCP4: true},
//	    },
//	}
//	content, _ := gen.GenerateNetworkConfigFromStruct(networkData)
func (g *Generator) GenerateNetworkConfigFromStruct(networkData *NetworkData) (string, error) {
	if networkData == nil {
		return "", fmt.Errorf("networkData is required")
	}

	yamlData, err := yaml.Marshal(networkData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal network-config to YAML: %v", err)
	}

	return string(yamlData), nil
}

// GenerateUserData 生成 user-data 文件内容
// 从高级的 Config 结构生成，自动处理密码哈希等
func (g *Generator) GenerateUserData(config *Config) (string, error) {
	// 如果提供了自定义 user-data，直接使用
	if config.CustomUserData != "" {
		return config.CustomUserData, nil
	}

	// 构建标准的 cloud-init user-data 结构
	userData := &UserData{}

	// 处理组配置
	if len(config.Groups) > 0 {
		userData.Groups = make(map[string][]string)
		for _, group := range config.Groups {
			userData.Groups[group.Name] = group.Members
		}
	}

	// 处理用户配置
	var users []any

	// 如果有新的 Users 配置，使用新配置
	if len(config.Users) > 0 {
		// 添加 default 用户引用
		users = append(users, "default")

		// 添加自定义用户
		for _, user := range config.Users {
			// 处理密码哈希（如果提供了明文密码）
			if user.PlainTextPasswd != "" && user.Passwd == "" && user.HashedPasswd == "" {
				hashedPassword, err := HashPassword(user.PlainTextPasswd)
				if err != nil {
					return "", fmt.Errorf("failed to hash password for user %s: %v", user.Name, err)
				}
				user.Passwd = hashedPassword
			}

			users = append(users, user)
		}
	} else if config.Username != "" || config.Password != "" || len(config.SSHKeys) > 0 {
		// 向后兼容：使用旧的 Username/Password/SSHKeys 配置
		username := config.Username
		if username == "" {
			username = "ubuntu"
		}

		lockPasswd := false
		userConfig := User{
			Name:       username,
			Groups:     "sudo",
			Shell:      "/bin/bash",
			Sudo:       []string{"ALL=(ALL) NOPASSWD:ALL"},
			LockPasswd: &lockPasswd,
		}

		// 设置密码（如果提供）
		if config.Password != "" {
			hashedPassword, err := HashPassword(config.Password)
			if err != nil {
				return "", fmt.Errorf("failed to hash password: %v", err)
			}
			userConfig.Passwd = hashedPassword
		}

		// 设置 SSH 密钥
		if len(config.SSHKeys) > 0 {
			userConfig.SSHAuthorizedKeys = config.SSHKeys
		}

		users = []any{"default", userConfig}
	} else {
		// 默认只使用 default 用户
		users = []any{"default"}
	}

	userData.Users = users

	// 禁用 root 登录
	userData.DisableRoot = config.DisableRoot

	// 设置时区
	userData.Timezone = config.Timezone

	// 要安装的软件包
	userData.Packages = config.Packages

	// 启动后执行的命令
	userData.RunCmd = config.Commands

	// 要写入的文件
	if len(config.WriteFiles) > 0 {
		for _, file := range config.WriteFiles {
			owner := file.Owner
			if owner == "" {
				owner = "root:root"
			}
			permissions := file.Permissions
			if permissions == "" {
				permissions = "0644"
			}

			userData.WriteFiles = append(userData.WriteFiles, WriteFile{
				Path:        file.Path,
				Content:     file.Content,
				Owner:       owner,
				Permissions: permissions,
			})
		}
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

// GenerateNetworkConfig 生成 network-config 文件内容
func (g *Generator) GenerateNetworkConfig(network *Network) (string, error) {
	if network == nil {
		return "", nil
	}

	version := network.Version
	if version == "" {
		version = "2"
	}

	networkConfig := &NetworkData{
		Version:   version,
		Ethernets: network.Ethernets,
	}

	yamlData, err := yaml.Marshal(networkConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal network-config to YAML: %v", err)
	}

	return string(yamlData), nil
}

// HashPassword 使用 bcrypt 加密密码
func HashPassword(password string) (string, error) {
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
