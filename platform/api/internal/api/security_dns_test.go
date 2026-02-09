package api

import (
	"context"
	"net"
	"testing"
)

func TestValidateTargetURLs_DNSResolution(t *testing.T) {
	resolver := defaultSecurityTestResolver(t)

	t.Run("Localhost hostname", func(t *testing.T) {
		err := validateTargetURLsWithResolver(context.Background(), resolver, []string{"http://localhost"})
		if err == nil {
			t.Error("Expected localhost to be blocked via DNS resolution")
		}
	})

	t.Run("Loopback hostname", func(t *testing.T) {
		err := validateTargetURLsWithResolver(context.Background(), resolver, []string{"http://ip6-localhost"})
		if err == nil {
			t.Error("Expected ip6-localhost to be blocked via DNS resolution")
		}
	})
}

func TestIsDisallowedIP_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		shouldBlock bool
	}{
		// Unspecified / reserved
		{"Unspecified IPv4", "0.0.0.0", true},

		// Loopback
		{"IPv4 loopback 127.0.0.1", "127.0.0.1", true},
		{"IPv4 loopback 127.1.1.1", "127.1.1.1", true},
		{"IPv6 loopback", "::1", true},

		// RFC1918 Private
		{"Private 10.x", "10.0.0.1", true},
		{"Private 10.x max", "10.255.255.255", true},
		{"Private 192.168.x", "192.168.1.1", true},
		{"Private 172.16.x min", "172.16.0.0", true},
		{"Private 172.31.x max", "172.31.255.255", true},
		{"Carrier-grade NAT", "100.64.0.1", true},

		// Link-local
		{"Link-local 169.254.x", "169.254.1.1", true},
		{"Metadata endpoint", "169.254.169.254", true},
		{"Benchmark network", "198.18.0.1", true},
		{"TEST-NET-1", "192.0.2.10", true},
		{"TEST-NET-2", "198.51.100.10", true},
		{"TEST-NET-3", "203.0.113.10", true},
		{"IPv4 multicast", "224.0.0.10", true},
		{"IPv4 reserved", "240.0.0.1", true},

		// IPv6 special
		{"IPv6 unspecified", "::", true},
		{"IPv6 link-local", "fe80::1", true},
		{"IPv6 unique local fc00", "fc00::1", true},
		{"IPv6 unique local fd00", "fd00::1", true},
		{"IPv6 multicast", "ff02::1", true},
		{"IPv6 docs prefix", "2001:db8::1", true},

		// Public IPs (should NOT block)
		{"Public Google DNS", "8.8.8.8", false},
		{"Public Cloudflare DNS", "1.1.1.1", false},
		{"Public example IP", "93.184.216.34", false},
		{"IPv6 public", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			result := isDisallowedIP(ip)
			if result != tt.shouldBlock {
				if tt.shouldBlock {
					t.Errorf("Expected %s to be blocked, but was allowed", tt.ip)
				} else {
					t.Errorf("Expected %s to be allowed, but was blocked", tt.ip)
				}
			}
		})
	}
}
