package cloudinit

import (
	"strings"
	"testing"
)

func TestGenerateMetaData(t *testing.T) {
	gen := NewGenerator()

	metaData, err := gen.GenerateMetaData("test-server")
	if err != nil {
		t.Fatalf("Failed to generate meta-data: %v", err)
	}

	if !strings.Contains(metaData, "instance-id:") {
		t.Error("Missing instance-id field")
	}
	if !strings.Contains(metaData, "local-hostname: test-server") {
		t.Error("Missing or incorrect local-hostname field")
	}

	t.Logf("Generated meta-data:\n%s", metaData)
}

func TestGenerateUserDataWithNewConfig(t *testing.T) {
	gen := NewGenerator()
	lockPasswd := false

	config := &Config{
		Hostname: "production-server",
		Groups: []Group{
			{
				Name:    "developers",
				Members: []string{"john"},
			},
		},
		Users: []User{
			{
				Name:              "admin",
				Gecos:             "Administrator",
				Groups:            "sudo",
				Shell:             "/bin/bash",
				Sudo:              "ALL=(ALL) NOPASSWD:ALL",
				PlainTextPasswd:   "admin123",
				LockPasswd:        &lockPasswd,
				SSHAuthorizedKeys: []string{"ssh-ed25519 AAAA..."},
			},
		},
		Timezone: "Asia/Shanghai",
		Packages: []string{"docker.io", "git"},
		Commands: []string{"echo 'test' > /tmp/test.txt"},
	}

	userData, err := gen.GenerateUserData(config)
	if err != nil {
		t.Fatalf("Failed to generate user-data: %v", err)
	}

	// 验证 cloud-config header
	if !strings.HasPrefix(userData, "#cloud-config\n") {
		t.Error("Missing #cloud-config header")
	}

	// 验证关键配置是否存在
	requiredFields := []string{
		"groups:",
		"developers:",
		"users:",
		"- default",
		"name: admin",
		"timezone: Asia/Shanghai",
		"packages:",
		"- docker.io",
		"- git",
		"runcmd:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(userData, field) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("Generated user-data:\n%s", userData)
}

func TestGenerateUserDataBackwardCompatibility(t *testing.T) {
	gen := NewGenerator()

	// 使用旧的配置方式
	config := &Config{
		Hostname: "old-server",
		Username: "olduser",
		Password: "oldpass",
		SSHKeys:  []string{"ssh-ed25519 AAAA..."},
		Timezone: "UTC",
	}

	userData, err := gen.GenerateUserData(config)
	if err != nil {
		t.Fatalf("Failed to generate user-data: %v", err)
	}

	// 验证向后兼容
	if !strings.Contains(userData, "name: olduser") {
		t.Error("Old username not found")
	}
	if !strings.Contains(userData, "timezone: UTC") {
		t.Error("Timezone not found")
	}

	t.Logf("Generated user-data (backward compatibility):\n%s", userData)
}

func TestGenerateNetworkConfig(t *testing.T) {
	gen := NewGenerator()

	network := &Network{
		Version: "2",
		Ethernets: map[string]Ethernet{
			"eth0": {
				DHCP4:     true,
				Addresses: []string{"192.168.1.100/24"},
				Gateway4:  "192.168.1.1",
			},
		},
	}

	// 设置 nameservers
	eth0 := network.Ethernets["eth0"]
	eth0.Nameservers.Addresses = []string{"8.8.8.8", "8.8.4.4"}
	network.Ethernets["eth0"] = eth0

	networkConfig, err := gen.GenerateNetworkConfig(network)
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
	}

	for _, field := range requiredFields {
		if !strings.Contains(networkConfig, field) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("Generated network-config:\n%s", networkConfig)
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("Invalid bcrypt hash format: %s", hash)
	}

	t.Logf("Generated password hash: %s", hash)
}

func TestCustomUserData(t *testing.T) {
	gen := NewGenerator()

	customData := `#cloud-config
users:
  - name: custom
    gecos: Custom User
packages:
  - nginx
`

	config := &Config{
		CustomUserData: customData,
		Hostname:       "ignored",
		Packages:       []string{"ignored"},
	}

	userData, err := gen.GenerateUserData(config)
	if err != nil {
		t.Fatalf("Failed to generate custom user-data: %v", err)
	}

	if userData != customData {
		t.Errorf("Custom user-data was not used as-is")
	}

	t.Logf("Generated custom user-data:\n%s", userData)
}
