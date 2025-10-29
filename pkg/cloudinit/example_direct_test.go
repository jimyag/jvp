package cloudinit_test

import (
	"fmt"

	"github.com/jimyag/jvp/pkg/cloudinit"
)

// ExampleGenerator_GenerateUserDataFromStruct 展示如何直接使用 UserData 结构
func ExampleGenerator_GenerateUserDataFromStruct() {
	gen := cloudinit.NewGenerator()

	lockPasswd := false
	userData := &cloudinit.UserData{
		Groups: map[string][]string{
			"developers": {"john", "jane"},
		},
		Users: []any{
			"default",
			cloudinit.User{
				Name:       "admin",
				Groups:     "sudo",
				LockPasswd: &lockPasswd,
				SSHAuthorizedKeys: []string{
					"ssh-ed25519 AAAA...",
				},
			},
		},
		Packages: []string{"vim", "git"},
		RunCmd:   []string{"apt-get update"},
	}

	content, err := gen.GenerateUserDataFromStruct(userData)
	if err != nil {
		panic(err)
	}

	fmt.Println(content)
	// Output:
	// #cloud-config
	// groups:
	//     developers:
	//         - john
	//         - jane
	// users:
	//     - default
	//     - name: admin
	//       groups: sudo
	//       lock_passwd: false
	//       ssh_authorized_keys:
	//         - ssh-ed25519 AAAA...
	// packages:
	//     - vim
	//     - git
	// runcmd:
	//     - apt-get update
}

// ExampleGenerator_GenerateNetworkConfigFromStruct 展示如何直接使用 NetworkData 结构
func ExampleGenerator_GenerateNetworkConfigFromStruct() {
	gen := cloudinit.NewGenerator()

	eth0 := cloudinit.Ethernet{
		DHCP4:     true,
		Addresses: []string{"192.168.1.100/24"},
		Gateway4:  "192.168.1.1",
	}
	eth0.Nameservers.Addresses = []string{"8.8.8.8"}

	networkData := &cloudinit.NetworkData{
		Version: "2",
		Ethernets: map[string]cloudinit.Ethernet{
			"eth0": eth0,
		},
	}

	content, err := gen.GenerateNetworkConfigFromStruct(networkData)
	if err != nil {
		panic(err)
	}

	fmt.Println(content)
	// Output:
	// version: "2"
	// ethernets:
	//     eth0:
	//         dhcp4: true
	//         addresses:
	//             - 192.168.1.100/24
	//         gateway4: 192.168.1.1
	//         nameservers:
	//             addresses:
	//                 - 8.8.8.8
}

// ExampleUserData_advanced 展示高级特性
func ExampleUserData_advanced() {
	gen := cloudinit.NewGenerator()

	sshPwauth := true
	userData := &cloudinit.UserData{
		Users: []any{
			"default",
			cloudinit.User{Name: "admin", Groups: "sudo"},
		},
		SSHPwauth: &sshPwauth,
		ChPasswd: &cloudinit.ChPasswd{
			Expire: true,
			List:   []string{"admin:password123"},
		},
		Locale:       "en_US.UTF-8",
		FinalMessage: "Cloud-init completed!",
		PowerState: &cloudinit.PowerState{
			Mode:    "reboot",
			Message: "Rebooting after setup",
			Timeout: 30,
		},
	}

	content, _ := gen.GenerateUserDataFromStruct(userData)
	fmt.Println(content)
}

// ExampleNetworkData_bridge 展示网桥配置
func ExampleNetworkData_bridge() {
	gen := cloudinit.NewGenerator()

	br0 := cloudinit.Bridge{
		Interfaces: []string{"eth0", "eth1"},
		DHCP4:      false,
		Addresses:  []string{"10.0.0.1/24"},
		Gateway4:   "10.0.0.254",
	}
	br0.Nameservers.Addresses = []string{"8.8.8.8"}

	networkData := &cloudinit.NetworkData{
		Version: "2",
		Bridges: map[string]cloudinit.Bridge{
			"br0": br0,
		},
	}

	content, _ := gen.GenerateNetworkConfigFromStruct(networkData)
	fmt.Println(content)
}
