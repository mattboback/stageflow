package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type targetHost struct {
	raw  string
	host string
}

type ipAddrResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// validateTargetURLs enforces basic SSRF protections for URL-based jobs.
// It rejects URLs with non-HTTP schemes and targets in private/loopback/metadata ranges.
func validateTargetURLs(ctx context.Context, urls []string) error {
	return validateTargetURLsWithResolver(ctx, net.DefaultResolver, urls)
}

func validateTargetURLsWithResolver(ctx context.Context, resolver ipAddrResolver, urls []string) error {
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	for _, raw := range urls {
		target, parseErr := parseTargetURL(raw)
		if parseErr != nil {
			return parseErr
		}

		if hostErr := validateHost(ctx, resolver, target); hostErr != nil {
			return hostErr
		}
	}

	return nil
}

func parseTargetURL(raw string) (*targetHost, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("URL cannot be empty")
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s", trimmed)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme for %s (only http/https allowed)", trimmed)
	}

	host := u.Host
	if host == "" {
		return nil, fmt.Errorf("URL missing host: %s", trimmed)
	}

	if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		host = h
	}

	host = strings.Trim(host, "[]")

	return &targetHost{
		raw:  trimmed,
		host: host,
	}, nil
}

func validateHost(ctx context.Context, resolver ipAddrResolver, target *targetHost) error {
	if ip := net.ParseIP(target.host); ip != nil {
		return ensureAllowedIP(target.raw, target.host, ip)
	}

	return resolveAndValidate(ctx, resolver, target)
}

func resolveAndValidate(ctx context.Context, resolver ipAddrResolver, target *targetHost) error {
	ips, err := resolver.LookupIPAddr(ctx, target.host)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", target.host, err)
	}

	for _, resolved := range ips {
		if isDisallowedIP(resolved.IP) {
			return fmt.Errorf("hostname %s resolves to disallowed address %s", target.host, resolved.IP.String())
		}
	}

	return nil
}

func ensureAllowedIP(raw, host string, ip net.IP) error {
	if isDisallowedIP(ip) {
		return fmt.Errorf("URL host %s for %s resolves to a disallowed address", host, raw)
	}

	return nil
}

// isDisallowedIP returns true for IPs in private, loopback, link-local, or metadata ranges.
func isDisallowedIP(ip net.IP) bool {
	privateCIDRs := []string{
		"127.0.0.0/8",    // loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // link-local
	}

	for _, cidr := range privateCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		if network.Contains(ip) {
			return true
		}
	}

	// Common cloud metadata IPv4
	metadataIPs := []string{
		"169.254.169.254",
	}

	for _, m := range metadataIPs {
		if ip.Equal(net.ParseIP(m)) {
			return true
		}
	}

	// Basic IPv6 local/unique-local checks
	if ip.To4() == nil {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
		// Unique local addresses fc00::/7
		_, ula, err := net.ParseCIDR("fc00::/7")
		if err == nil && ula.Contains(ip) {
			return true
		}
	}

	return false
}
