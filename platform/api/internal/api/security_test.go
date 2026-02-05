package api

import (
	"context"
	"testing"
)

func TestValidateTargetURLs_AllowsPublicURLs(t *testing.T) {
	resolver := defaultSecurityTestResolver(t)

	tests := []struct {
		name string
		url  string
	}{
		{"Public domain with HTTPS", "https://example.com"},
		{"Public domain with HTTP", "http://example.com"},
		{"Public IP address", "https://8.8.8.8"},
		{"Public domain with path", "https://example.com/path/to/page"},
		{"Public domain with port", "https://example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetURLsWithResolver(context.Background(), resolver, []string{tt.url})
			if err != nil {
				t.Errorf("Expected %s to be allowed, got error: %v", tt.url, err)
			}
		})
	}
}

func TestValidateTargetURLs_BlocksPrivateIPv4(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"Loopback 127.0.0.1", "http://127.0.0.1"},
		{"Loopback 127.1.1.1", "http://127.1.1.1"},
		{"Private 10.x", "http://10.0.0.1"},
		{"Private 192.168.x", "http://192.168.1.1"},
		{"Private 172.16.x", "http://172.16.0.1"},
		{"Private 172.31.x (edge)", "http://172.31.255.255"},
		{"AWS metadata endpoint", "http://169.254.169.254"},
		{"Link-local", "http://169.254.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetURLs(context.Background(), []string{tt.url})
			if err == nil {
				t.Errorf("Expected %s to be blocked, but was allowed", tt.url)
			}
		})
	}
}

func TestValidateTargetURLs_BlocksPrivateIPv6(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"IPv6 loopback", "http://[::1]"},
		{"IPv6 loopback expanded", "http://[0000:0000:0000:0000:0000:0000:0000:0001]"},
		{"IPv6 unique local fc00", "http://[fc00::1]"},
		{"IPv6 unique local fd00", "http://[fd00::1]"},
		{"IPv6 link local", "http://[fe80::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetURLs(context.Background(), []string{tt.url})
			if err == nil {
				t.Errorf("Expected %s to be blocked, but was allowed", tt.url)
			}
		})
	}
}

func TestValidateTargetURLs_BlocksInvalidSchemes(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"File scheme", "file:///etc/passwd"},
		{"FTP scheme", "ftp://example.com"},
		{"Gopher scheme", "gopher://example.com"},
		{"Data scheme", "data:text/html,<script>alert(1)</script>"},
		{"JavaScript scheme", "javascript:alert(1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetURLs(context.Background(), []string{tt.url})
			if err == nil {
				t.Errorf("Expected %s to be blocked, but was allowed", tt.url)
			}
		})
	}
}

func TestValidateTargetURLs_RejectsEmptyURLs(t *testing.T) {
	resolver := defaultSecurityTestResolver(t)

	tests := []struct {
		name string
		urls []string
	}{
		{"Single empty string", []string{""}},
		{"Whitespace only", []string{"   "}},
		{"Empty in list", []string{"https://example.com", "", "https://test.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetURLsWithResolver(context.Background(), resolver, tt.urls)
			if err == nil {
				t.Errorf("Expected empty/whitespace URL to be rejected")
			}
		})
	}
}

func TestValidateTargetURLs_RejectsInvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"Not a URL", "not-a-url"},
		{"Missing scheme", "example.com"},
		{"Invalid characters", "http://exam ple.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetURLs(context.Background(), []string{tt.url})
			if err == nil {
				t.Errorf("Expected %s to be rejected as invalid, but was allowed", tt.url)
			}
		})
	}
}

func TestValidateTargetURLs_MultipleURLs(t *testing.T) {
	resolver := defaultSecurityTestResolver(t)

	t.Run("All valid URLs", func(t *testing.T) {
		urls := []string{
			"https://example.com",
			"https://example.com",
			"http://8.8.8.8",
		}

		err := validateTargetURLsWithResolver(context.Background(), resolver, urls)
		if err != nil {
			t.Errorf("Expected all valid URLs to be allowed, got error: %v", err)
		}
	})

	t.Run("One invalid URL in list", func(t *testing.T) {
		urls := []string{
			"https://example.com",
			"http://127.0.0.1", // Should be blocked
			"https://example.com",
		}

		err := validateTargetURLsWithResolver(context.Background(), resolver, urls)
		if err == nil {
			t.Error("Expected validation to fail when one URL is private")
		}
	})
}

func TestValidateTargetURLs_PortHandling(t *testing.T) {
	resolver := defaultSecurityTestResolver(t)

	t.Run("Public IP with port", func(t *testing.T) {
		err := validateTargetURLs(context.Background(), []string{"http://8.8.8.8:80"})
		if err != nil {
			t.Errorf("Public IP with port should be allowed, got: %v", err)
		}
	})

	t.Run("Private IP with port", func(t *testing.T) {
		err := validateTargetURLs(context.Background(), []string{"http://127.0.0.1:8080"})
		if err == nil {
			t.Error("Private IP with port should be blocked")
		}
	})

	t.Run("Domain with port", func(t *testing.T) {
		err := validateTargetURLsWithResolver(context.Background(), resolver, []string{"https://example.com:8443"})
		if err != nil {
			t.Errorf("Public domain with port should be allowed, got: %v", err)
		}
	})
}
