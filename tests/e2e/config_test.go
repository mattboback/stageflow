package e2e

import (
	"os"
	"strings"
	"testing"
)

var apiBaseURL = getAPIBaseURL()

func getAPIBaseURL() string {
	return resolveAPIBaseURL(os.Getenv("API_BASE_URL"))
}

func resolveAPIBaseURL(raw string) string {
	if raw == "" {
		return "http://localhost:8080/api/v1"
	}

	base := strings.TrimRight(raw, "/")

	if strings.HasSuffix(base, "/api/v1") || strings.Contains(base, "/api/") {
		return base
	}

	if strings.HasSuffix(base, "/api") {
		return base + "/v1"
	}

	return base + "/api/v1"
}

func TestResolveAPIBaseURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty_uses_default", in: "", want: "http://localhost:8080/api/v1"},
		{name: "host_only_appends_api_v1", in: "http://localhost:8080", want: "http://localhost:8080/api/v1"},
		{name: "api_path_preserved", in: "http://localhost:8080/api/v1/", want: "http://localhost:8080/api/v1"},
		{
			name: "versioned_api_path_preserved",
			in:   "http://localhost:8080/api/v2",
			want: "http://localhost:8080/api/v2",
		},
		{name: "api_without_version_gets_v1", in: "http://localhost:8080/api", want: "http://localhost:8080/api/v1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveAPIBaseURL(tc.in); got != tc.want {
				t.Fatalf("resolveAPIBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
