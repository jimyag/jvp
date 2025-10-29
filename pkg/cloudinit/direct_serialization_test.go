package cloudinit

import (
	"strings"
	"testing"
)

// TestGenerateUserDataFromStruct 测试直接从 UserData 结构生成配置
func TestGenerateUserDataFromStruct(t *testing.T) {
	gen := NewGenerator()

	lockPasswd := false
	userData := &UserData{
		Groups: map[string][]string{
			"developers": {"john", "jane"},
			"admins":     {"root"},
		},
		Users: []any{
			"default",
			User{
				Name:       "admin",
				Gecos:      "System Administrator",
				Groups:     "sudo,admins",
				Shell:      "/bin/bash",
				Sudo:       "ALL=(ALL) NOPASSWD:ALL",
				LockPasswd: &lockPasswd,
				SSHAuthorizedKeys: []string{
					"ssh-ed25519 AAAA...",
				},
			},
			User{
				Name:   "developer",
				Gecos:  "Developer User",
				Groups: "developers",
				SSHImportID: []string{
					"gh:johndoe",
				},
			},
		},
		DisableRoot: true,
		Timezone:    "Asia/Shanghai",
		Packages:    []string{"docker.io", "git", "vim"},
		RunCmd: []string{
			"systemctl enable docker",
			"systemctl start docker",
		},
		WriteFiles: []WriteFile{
			{
				Path:        "/etc/motd",
				Content:     "Welcome to Production Server!",
				Owner:       "root:root",
				Permissions: "0644",
			},
		},
	}

	content, err := gen.GenerateUserDataFromStruct(userData)
	if err != nil {
		t.Fatalf("Failed to generate user-data: %v", err)
	}

	// 验证必需字段
	requiredFields := []string{
		"#cloud-config",
		"groups:",
		"developers:",
		"- john",
		"- jane",
		"users:",
		"- default",
		"name: admin",
		"gecos: System Administrator",
		"name: developer",
		"ssh_import_id:",
		"- gh:johndoe",
		"disable_root: true",
		"timezone: Asia/Shanghai",
		"packages:",
		"- docker.io",
		"runcmd:",
		"- systemctl enable docker",
		"write_files:",
		"path: /etc/motd",
	}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("Generated user-data:\n%s", content)
}

// TestGenerateUserDataWithAdvancedFeatures 测试高级特性
func TestGenerateUserDataWithAdvancedFeatures(t *testing.T) {
	gen := NewGenerator()

	sshPwauth := true
	userData := &UserData{
		Users: []any{
			"default",
			User{Name: "admin", Groups: "sudo"},
		},
		SSHPwauth: &sshPwauth,
		ChPasswd: &ChPasswd{
			Expire: true,
			List: []string{
				"admin:password123",
			},
		},
		Locale: "en_US.UTF-8",
		Bootcmd: []string{
			"echo 'Boot command executed'",
		},
		FinalMessage: "Cloud-init completed successfully!",
		PowerState: &PowerState{
			Mode:    "reboot",
			Message: "Rebooting after cloud-init",
			Timeout: 30,
		},
	}

	content, err := gen.GenerateUserDataFromStruct(userData)
	if err != nil {
		t.Fatalf("Failed to generate user-data: %v", err)
	}

	// 验证高级字段
	advancedFields := []string{
		"ssh_pwauth: true",
		"chpasswd:",
		"expire: true",
		"locale: en_US.UTF-8",
		"bootcmd:",
		"final_message:",
		"power_state:",
		"mode: reboot",
	}

	for _, field := range advancedFields {
		if !strings.Contains(content, field) {
			t.Errorf("Missing advanced field: %s", field)
		}
	}

	t.Logf("Generated user-data with advanced features:\n%s", content)
}

// TestGenerateNetworkConfigFromStruct 测试直接从 NetworkData 结构生成网络配置
func TestGenerateNetworkConfigFromStruct(t *testing.T) {
	gen := NewGenerator()

	eth0 := Ethernet{
		DHCP4:     true,
		Addresses: []string{"192.168.1.100/24"},
		Gateway4:  "192.168.1.1",
	}
	eth0.Nameservers.Addresses = []string{"8.8.8.8", "8.8.4.4"}

	networkData := &NetworkData{
		Version: "2",
		Ethernets: map[string]Ethernet{
			"eth0": eth0,
		},
	}

	content, err := gen.GenerateNetworkConfigFromStruct(networkData)
	if err != nil {
		t.Fatalf("Failed to generate network-config: %v", err)
	}

	requiredFields := []string{
		"version: \"2\"",
		"ethernets:",
		"eth0:",
		"dhcp4: true",
		"addresses:",
		"- 192.168.1.100/24",
		"gateway4: 192.168.1.1",
		"nameservers:",
		"- 8.8.8.8",
	}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("Generated network-config:\n%s", content)
}

// TestGenerateNetworkConfigWithBridge 测试网桥配置
func TestGenerateNetworkConfigWithBridge(t *testing.T) {
	gen := NewGenerator()

	br0 := Bridge{
		Interfaces: []string{"eth0", "eth1"},
		DHCP4:      false,
		Addresses:  []string{"10.0.0.1/24"},
		Gateway4:   "10.0.0.254",
	}
	br0.Nameservers.Addresses = []string{"8.8.8.8"}

	networkData := &NetworkData{
		Version: "2",
		Bridges: map[string]Bridge{
			"br0": br0,
		},
	}

	content, err := gen.GenerateNetworkConfigFromStruct(networkData)
	if err != nil {
		t.Fatalf("Failed to generate network-config: %v", err)
	}

	requiredFields := []string{
		"version: \"2\"",
		"bridges:",
		"br0:",
		"interfaces:",
		"- eth0",
		"- eth1",
		"addresses:",
		"- 10.0.0.1/24",
	}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("Generated network-config with bridge:\n%s", content)
}

// TestMinimalUserData 测试最小化配置
func TestMinimalUserData(t *testing.T) {
	gen := NewGenerator()

	userData := &UserData{
		Users: []any{"default"},
	}

	content, err := gen.GenerateUserDataFromStruct(userData)
	if err != nil {
		t.Fatalf("Failed to generate minimal user-data: %v", err)
	}

	if !strings.Contains(content, "#cloud-config") {
		t.Error("Missing cloud-config header")
	}

	if !strings.Contains(content, "users:") {
		t.Error("Missing users field")
	}

	if !strings.Contains(content, "- default") {
		t.Error("Missing default user")
	}

	t.Logf("Generated minimal user-data:\n%s", content)
}

// TestWriteFileWithEncoding 测试文件写入的编码选项
func TestWriteFileWithEncoding(t *testing.T) {
	gen := NewGenerator()

	userData := &UserData{
		Users: []any{"default"},
		WriteFiles: []WriteFile{
			{
				Path:        "/tmp/test.txt",
				Content:     "Hello World",
				Encoding:    "base64",
				Owner:       "root:root",
				Permissions: "0644",
				Append:      false,
				Defer:       true,
			},
		},
	}

	content, err := gen.GenerateUserDataFromStruct(userData)
	if err != nil {
		t.Fatalf("Failed to generate user-data: %v", err)
	}

	requiredFields := []string{
		"write_files:",
		"path: /tmp/test.txt",
		"encoding: base64",
		"defer: true",
	}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("Generated user-data with file encoding:\n%s", content)
}
