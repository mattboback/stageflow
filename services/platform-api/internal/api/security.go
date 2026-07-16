package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"sync"
)

type targetHost struct {
	host string
}

type ipAddrResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type targetValidationMode uint8

const (
	targetValidationModePublic targetValidationMode = iota
	targetValidationModePrivate
)

type targetIPDecision uint8

const (
	targetIPDecisionAllow targetIPDecision = iota
	targetIPDecisionAllowInPrivateMode
	targetIPDecisionBlock
)

// validateTargetURLs enforces basic SSRF protections for URL-based jobs.
// It rejects URLs with non-HTTP schemes and targets in private/loopback/metadata ranges.
func validateTargetURLs(ctx context.Context, urls []string) error {
	return validateTargetURLsWithResolver(ctx, net.DefaultResolver, urls, targetValidationModePublic)
}

func validateTargetURLsWithResolver(
	ctx context.Context,
	resolver ipAddrResolver,
	urls []string,
	mode targetValidationMode,
) error {
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	for _, raw := range urls {
		target, parseErr := parseTargetURL(raw)
		if parseErr != nil {
			return parseErr
		}

		if hostErr := validateHost(ctx, resolver, target, mode); hostErr != nil {
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
		return nil, errors.New("invalid URL")
	}

	if u.User != nil {
		// Embedded credentials are persisted as part of the submitted URL and can
		// surface in browser errors. Require the dedicated auth recipe instead and
		// never echo the rejected credential-bearing URL in the response.
		return nil, errors.New("URL must not include embedded credentials")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("unsupported URL scheme (only http/https allowed)")
	}

	host := u.Host
	if host == "" {
		return nil, errors.New("URL missing host")
	}

	if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		host = h
	}

	host = strings.Trim(host, "[]")

	return &targetHost{host: host}, nil
}

func validateHost(ctx context.Context, resolver ipAddrResolver, target *targetHost, mode targetValidationMode) error {
	if ip := net.ParseIP(target.host); ip != nil {
		return ensureAllowedIP(target.host, ip, mode)
	}

	return resolveAndValidate(ctx, resolver, target, mode)
}

func resolveAndValidate(
	ctx context.Context,
	resolver ipAddrResolver,
	target *targetHost,
	mode targetValidationMode,
) error {
	ips, err := resolver.LookupIPAddr(ctx, target.host)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", target.host, err)
	}

	for _, resolved := range ips {
		if !isAllowedTargetIP(resolved.IP, mode) {
			return fmt.Errorf("hostname %s resolves to disallowed address %s", target.host, resolved.IP.String())
		}
	}

	return nil
}

func ensureAllowedIP(host string, ip net.IP, mode targetValidationMode) error {
	if !isAllowedTargetIP(ip, mode) {
		return fmt.Errorf("URL host %s resolves to a disallowed address", host)
	}

	return nil
}

var blockedIPPrefixValues = []string{
	"0.0.0.0/8",
	"100.64.0.0/10",
	"169.254.0.0/16",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"100::/64",
	"2001:db8::/32",
	"fc00::/7",
	"fe80::/10",
	"fec0::/10",
	"ff00::/8",
}

var allowedPrivateIPPrefixValues = []string{
	"10.0.0.0/8",
	"127.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"::1/128",
}

var metadataServiceAddrValues = []string{
	"169.254.169.254", // IPv4 metadata service
}

type securityPolicyConfig struct {
	blockedPrefixes        []netip.Prefix
	allowedPrivatePrefixes []netip.Prefix
	metadataAddrs          []netip.Addr
}

var (
	securityPolicyOnce   sync.Once
	securityPolicyParsed securityPolicyConfig
	securityPolicyErr    error
)

// ValidateSecurityConfig verifies hardcoded SSRF disallow lists at process startup.
func ValidateSecurityConfig() error {
	_, err := loadSecurityPolicyConfig()

	return err
}

func loadSecurityPolicyConfig() (securityPolicyConfig, error) {
	securityPolicyOnce.Do(func() {
		securityPolicyParsed, securityPolicyErr = parseSecurityPolicyConfig(
			blockedIPPrefixValues,
			allowedPrivateIPPrefixValues,
			metadataServiceAddrValues,
		)
	})

	if securityPolicyErr != nil {
		return securityPolicyConfig{}, securityPolicyErr
	}

	return securityPolicyParsed, nil
}

func parseSecurityPolicyConfig(
	blockedPrefixes []string,
	allowedPrivatePrefixes []string,
	metadataAddrs []string,
) (securityPolicyConfig, error) {
	parsedBlockedPrefixes := make([]netip.Prefix, 0, len(blockedPrefixes))
	for _, prefix := range blockedPrefixes {
		value, err := netip.ParsePrefix(prefix)
		if err != nil {
			return securityPolicyConfig{}, fmt.Errorf("invalid disallowed CIDR %q: %w", prefix, err)
		}

		parsedBlockedPrefixes = append(parsedBlockedPrefixes, value)
	}

	parsedAllowedPrivatePrefixes := make([]netip.Prefix, 0, len(allowedPrivatePrefixes))
	for _, prefix := range allowedPrivatePrefixes {
		value, err := netip.ParsePrefix(prefix)
		if err != nil {
			return securityPolicyConfig{}, fmt.Errorf("invalid allowed private CIDR %q: %w", prefix, err)
		}

		parsedAllowedPrivatePrefixes = append(parsedAllowedPrivatePrefixes, value)
	}

	parsedMetadataAddrs := make([]netip.Addr, 0, len(metadataAddrs))
	for _, value := range metadataAddrs {
		addr, err := netip.ParseAddr(value)
		if err != nil {
			return securityPolicyConfig{}, fmt.Errorf("invalid disallowed IP %q: %w", value, err)
		}

		parsedMetadataAddrs = append(parsedMetadataAddrs, addr)
	}

	return securityPolicyConfig{
		blockedPrefixes:        parsedBlockedPrefixes,
		allowedPrivatePrefixes: parsedAllowedPrivatePrefixes,
		metadataAddrs:          parsedMetadataAddrs,
	}, nil
}

// isDisallowedIP returns true for IPs in non-public, loopback, link-local, or metadata ranges.
func isDisallowedIP(ip net.IP) bool {
	return !isAllowedTargetIP(ip, targetValidationModePublic)
}

func isAllowedTargetIP(ip net.IP, mode targetValidationMode) bool {
	switch classifyTargetIP(ip) {
	case targetIPDecisionAllow:
		return true
	case targetIPDecisionAllowInPrivateMode:
		return mode == targetValidationModePrivate
	case targetIPDecisionBlock:
		return false
	}

	return false
}

func classifyTargetIP(ip net.IP) targetIPDecision {
	policy, err := loadSecurityPolicyConfig()
	if err != nil {
		return targetIPDecisionBlock
	}

	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return targetIPDecisionBlock
	}

	if addr.Is4In6() {
		addr = addr.Unmap()
	}

	if addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return targetIPDecisionBlock
	}

	for _, metadataAddr := range policy.metadataAddrs {
		if addr == metadataAddr {
			return targetIPDecisionBlock
		}
	}

	for _, prefix := range policy.allowedPrivatePrefixes {
		if prefix.Contains(addr) {
			return targetIPDecisionAllowInPrivateMode
		}
	}

	for _, prefix := range policy.blockedPrefixes {
		if prefix.Contains(addr) {
			return targetIPDecisionBlock
		}
	}

	return targetIPDecisionAllow
}
