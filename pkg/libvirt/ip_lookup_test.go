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
