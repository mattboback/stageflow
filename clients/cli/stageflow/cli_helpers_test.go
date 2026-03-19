package main

import (
	"strings"
	"testing"
)

func TestNormalizeTargetURLs(t *testing.T) {
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
		{
			name:        "empty list rejected",
			raw:         []string{},
			errContains: "at least one URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTargetURLs(tt.raw)
			if tt.errContains != "" {
				if err == nil {
					t.Fatalf("normalizeTargetURLs(%#v) err = nil, want non-nil", tt.raw)
				}

				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf(
						"normalizeTargetURLs(%#v) err = %q, want to contain %q",
						tt.raw,
						err.Error(),
						tt.errContains,
					)
				}

				return
			}

			requireNoErr(t, err)
			requireDeepEqual(t, got, tt.want, "normalized URLs")
		})
	}
}
