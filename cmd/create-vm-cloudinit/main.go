package main

import (
	"log"

	"github.com/jimyag/jvp/pkg/cloudinit"
	libvirtclient "github.com/jimyag/jvp/pkg/libvirt"
)

func main() {
	// 连接到 libvirt
	client, err := libvirtclient.New()
	if err != nil {
		log.Fatalf("Failed to connect to libvirt: %v", err)
	}

	// 配置虚拟机参数（包含 cloud-init）
	config := &libvirtclient.CreateVMConfig{
		Name:          "cloudinit-test-vm",                  // 虚拟机名称
		Memory:        2 * 1024 * 1024,                      // 2GB 内存（单位：KB）
		VCPUs:         2,                                    // 2 核心 CPU
		DiskPath:      "/var/lib/libvirt/images/test.qcow2", // 磁盘路径（需要预先存在）
		NetworkType:   "bridge",                             // 网络类型：桥接
		NetworkSource: "br0",                                // 桥接网卡：br0

		// Cloud-Init 配置
		CloudInit: &cloudinit.Config{
			Hostname:    "jimyag",        // 主机名
			Username:    "jimyag",        // 用户名
			Password:    "jimyag",        // 密码（会自动 hash）
			DisableRoot: false,           // 禁用 root 登录
			Timezone:    "Asia/Shanghai", // 设置时区
			Packages:    []string{},      // 要安装的软件包
			Commands: []string{ // 启动后执行的命令
				"echo 'Hello from cloud-init!' > /tmp/hello.txt",
				"apt-get update",
			},
			WriteFiles: []cloudinit.File{ // 写入配置文件
				{
					Path:        "/etc/motd",
					Content:     "Welcome to jimyag's Cloud-Init VM!\n",
					Owner:       "root:root",
					Permissions: "0644",
				},
			},
		},
	}

	log.Printf("Creating VM with cloud-init configuration...")
	log.Printf("  VM Name:    %s", config.Name)
	log.Printf("  Hostname:   %s", config.CloudInit.Hostname)
	log.Printf("  Username:   %s", config.CloudInit.Username)
	log.Printf("  Memory:     %d KB (%.2f GB)", config.Memory, float64(config.Memory)/1024/1024)
	log.Printf("  VCPUs:      %d", config.VCPUs)

	// 创建并启动虚拟机
	domain, err := client.CreateDomain(config, true) // true = 立即启动
	if err != nil {
		log.Fatalf("Failed to create domain: %v", err)
	}

	log.Printf("✓ VM created and started successfully")
	log.Printf("  UUID: %x", domain.UUID)
	log.Printf("  VNC Socket: /var/lib/libvirt/qemu/%s.vnc", config.Name)
	log.Printf("\nCloud-init will run on first boot and configure the VM.")
	log.Printf("You can login with:")
	log.Printf("  Username: %s", config.CloudInit.Username)
	log.Printf("  Password: %s", config.CloudInit.Password)
	log.Printf("\nOr use SSH key authentication if configured.")

	// 获取 cloud-init ISO 路径
	cloudInitISO := client.GetCloudInitISOPath(config.Name)
	log.Printf("\nCloud-init ISO: %s", cloudInitISO)
	log.Printf("Note: You can remove this ISO after first boot completes.")

	//可选：首次启动完成后清理 cloud-init ISO
	log.Printf("\nCleaning up cloud-init ISO...")
	err = client.CleanupCloudInitISO(config.Name)
	if err != nil {
		log.Printf("Warning: Failed to cleanup cloud-init ISO: %v", err)
	}
}
