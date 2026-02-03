package storage

import (
	"testing"
)

func TestBuildPublicProxyURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *MinIOConfig
		bucket   string
		path     string
		expected string
	}{
		{
			name: "HTTPS with public endpoint",
			config: &MinIOConfig{
				Endpoint:       "minio:9000",
				PublicEndpoint: "example.com",
				PublicUseSSL:   true,
				UseProxyURLs:   true,
			},
			bucket:   "scanner-artifacts",
			path:     "job123/report.html",
			expected: "https://example.com/scanner-artifacts/job123/report.html",
		},
		{
			name: "HTTP with public endpoint",
			config: &MinIOConfig{
				Endpoint:       "minio:9000",
				PublicEndpoint: "localhost:8080",
				PublicUseSSL:   false,
				UseProxyURLs:   true,
			},
			bucket:   "scanner-staging",
			path:     "uploads/file.zip",
			expected: "http://localhost:8080/scanner-staging/uploads/file.zip",
		},
		{
			name: "Falls back to internal endpoint when no public endpoint",
			config: &MinIOConfig{
				Endpoint:       "minio:9000",
				PublicEndpoint: "",
				PublicUseSSL:   false,
				UseProxyURLs:   true,
			},
			bucket:   "scanner-artifacts",
			path:     "test/file.json",
			expected: "http://minio:9000/scanner-artifacts/test/file.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MinIOClient{config: tt.config}
			result := client.buildPublicProxyURL(tt.bucket, tt.path)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
