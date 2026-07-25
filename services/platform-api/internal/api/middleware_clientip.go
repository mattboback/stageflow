package api

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Deriving a rate-limit key from the client address.
//
// Trusting X-Forwarded-For unconditionally would let any client forge a fresh
// rate-limit identity per request, so a forwarded address is only honored when the
// immediate peer is a configured trusted proxy.

// trustedProxies caches the parsed PLATFORM_API_TRUSTED_PROXIES list. It is
// populated lazily on first use so t.Setenv in tests can change the value
// before the rate limiter reads it. Call resetTrustedProxiesForTest to force
// a reload.
var (
	trustedProxiesOnce sync.Once
	trustedProxiesVal  []*net.IPNet
)

func trustedProxies() []*net.IPNet {
	trustedProxiesOnce.Do(func() {
		trustedProxiesVal = loadTrustedProxies()
	})

	return trustedProxiesVal
}

func resetTrustedProxiesForTest() {
	trustedProxiesOnce = sync.Once{}
	trustedProxiesVal = nil
}

func loadTrustedProxies() []*net.IPNet {
	raw := strings.TrimSpace(os.Getenv("PLATFORM_API_TRUSTED_PROXIES"))
	if raw == "" {
		// The supplied Caddy topology connects over loopback. Trusting only
		// loopback by default gives public users independent limiter buckets
		// without allowing network clients to spoof forwarding headers.
		raw = defaultTrustedProxyCIDRs
	}

	var nets []*net.IPNet

	for _, part := range strings.Split(raw, ",") {
		cidr := strings.TrimSpace(part)
		if cidr == "" {
			continue
		}

		if !strings.Contains(cidr, "/") {
			if ip := net.ParseIP(cidr); ip != nil {
				if ip.To4() != nil {
					cidr += "/32"
				} else {
					cidr += "/128"
				}
			}
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("Ignoring invalid CIDR in PLATFORM_API_TRUSTED_PROXIES", "value", cidr, "error", err)

			continue
		}

		nets = append(nets, ipNet)
	}

	return nets
}

func remoteAddrIsTrusted(remoteAddr string) bool {
	nets := trustedProxies()
	if len(nets) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil || host == "" {
		host = strings.TrimSpace(remoteAddr)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func rateLimitKey(r *http.Request) string {
	if forwardedKey := trustedForwardedRateLimitKey(r); forwardedKey != "" {
		return forwardedKey
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return unknownRateLimitKey
}

func trustedForwardedRateLimitKey(r *http.Request) string {
	if !remoteAddrIsTrusted(r.RemoteAddr) {
		return ""
	}

	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded == "" {
		return ""
	}

	parts := strings.Split(forwarded, ",")

	var leftmostValid string

	// X-Forwarded-For is appended from left to right by proxies. Walk from the
	// trusted edge toward the client and select the first untrusted address so
	// a caller-supplied leftmost value cannot mint fresh limiter buckets.
	for index := len(parts) - 1; index >= 0; index-- {
		ip := parseForwardedIP(parts[index])
		if ip == nil {
			return ""
		}

		leftmostValid = ip.String()
		if !ipIsTrustedProxy(ip) {
			return leftmostValid
		}
	}

	return leftmostValid
}

func parseForwardedIP(value string) net.IP {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}

	return net.ParseIP(strings.Trim(value, "[]"))
}

func ipIsTrustedProxy(ip net.IP) bool {
	for _, network := range trustedProxies() {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
