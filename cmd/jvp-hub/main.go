package main

import (
	"fmt"
	"log"

	_ "github.com/jimmicro/version"
	jvplibvirt "github.com/jimyag/jvp/pkg/libvirt"
)

func main() {
	c, err := jvplibvirt.New()
	if err != nil {
		log.Fatalf("failed to create libvirt client: %v", err)
	}

	// 获取所有域的摘要信息
	fmt.Println("=== All Domains ===")
	domains, err := c.GetVMSummaries()
	if err != nil {
		log.Fatalf("failed to get vm summaries: %v", err)
	}

	// 如果有域，优先选择运行中的域获取详细信息
	if len(domains) > 0 {
		for _, domain := range domains {
			domainInfo, err := c.GetDomainInfo(domain.UUID)
			if err != nil {
				log.Printf("failed to get domain info for '%s' (UUID: %x): %v", domain.Name, domain.UUID, err)
			} else {
				c.PrintDomainInfo(domainInfo)
			}
		}
	}

	fmt.Println("\ndone")
}
