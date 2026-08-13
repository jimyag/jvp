package service

import (
	"strings"
	"testing"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/cloudinit"
	"gopkg.in/yaml.v3"
)

func TestConvertWindowsUserDataToCloudInit(t *testing.T) {
	service := &InstanceService{}
	config, userData, err := service.convertWindowsUserDataToCloudInit("win-instance", &entity.UserDataConfig{
		StructuredUserData: &entity.StructuredUserData{
			Hostname: "win-client",
			Timezone: "Asia/Shanghai",
			Users: []entity.User{{
				Name:            "operator",
				Groups:          "Administrators",
				PlainTextPasswd: "StrongPassw0rd!",
				Sudo:            "ALL=(ALL) NOPASSWD:ALL",
				Shell:           "/bin/bash",
			}},
			RunCmd: []string{`powershell.exe -NoProfile -Command "Set-Content C:\ready.txt ready"`},
		},
	})
	if err != nil {
		t.Fatalf("convert Windows user data: %v", err)
	}
	if config.Hostname != "win-client" {
		t.Fatalf("hostname = %q, want win-client", config.Hostname)
	}

	content, err := cloudinit.NewGenerator().GenerateUserDataFromStruct(userData)
	if err != nil {
		t.Fatalf("generate Windows user data: %v", err)
	}
	for _, expected := range []string{
		"#cloud-config",
		"set_timezone: Asia/Shanghai",
		"name: operator",
		"groups: Administrators",
		"passwd: StrongPassw0rd!",
		"runcmd:",
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("generated user data does not contain %q:\n%s", expected, content)
		}
	}
	for _, unexpected := range []string{"sudo:", "shell:", "plain_text_passwd:"} {
		if strings.Contains(content, unexpected) {
			t.Errorf("generated Windows user data contains Linux field %q:\n%s", unexpected, content)
		}
	}
}

func TestConvertWindowsRawUserDataPreservesPowerShell(t *testing.T) {
	const script = "#ps1_sysnative\nSet-Content C:\\ready.txt ready"
	service := &InstanceService{}
	config, userData, err := service.convertWindowsUserDataToCloudInit("win-instance", &entity.UserDataConfig{
		RawUserData: script,
	})
	if err != nil {
		t.Fatalf("convert raw Windows user data: %v", err)
	}
	if userData != nil {
		t.Fatalf("userData = %#v, want nil for raw script", userData)
	}
	if config.CustomUserData != script {
		t.Fatalf("custom user data = %q, want %q", config.CustomUserData, script)
	}
}

func TestGenerateMetaDataWithPublicKeys(t *testing.T) {
	content, err := cloudinit.NewGenerator().GenerateMetaDataWithPublicKeys("win-client", []string{"ssh-ed25519 AAAA test"})
	if err != nil {
		t.Fatalf("generate meta-data: %v", err)
	}
	var metadata cloudinit.MetaData
	if err := yaml.Unmarshal([]byte(content), &metadata); err != nil {
		t.Fatalf("parse meta-data: %v", err)
	}
	if metadata.LocalHostname != "win-client" {
		t.Fatalf("local hostname = %q, want win-client", metadata.LocalHostname)
	}
	if len(metadata.PublicKeys) != 1 || metadata.PublicKeys[0] != "ssh-ed25519 AAAA test" {
		t.Fatalf("public keys = %#v", metadata.PublicKeys)
	}
}

func TestIsWindowsTemplate(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		template *entity.Template
		want     bool
	}{
		{name: "OS metadata", template: &entity.Template{OS: entity.TemplateOS{Name: "Windows 11"}}, want: true},
		{name: "tag", template: &entity.Template{Tags: []string{"windows", "cloud-image"}}, want: true},
		{name: "short image name", template: &entity.Template{VolumeName: "win11-pro.qcow2"}, want: true},
		{name: "Linux", template: &entity.Template{OS: entity.TemplateOS{Name: "Ubuntu"}}, want: false},
		{name: "nil", template: nil, want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isWindowsTemplate(testCase.template); got != testCase.want {
				t.Fatalf("isWindowsTemplate() = %v, want %v", got, testCase.want)
			}
		})
	}
}
