package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only returns nil",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "single value",
			input:    scannerTypeAxe,
			expected: []string{scannerTypeAxe},
		},
		{
			name:     "multiple values",
			input:    scannerTypeAxe + ",lighthouse,seo",
			expected: []string{scannerTypeAxe, "lighthouse", "seo"},
		},
		{
			name:     "trims whitespace around values",
			input:    " " + scannerTypeAxe + " , lighthouse , seo ",
			expected: []string{scannerTypeAxe, "lighthouse", "seo"},
		},
		{
			name:     "filters empty parts",
			input:    scannerTypeAxe + ",,lighthouse",
			expected: []string{scannerTypeAxe, "lighthouse"},
		},
		{
			name:     "handles trailing comma",
			input:    scannerTypeAxe + ",lighthouse,",
			expected: []string{scannerTypeAxe, "lighthouse"},
		},
		{
			name:     "handles leading comma",
			input:    "," + scannerTypeAxe + ",lighthouse",
			expected: []string{scannerTypeAxe, "lighthouse"},
		},
		{
			name:     "handles multiple empty parts",
			input:    scannerTypeAxe + ",,,lighthouse,,seo",
			expected: []string{scannerTypeAxe, "lighthouse", "seo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCSV(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d: %v", len(tt.expected), len(result), result)

				return
			}

			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
				}
			}
		})
	}
}

func TestIntakeRejectsUnknownModuleWithTypedError(t *testing.T) {
	server, _, _ := newTestServer(t)

	urlReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs/urls",
		strings.NewReader(`{"urls":["https://example.com"],"modules":["axe","no-such-scanner"]}`),
	)
	urlReq.Header.Set("Content-Type", "application/json")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe,no-such-scanner")

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	zipReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	zipReq.Header.Set("Content-Type", writer.FormDataContentType())

	for name, run := range map[string]func(*httptest.ResponseRecorder){
		"url": func(rr *httptest.ResponseRecorder) { server.handleJobURLSubmit(rr, urlReq) },
		"zip": func(rr *httptest.ResponseRecorder) { server.handleJobZipUpload(rr, zipReq) },
	} {
		rr := httptest.NewRecorder()
		run(rr)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400: %s", name, rr.Code, rr.Body.String())
		}

		var got httputil.ErrorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: decode error body: %v", name, err)
		}

		if got.Error.Code != httputil.ErrCodeUnsupportedModule ||
			got.Error.Message != "Unsupported scanner module: no-such-scanner" ||
			got.Error.Details != "Supported modules: axe, lighthouse, link-checker, security-headers, seo" {
			t.Fatalf("%s: unexpected error %+v", name, got.Error)
		}
	}
}
