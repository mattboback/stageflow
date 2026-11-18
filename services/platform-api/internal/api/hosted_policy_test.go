package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

type unreadUploadBody struct{ t *testing.T }

func (b unreadUploadBody) Read([]byte) (int, error) {
	b.t.Fatal("disabled ZIP intake read request body")
	return 0, nil
}
func (b unreadUploadBody) Close() error { return nil }

func TestDisabledZIPIntakeDoesNotReadBodyOrTouchStorage(t *testing.T) {
	for _, token := range []string{"", "owner-token"} {
		t.Run(token, func(t *testing.T) {
			server, _, jobs := newTestServer(t)
			server.config.DisableZIPUploads = true
			server.config.APIToken = "owner-token"
			server.config.CheckDisk = func(_ context.Context) error { t.Fatal("disabled ZIP intake checked storage"); return nil }
			req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", unreadUploadBody{t})
			req.Header.Set("X-Api-Key", token)

			rec := httptest.NewRecorder()
			server.handleJobZipUpload(rec, req)

			if rec.Code != http.StatusForbidden || len(jobs.admitted) != 0 {
				t.Fatalf("status %d, admitted %d", rec.Code, len(jobs.admitted))
			}
		})
	}
}

func TestConfiguredURLPageLimit(t *testing.T) {
	for _, limit := range []int{25, 100} {
		urls := make([]string, limit+1)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://example.com/%d", i)
		}

		if err := validateURLSubmitRequest(urls[:limit], limit); err != nil {
			t.Fatalf("boundary %d rejected: %+v", limit, err)
		}

		if err := validateURLSubmitRequest(urls, limit); err == nil {
			t.Fatalf("accepted more than %d URLs", limit)
		}
	}
}

func TestDiscoveryUsesConfiguredLimitAndLabelsIncompleteCoverage(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<a href="/one">One</a><a href="/two">Two</a>`)
	}))
	defer site.Close()

	server, _, _ := newTestServer(t)
	server.config.MaxPages = 2
	server.discoveryMode = targetValidationModePrivate
	rec := httptest.NewRecorder()
	server.handleDiscover(
		rec,
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/discover",
			strings.NewReader(fmt.Sprintf(`{"url":%q}`, site.URL)),
		),
	)

	var result discoveryResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if len(result.URLs) != 2 || result.Complete || !result.Truncated ||
		!slices.Contains(result.StopReasons, "url_limit") {
		t.Fatalf("unexpected discovery: %+v", result)
	}
}

func TestHostedURLLimitAppliesToTokenCallers(t *testing.T) {
	for _, count := range []int{25, 26} {
		server, _, jobs := newTestServer(t)
		server.config.MaxPages = 25
		server.config.APIToken = "owner-token"

		urls := make([]string, count)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://example.com/%d", i)
		}

		body, _ := json.Marshal(map[string]any{"urls": urls})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", strings.NewReader(string(body)))
		req.Header.Set("X-Api-Key", "owner-token")

		rec := httptest.NewRecorder()
		server.handleJobURLSubmit(rec, req)

		expected := http.StatusCreated
		if count > 25 {
			expected = http.StatusBadRequest
		}

		if rec.Code != expected {
			t.Fatalf("%d URLs: status %d, body %s", count, rec.Code, rec.Body.String())
		}

		if count > 25 && len(jobs.admitted) != 0 {
			t.Fatal("oversized job was admitted")
		}
	}
}
