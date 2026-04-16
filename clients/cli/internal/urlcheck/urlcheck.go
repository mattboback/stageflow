// Package urlcheck provides pure helpers for validating and classifying URLs
// that the CLI submits as scan targets.
package urlcheck

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// ParseModules splits a comma-separated scanner list into a normalized slice,
// rejecting empty entries.
func ParseModules(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	modules := strings.Split(raw, ",")

	parsed := make([]string, 0, len(modules))
	for _, item := range modules {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, errors.New("scanner list contains an empty module name")
		}

		parsed = append(parsed, trimmed)
	}

	return parsed, nil
}

// NormalizeTargets validates and normalizes raw scan target URLs. Bare hosts
// are prefixed with http:// to keep parity with the original CLI behavior.
func NormalizeTargets(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("at least one URL is required")
	}

	out := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, errors.New("URL list contains an empty item")
		}

		if !strings.Contains(trimmed, "://") {
			trimmed = "http://" + trimmed
		}

		u, err := url.Parse(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", trimmed, err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("unsupported URL scheme %q in %q (expected http or https)", u.Scheme, trimmed)
		}

		if u.Host == "" {
			return nil, fmt.Errorf("invalid URL %q: missing host", trimmed)
		}

		out = append(out, trimmed)
	}

	return out, nil
}

// ValidateLocalTargets rejects submissions that would send private/loopback
// scan targets to a non-local API base URL.
func ValidateLocalTargets(apiBaseURL string, targetURLs []string) error {
	if !ContainsPrivateTargets(targetURLs) {
		return nil
	}

	isLocalAPI, err := IsLoopbackHost(apiBaseURL)
	if err != nil {
		return err
	}

	if !isLocalAPI {
		return fmt.Errorf(
			"refusing to submit private/loopback targets to a non-local API (%s); run the StageFlow stack locally and set --api to http://localhost:8080",
			apiBaseURL,
		)
	}

	return nil
}

// IsLoopbackHost reports whether the given URL's host resolves to a loopback
// literal (localhost or a loopback IP).
func IsLoopbackHost(rawURL string) (bool, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false, fmt.Errorf("invalid API URL %q: %w", rawURL, err)
	}

	host := u.Host
	if host == "" {
		return false, errors.New("API URL missing host")
	}

	if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		host = h
	}

	host = strings.Trim(host, "[]")

	if strings.EqualFold(host, "localhost") {
		return true, nil
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true, nil
	}

	return false, nil
}

// ContainsPrivateTargets reports whether any of the URLs resolve to a
// private/loopback literal host.
func ContainsPrivateTargets(urls []string) bool {
	for _, raw := range urls {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		u, err := url.Parse(trimmed)
		if err != nil {
			continue
		}

		host := u.Host
		if host == "" {
			continue
		}

		if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
			host = h
		}

		host = strings.Trim(host, "[]")

		if strings.EqualFold(host, "localhost") {
			return true
		}

		if ip := net.ParseIP(host); isPrivateLiteralIP(ip) {
			return true
		}
	}

	return false
}

func isPrivateLiteralIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}

	if addr.Is4In6() {
		addr = addr.Unmap()
	}

	if addr.IsLoopback() {
		return true
	}

	return addr.IsPrivate()
}
