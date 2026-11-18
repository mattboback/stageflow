package api

import (
	"context"
	"net"
)

// validateTargetURLs enforces basic SSRF protections for URL-based jobs.
// It rejects URLs with non-HTTP schemes and targets in private/loopback/metadata ranges.
func validateTargetURLs(ctx context.Context, urls []string) error {
	return validateTargetURLsWithResolver(ctx, net.DefaultResolver, urls, targetValidationModePublic)
}

// isDisallowedIP returns true for IPs in non-public, loopback, link-local, or metadata ranges.
func isDisallowedIP(ip net.IP) bool {
	return !isAllowedTargetIP(ip, targetValidationModePublic)
}
