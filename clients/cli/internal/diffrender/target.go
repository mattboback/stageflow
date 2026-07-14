package diffrender

import (
	"net"
	"strings"
)

// IsRemoteTarget reports whether a diff target should be scanned live.
func IsRemoteTarget(target string) bool {
	trimmed := strings.TrimSpace(target)

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}

	if isLocalDiffTarget(trimmed, lower) {
		return false
	}

	return isHostLikeTarget(hostFromTarget(trimmed))
}

func isLocalDiffTarget(trimmed, lower string) bool {
	return trimmed == "" || strings.Contains(trimmed, "://") || hasLocalPathPrefix(trimmed) ||
		(!strings.ContainsAny(trimmed, `/\`) && strings.HasSuffix(lower, ".json"))
}

func hasLocalPathPrefix(target string) bool {
	return strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") ||
		strings.HasPrefix(target, "../") || strings.HasPrefix(target, `\`) ||
		strings.HasPrefix(target, `.\`) || strings.HasPrefix(target, `..\`)
}

func hostFromTarget(trimmed string) string {
	host := trimmed
	if idx := strings.IndexAny(host, `/\`); idx >= 0 {
		host = host[:idx]
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	return strings.Trim(host, "[]")
}

func isHostLikeTarget(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil || strings.Contains(host, ".")
}
