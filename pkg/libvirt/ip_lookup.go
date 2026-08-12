package libvirt

import (
	"bytes"
	"context"
	"net/netip"
	"os/exec"
	"strings"
	"time"
)

// ResolveIPsByMAC 从 DHCP 租约和 ARP/neigh 解析给定 MAC 的 IP 列表
func ResolveIPsByMAC(client LibvirtClient, mac string) ([]string, error) {
	return ResolveIPsByMACOnInterface(client, mac, "")
}

// ResolveIPsByMACOnInterface 从 DHCP 租约和 ARP/neigh 解析给定 MAC 的 IP 列表。
// preferredInterface 用于将主动 ARP 扫描限制在 VM 所在 bridge，避免扫描无关网络。
func ResolveIPsByMACOnInterface(client LibvirtClient, mac string, preferredInterface string) ([]string, error) {
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
	arpIPs := lookupARPByMAC(client, mac, preferredInterface)
	for _, ip := range arpIPs {
		ipSet[ip] = struct{}{}
	}

	ips := make([]string, 0, len(ipSet))
	for ip := range ipSet {
		ips = append(ips, ip)
	}
	return ips, nil
}

func lookupARPByMAC(client LibvirtClient, mac string, preferredInterface string) []string {
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
		ips := parseIpNeigh(out, mac)
		if hasIPv4(ips) {
			return ips
		}
		ips = append(ips, scanLocalNetworksByARP(mac, preferredInterface)...)
		return ips
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
		matched := false
		for i, field := range fields {
			if field == "lladdr" && i+1 < len(fields) && strings.ToLower(fields[i+1]) == mac {
				matched = true
				break
			}
		}
		if !matched && len(fields) >= 5 && strings.ToLower(fields[4]) == mac {
			matched = true
		}
		if matched {
			ips = append(ips, fields[0])
		}
	}
	return ips
}

func scanLocalNetworksByARP(mac string, preferredInterface string) []string {
	interfaces := preferredInterfaces(preferredInterface)
	if len(interfaces) == 0 {
		interfaces = listBridgeInterfaces()
	}
	if len(interfaces) == 0 {
		interfaces = listIPv4Interfaces()
	}
	var ips []string

	for _, iface := range interfaces {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		scanOut, scanErr := exec.CommandContext(ctx, "arp-scan", "--interface", iface, "--localnet").Output()
		cancel()
		if scanErr != nil {
			continue
		}
		ips = append(ips, parseArpScanOutput(scanOut, mac)...)
		if hasIPv4(ips) {
			return ips
		}
	}
	return ips
}

func preferredInterfaces(preferredInterface string) []string {
	if preferredInterface == "" {
		return nil
	}
	return []string{preferredInterface}
}

func listBridgeInterfaces() []string {
	out, err := exec.Command("ip", "-o", "link", "show", "type", "bridge").Output()
	if err != nil {
		return nil
	}
	return parseInterfaceNames(string(out))
}

func listIPv4Interfaces() []string {
	out, err := exec.Command("ip", "-o", "-4", "addr", "show", "scope", "global").Output()
	if err != nil {
		return nil
	}
	return parseInterfaceNames(string(out))
}

func parseInterfaceNames(output string) []string {
	seen := make(map[string]struct{})
	var interfaces []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		iface := strings.TrimSuffix(fields[1], ":")
		if iface == "" {
			continue
		}
		if _, ok := seen[iface]; ok {
			continue
		}
		seen[iface] = struct{}{}
		interfaces = append(interfaces, iface)
	}
	return interfaces
}

func parseArpScanOutput(out []byte, mac string) []string {
	lines := bytes.Split(out, []byte("\n"))
	ips := []string{}
	for _, line := range lines {
		fields := strings.Fields(string(line))
		if len(fields) >= 2 && strings.ToLower(fields[1]) == mac {
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

func hasIPv4(ips []string) bool {
	for _, ip := range ips {
		addr, err := netip.ParseAddr(ip)
		if err == nil && addr.Is4() {
			return true
		}
	}
	return false
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
