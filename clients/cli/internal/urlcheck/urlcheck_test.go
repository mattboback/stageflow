package urlcheck

import (
	"reflect"
	"strings"
	"testing"
)

func TestContainsPrivateTargets(t *testing.T) {
	tests := []struct {
		name string
		urls []string
		want bool
	}{
		{"localhost", []string{"http://localhost", "http://localhost:3000"}, true},
		{"ipv4 loopback", []string{"http://127.0.0.1", "http://127.0.0.1:3000"}, true},
		{"ipv6 loopback", []string{"http://[::1]", "http://[::1]:3000"}, true},
		{"public and invalid", []string{"https://example.com", "", "not-a-url"}, false},
		{"rfc1918 private ipv4", []string{"http://10.42.0.7:3000", "http://192.168.1.50"}, true},
		{"ipv6 ula", []string{"http://[fd12:3456:789a::1]:3000"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsPrivateTargets(tt.urls); got != tt.want {
				t.Fatalf("ContainsPrivateTargets(%#v) = %v, want %v", tt.urls, got, tt.want)
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
		{name: "localhost", raw: "http://localhost:8080", want: true},
		{name: "ipv4 loopback", raw: "http://127.0.0.1:8080", want: true},
		{name: "ipv6 loopback", raw: "http://[::1]:8080", want: true},
		{name: "public host", raw: "https://stageflow.org", want: false},
		{name: "missing scheme/host parse", raw: "localhost:8080", wantErr: true, errContains: "missing host"},
		{name: "not a url", raw: "not-a-url", wantErr: true, errContains: "missing host"},
		{name: "scheme but empty host", raw: "http://", wantErr: true, errContains: "missing host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsLoopbackHost(tt.raw)
			if got != tt.want {
				t.Fatalf("IsLoopbackHost(%q) = %v, want %v", tt.raw, got, tt.want)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatalf("IsLoopbackHost(%q) err = nil, want non-nil", tt.raw)
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("IsLoopbackHost(%q) err = %q, want to contain %q", tt.raw, err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("IsLoopbackHost(%q) err = %v, want nil", tt.raw, err)
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
			name: "local api + loopback targets", apiBaseURL: "http://localhost:8080",
			targetURLs: []string{"http://127.0.0.1:3000"},
		},
		{
			name: "non-local api + loopback targets", apiBaseURL: "https://stageflow.org",
			targetURLs: []string{"http://localhost:3000"}, wantErr: true,
			errContains: "refusing to submit private/loopback targets",
		},
		{
			name: "non-local api + private targets", apiBaseURL: "https://stageflow.org",
			targetURLs: []string{"http://10.0.0.42:3000"}, wantErr: true,
			errContains: "refusing to submit private/loopback targets",
		},
		{
			name: "non-local api + public targets", apiBaseURL: "https://stageflow.org",
			targetURLs: []string{"https://example.com"},
		},
		{
			name: "invalid api url + loopback targets", apiBaseURL: "http://",
			targetURLs: []string{"http://127.0.0.1"}, wantErr: true, errContains: "missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocalTargets(tt.apiBaseURL, tt.targetURLs)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateLocalTargets(%q, %#v) err = nil, want non-nil", tt.apiBaseURL, tt.targetURLs)
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf(
						"ValidateLocalTargets(%q, %#v) err = %q, want to contain %q",
						tt.apiBaseURL, tt.targetURLs, err.Error(), tt.errContains,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("ValidateLocalTargets(%q, %#v) err = %v, want nil", tt.apiBaseURL, tt.targetURLs, err)
			}
		})
	}
}

func TestNormalizeTargets(t *testing.T) {
	tests := []struct {
		name        string
		raw         []string
		want        []string
		errContains string
	}{
		{
			name: "schemeless localhost gets http",
			raw:  []string{"localhost:8000"},
			want: []string{"http://localhost:8000"},
		},
		{
			name: "schemeless host with path gets http",
			raw:  []string{"example.com/docs"},
			want: []string{"http://example.com/docs"},
		},
		{
			name: "explicit https unchanged",
			raw:  []string{"https://localhost:8443"},
			want: []string{"https://localhost:8443"},
		},
		{
			name:        "unsupported scheme still rejected",
			raw:         []string{"ftp://example.com"},
			errContains: "unsupported URL scheme",
		},
		{name: "empty list rejected", raw: []string{}, errContains: "at least one URL is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTargets(tt.raw)
			if tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("NormalizeTargets(%#v) err = %v, want contains %q", tt.raw, err, tt.errContains)
				}

				return
			}

			if err != nil {
				t.Fatalf("NormalizeTargets(%#v) err = %v", tt.raw, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("NormalizeTargets(%#v) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}
