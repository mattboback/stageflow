package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

func validateLocalTargets(apiBaseURL string, targetURLs []string) error {
	if !containsLoopbackTargets(targetURLs) {
		return nil
	}

	isLocalAPI, err := isLoopbackHost(apiBaseURL)
	if err != nil {
		return err
	}

	if !isLocalAPI {
		return fmt.Errorf(
			"refusing to submit loopback targets to a non-local API (%s); run the StageFlow stack locally and set --api to http://localhost:8080",
			apiBaseURL,
		)
	}

	return nil
}

func isLoopbackHost(rawURL string) (bool, error) {
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

func containsLoopbackTargets(urls []string) bool {
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

		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return true
		}
	}

	return false
}
