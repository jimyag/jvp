package main

import (
	"log"

	libvirtclient "github.com/jimyag/jvp/pkg/libvirt"
	// Uncomment if you want to use DeleteDomain method
	// "github.com/digitalocean/go-libvirt"
)

func main() {
	// 连接到 libvirt
	client, err := libvirtclient.New()
	if err != nil {
		log.Fatalf("Failed to connect to libvirt: %v", err)
	}

	// 配置虚拟机参数
	config := &libvirtclient.CreateVMConfig{
		Name:          "test-vm",                            // 虚拟机名称
		Memory:        2 * 1024 * 1024,                      // 2GB 内存（单位：KB）
		VCPUs:         1,                                    // 1 核心 CPU
		DiskPath:      "/var/lib/libvirt/images/test.qcow2", // 磁盘路径
		NetworkType:   "bridge",                             // 网络类型：桥接
		NetworkSource: "br0",                                // 桥接网卡：br0
		// 以下参数使用默认值
		// DiskBus:       "virtio",                             // 默认值
		// OSType:        "hvm",                                // 默认值
		// Architecture:  "x86_64",                             // 默认值
		// VNCSocket:     "/var/lib/libvirt/qemu/test-vm.vnc",  // 默认值
	}

	log.Printf("Creating VM with following configuration:")
	log.Printf("  Name:          %s", config.Name)
	log.Printf("  Memory:        %d KB (%.2f GB)", config.Memory, float64(config.Memory)/1024/1024)
	log.Printf("  VCPUs:         %d", config.VCPUs)
	log.Printf("  Disk:          %s", config.DiskPath)
	log.Printf("  Network Type:  %s", config.NetworkType)
	log.Printf("  Network Source:%s", config.NetworkSource)

	// 方法 1: 定义域（不启动）
	// domain, err := client.DefineDomain(config)
	// if err != nil {
	// 	log.Fatalf("Failed to define domain: %v", err)
	// }
	// log.Printf("✓ Domain '%s' defined successfully (UUID: %x)", domain.Name, domain.UUID)
	//
	// // 手动启动域
	// err = client.StartDomain(domain)
	// if err != nil {
	// 	log.Fatalf("Failed to start domain: %v", err)
	// }
	// log.Printf("✓ Domain '%s' started successfully", domain.Name)

	// 方法 2: 定义并立即启动（推荐）
	domain, err := client.CreateDomain(config, true) // true = 立即启动
	if err != nil {
		log.Fatalf("Failed to create and start domain: %v", err)
	}

	log.Printf("✓ Domain '%s' created and started successfully", domain.Name)
	log.Printf("  UUID: %x", domain.UUID)
	log.Printf("  VNC Socket: /var/lib/libvirt/qemu/%s.vnc", config.Name)

	// 获取域的详细信息
	info, err := client.GetDomainInfo(domain.UUID)
	if err != nil {
		log.Printf("Warning: Failed to get domain info: %v", err)
	} else {
		log.Printf("\nDomain Information:")
		log.Printf("  State:         %s", info.State)
		log.Printf("  Memory:        %d KB (%.2f GB)", info.Memory, float64(info.Memory)/1024/1024)
		log.Printf("  VCPUs:         %d", info.VCPUs)
		log.Printf("  OS Type:       %s", info.OSType)
		log.Printf("  Autostart:     %t", info.Autostart)
		log.Printf("  Persistent:    %t", info.Persistent)

		if len(info.NetworkInfo) > 0 {
			log.Printf("\nNetwork Interfaces:")
			for i, iface := range info.NetworkInfo {
				log.Printf("  Interface %d:", i+1)
				log.Printf("    Type:   %s", iface.Type)
				log.Printf("    Source: %s", iface.Source)
				log.Printf("    Model:  %s", iface.Model)
				if iface.MAC != "" {
					log.Printf("    MAC:    %s", iface.MAC)
				}
			}
		}
	}

	log.Printf("\n✓ VM is ready to use!")
	log.Printf("  Connect to VNC: socat - UNIX-CONNECT:/var/lib/libvirt/qemu/%s.vnc", config.Name)

	// 可选：删除虚拟机（取消注释以启用）
	// log.Printf("\nDeleting domain...")
	// err = client.DeleteDomain(domain, libvirt.DomainUndefineManagedSave)
	// if err != nil {
	// 	log.Fatalf("Failed to delete domain: %v", err)
	// }
	// log.Printf("✓ Domain deleted successfully")
}
