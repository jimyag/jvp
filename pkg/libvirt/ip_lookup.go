package libvirt

import (
	"bytes"
	"os/exec"
	"strings"
)

// ResolveIPsByMAC 从 DHCP 租约和 ARP/neigh 解析给定 MAC 的 IP 列表
func ResolveIPsByMAC(client LibvirtClient, mac string) ([]string, error) {
	if mac == "" {
		return nil, nil
	}
	mac = strings.ToLower(mac)
	ipSet := make(map[string]struct{})

	// 1) 尝试读取 libvirt network DHCP leases
	networks, _ := client.ListNetworks()
	for _, net := range networks {
		leases, err := client.ListNetworkDHCPLeases(net)
		if err != nil {
			continue
		}
		for _, l := range leases {
			for _, m := range l.MACs {
				if strings.ToLower(m) == mac && l.IP != "" {
					ipSet[l.IP] = struct{}{}
				}
			}
		}
	}

	// 2) ARP/neigh 表
	arpIPs := lookupARPByMAC(client, mac)
	for _, ip := range arpIPs {
		ipSet[ip] = struct{}{}
	}

	ips := make([]string, 0, len(ipSet))
	for ip := range ipSet {
		ips = append(ips, ip)
	}
	return ips, nil
}

func lookupARPByMAC(client LibvirtClient, mac string) []string {
	if client.IsRemoteConnection() {
		if data, err := client.ReadRemoteFile("/proc/net/arp"); err == nil {
			return parseProcNetARP(data, mac)
		}
		// fallback: try ip neigh output via remote command; ignore errors
		_ = client.ExecuteRemoteCommand("ip neigh > /tmp/.jvp_ipneigh && cat /tmp/.jvp_ipneigh")
		if data, err2 := client.ReadRemoteFile("/tmp/.jvp_ipneigh"); err2 == nil {
			return parseIpNeigh(data, mac)
		}
		return nil
	}

	if out, err := exec.Command("ip", "neigh").Output(); err == nil {
		return parseIpNeigh(out, mac)
	}
	if out, err := exec.Command("arp", "-an").Output(); err == nil {
		return parseArpOutput(out, mac)
	}
	return nil
}

func parseIpNeigh(out []byte, mac string) []string {
	lines := bytes.Split(out, []byte("\n"))
	ips := []string{}
	for _, line := range lines {
		fields := strings.Fields(string(line))
		if len(fields) >= 5 && strings.ToLower(fields[4]) == mac {
			ips = append(ips, fields[0])
		}
	}
	return ips
}

func parseArpOutput(out []byte, mac string) []string {
	lines := bytes.Split(out, []byte("\n"))
	ips := []string{}
	for _, line := range lines {
		parts := strings.Fields(string(line))
		if len(parts) >= 4 {
			ip := strings.Trim(parts[1], "()")
			m := strings.ToLower(parts[3])
			if m == mac {
				ips = append(ips, ip)
			}
		}
	}
	return ips
}

func parseProcNetARP(out []byte, mac string) []string {
	lines := bytes.Split(out, []byte("\n"))
	ips := []string{}
	for i, line := range lines {
		if i == 0 {
			continue // header
		}
		fields := strings.Fields(string(line))
		if len(fields) >= 4 {
			m := strings.ToLower(fields[3])
			if m == mac {
				ips = append(ips, fields[0])
			}
		}
	}
	return ips
}
