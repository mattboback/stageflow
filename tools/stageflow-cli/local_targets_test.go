package main

import (
	"strings"
	"testing"
)

func TestContainsLoopbackTargets(t *testing.T) {
	tests := []struct {
		name string
		urls []string
		want bool
	}{
		{
			name: "localhost",
			urls: []string{"http://localhost", "http://localhost:3000"},
			want: true,
		},
		{
			name: "ipv4 loopback",
			urls: []string{"http://127.0.0.1", "http://127.0.0.1:3000"},
			want: true,
		},
		{
			name: "ipv6 loopback",
			urls: []string{"http://[::1]", "http://[::1]:3000"},
			want: true,
		},
		{
			name: "public and invalid",
			urls: []string{"https://example.com", "", "not-a-url"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsLoopbackTargets(tt.urls); got != tt.want {
				t.Fatalf("containsLoopbackTargets(%#v) = %v, want %v", tt.urls, got, tt.want)
			}
		})
	}
}

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		want        bool
		wantErr     bool
		errContains string
	}{
		{
			name:    "localhost",
			raw:     "http://localhost:8080",
			want:    true,
			wantErr: false,
		},
		{
			name:    "ipv4 loopback",
			raw:     "http://127.0.0.1:8080",
			want:    true,
			wantErr: false,
		},
		{
			name:    "ipv6 loopback",
			raw:     "http://[::1]:8080",
			want:    true,
			wantErr: false,
		},
		{
			name:    "public host",
			raw:     "https://stageflow.org",
			want:    false,
			wantErr: false,
		},
		{
			name:        "missing scheme/host parse",
			raw:         "localhost:8080",
			want:        false,
			wantErr:     true,
			errContains: "missing host",
		},
		{
			name:        "not a url",
			raw:         "not-a-url",
			want:        false,
			wantErr:     true,
			errContains: "missing host",
		},
		{
			name:        "scheme but empty host",
			raw:         "http://",
			want:        false,
			wantErr:     true,
			errContains: "missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isLoopbackHost(tt.raw)
			if got != tt.want {
				t.Fatalf("isLoopbackHost(%q) = %v, want %v", tt.raw, got, tt.want)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatalf("isLoopbackHost(%q) err = nil, want non-nil", tt.raw)
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("isLoopbackHost(%q) err = %q, want to contain %q", tt.raw, err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("isLoopbackHost(%q) err = %v, want nil", tt.raw, err)
			}
		})
	}
}

func TestValidateLocalTargets(t *testing.T) {
	tests := []struct {
		name        string
		apiBaseURL  string
		targetURLs  []string
		wantErr     bool
		errContains string
	}{
		{
			name:       "local api + loopback targets",
			apiBaseURL: "http://localhost:8080",
			targetURLs: []string{"http://127.0.0.1:3000"},
			wantErr:    false,
		},
		{
			name:        "non-local api + loopback targets",
			apiBaseURL:  "https://stageflow.org",
			targetURLs:  []string{"http://localhost:3000"},
			wantErr:     true,
			errContains: "refusing to submit loopback targets",
		},
		{
			name:       "non-local api + public targets",
			apiBaseURL: "https://stageflow.org",
			targetURLs: []string{"https://example.com"},
			wantErr:    false,
		},
		{
			name:        "invalid api url + loopback targets",
			apiBaseURL:  "http://",
			targetURLs:  []string{"http://127.0.0.1"},
			wantErr:     true,
			errContains: "missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocalTargets(tt.apiBaseURL, tt.targetURLs)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateLocalTargets(%q, %#v) err = nil, want non-nil", tt.apiBaseURL, tt.targetURLs)
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf(
						"validateLocalTargets(%q, %#v) err = %q, want to contain %q",
						tt.apiBaseURL,
						tt.targetURLs,
						err.Error(),
						tt.errContains,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("validateLocalTargets(%q, %#v) err = %v, want nil", tt.apiBaseURL, tt.targetURLs, err)
			}
		})
	}
}
