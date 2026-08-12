package libvirt

import (
	"reflect"
	"testing"
)

func TestParseIpNeighFindsMACAfterLladdr(t *testing.T) {
	out := []byte(`192.168.2.227 lladdr 52:54:00:7a:85:38 REACHABLE
fe80::5054:ff:fe7a:8538 dev br0 lladdr 52:54:00:7a:85:38 STALE
192.168.2.1 dev br0 lladdr 8c:de:f9:3c:cc:bf REACHABLE
`)

	got := parseIpNeigh(out, "52:54:00:7a:85:38")
	want := []string{"192.168.2.227", "fe80::5054:ff:fe7a:8538"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseIpNeigh() = %#v, want %#v", got, want)
	}
}

func TestParseArpScanOutputFindsMAC(t *testing.T) {
	out := []byte(`Interface: br0, type: EN10MB, MAC: 2c:f0:5d:02:74:30, IPv4: 192.168.2.100
Starting arp-scan 1.10.0 with 256 hosts
192.168.2.101	52:54:00:1b:29:aa	QEMU
192.168.2.227	52:54:00:7a:85:38	QEMU

3 packets received by filter, 0 packets dropped by kernel
`)

	got := parseArpScanOutput(out, "52:54:00:7a:85:38")
	want := []string{"192.168.2.227"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseArpScanOutput() = %#v, want %#v", got, want)
	}
}

func TestHasIPv4(t *testing.T) {
	if hasIPv4([]string{"fe80::5054:ff:fe7a:8538"}) {
		t.Fatal("hasIPv4() returned true for IPv6-only input")
	}
	if !hasIPv4([]string{"fe80::5054:ff:fe7a:8538", "192.168.2.227"}) {
		t.Fatal("hasIPv4() returned false for input containing IPv4")
	}
}

func TestParseInterfaceNames(t *testing.T) {
	output := `5: br0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default qlen 1000\    link/ether 2c:f0:5d:02:74:30 brd ff:ff:ff:ff:ff:ff
7: br0: inet 192.168.2.100/24 brd 192.168.2.255 scope global br0
9: virbr0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN mode DEFAULT group default qlen 1000
`

	got := parseInterfaceNames(output)
	want := []string{"br0", "virbr0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseInterfaceNames() = %#v, want %#v", got, want)
	}
}
