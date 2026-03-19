package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func envOr(getenv getenvFunc, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}

	return fallback
}

func parseModules(raw string) ([]string, error) {
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

func normalizeTargetURLs(raw []string) ([]string, error) {
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
