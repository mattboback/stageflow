package api

import (
	"context"
	"errors"
	"net"
	"testing"
)

type staticResolver struct {
	records map[string][]net.IPAddr
}

func (r *staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	addrs, ok := r.records[host]
	if !ok {
		return nil, errors.New("host not found")
	}

	return addrs, nil
}

func newStaticResolver(t *testing.T, records map[string][]string) *staticResolver {
	t.Helper()

	parsed := make(map[string][]net.IPAddr, len(records))

	for host, ips := range records {
		addrs := make([]net.IPAddr, 0, len(ips))
		for _, ip := range ips {
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				t.Fatalf("failed to parse IP %q for host %q", ip, host)
			}

			addrs = append(addrs, net.IPAddr{IP: parsedIP})
		}

		parsed[host] = addrs
	}

	return &staticResolver{records: parsed}
}

func defaultSecurityTestResolver(t *testing.T) *staticResolver {
	t.Helper()

	return newStaticResolver(t, map[string][]string{
		"example.com":     {"93.184.216.34"},
		"test.com":        {"93.184.216.34"},
		"localhost":       {"127.0.0.1"},
		"ip6-localhost":   {"::1"},
		"ip6-loopback":    {"::1"},
		"metadata.google": {"169.254.169.254"},
	})
}
