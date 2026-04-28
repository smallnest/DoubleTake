package game

import (
	"net"
	"regexp"
	"testing"
)

var ipv4Pattern = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

func TestGetLocalIP(t *testing.T) {
	ip, err := GetLocalIP()
	if err != nil {
		t.Fatalf("GetLocalIP() returned error: %v", err)
	}

	if !ipv4Pattern.MatchString(ip) {
		t.Fatalf("GetLocalIP() returned invalid IPv4 format: %q", ip)
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		t.Fatalf("GetLocalIP() returned unparseable IP: %q", ip)
	}

	if parsed.To4() == nil {
		t.Fatalf("GetLocalIP() returned non-IPv4 address: %q", ip)
	}

	if parsed.IsLoopback() {
		t.Fatalf("GetLocalIP() returned loopback address: %q", ip)
	}

	// Verify not link-local (169.254.x.x)
	if parsed[0] == 169 && parsed[1] == 254 {
		t.Fatalf("GetLocalIP() returned link-local address: %q", ip)
	}
}
