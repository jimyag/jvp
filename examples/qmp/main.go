package main

import (
	"fmt"
	"log"
	"time"

	"github.com/digitalocean/go-qemu/qemu"
	"github.com/digitalocean/go-qemu/qmp"
	_ "github.com/jimmicro/version"
)

func main() {
	m, err := qmp.NewSocketMonitor("127.0.0.1", "4444", 10*time.Second)
	if err != nil {
		log.Fatalf("failed to create qmp monitor: %v", err)
	}

	d, err := qemu.NewDomain(m, "test")
	if err != nil {
		log.Fatalf("failed to create qemu domain: %v", err)
	}
	ch, _, err := d.Events()
	if err != nil {
		log.Fatalf("failed to create qemu domain: %v", err)
	}
	for event := range ch {
		fmt.Println(event.Data)
	}
}
