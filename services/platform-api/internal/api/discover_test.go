package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func discoverViaHandler(t *testing.T, siteURL string) discoveryResult {
	t.Helper()

	server, _, _ := newTestServer(t)
	server.discoveryMode = targetValidationModePrivate

	body := strings.NewReader(fmt.Sprintf(`{"url":%q}`, siteURL))
	rec := httptest.NewRecorder()
	server.handleDiscover(rec, httptest.NewRequest(http.MethodPost, "/api/v1/discover", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var result discoveryResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return result
}

func TestDiscoverReadsSitemapIndexFromRobots(t *testing.T) {
	mux := http.NewServeMux()
	site := httptest.NewServer(mux)
	t.Cleanup(site.Close)

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "User-agent: *\nSitemap: %s/maps/index.xml\n", site.URL)
	})
	mux.HandleFunc("/maps/index.xml", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<sitemapindex><sitemap><loc>%s/maps/pages.xml</loc></sitemap></sitemapindex>`, site.URL)
	})
	mux.HandleFunc("/maps/pages.xml", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<urlset>
			<url><loc>%[1]s/</loc></url>
			<url><loc>%[1]s/about</loc></url>
			<url><loc>%[1]s/about#team</loc></url>
			<url><loc>https://elsewhere.example/page</loc></url>
		</urlset>`, site.URL)
	})

	result := discoverViaHandler(t, site.URL)

	want := []string{site.URL + "/", site.URL + "/about"}
	if result.Source != discoverySourceSitemap || !slices.Equal(result.URLs, want) {
		t.Fatalf("result = %+v, want sitemap %v", result, want)
	}
}

// A single-page app answers /sitemap.xml with 200 and its HTML shell.
func TestDiscoverCrawlsWhenSitemapIsSoft200HTML(t *testing.T) {
	pages := map[string]string{
		"/":                  `<a href="/projects">Projects</a><a href="/cv.pdf">CV</a><a href="/cdn-cgi/l/email-protection">Email</a><a href="https://elsewhere.example/">Out</a>`,
		"/projects":          `<a href="/projects/one">One</a><a href="/">Home</a>`,
		"/projects/one":      `<a href="/projects/one/deep">Deep</a>`,
		"/projects/one/deep": `<a href="/too-deep">Too deep</a>`,
	}

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if page, ok := pages[r.URL.Path]; ok {
			fmt.Fprint(w, page)

			return
		}

		fmt.Fprint(w, `<!DOCTYPE html><html><body><div id="root"></div></body></html>`)
	}))
	t.Cleanup(site.Close)

	result := discoverViaHandler(t, site.URL)

	want := []string{site.URL + "/", site.URL + "/projects", site.URL + "/projects/one"}
	if result.Source != discoverySourceCrawl || !slices.Equal(result.URLs, want) {
		t.Fatalf("result = %+v, want crawl %v", result, want)
	}
}

func TestDiscoverCapsResults(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sitemap.xml" {
			http.NotFound(w, r)

			return
		}

		fmt.Fprint(w, "<urlset>")

		for i := range maxURLCount + 20 {
			fmt.Fprintf(w, "<url><loc>/page-%d</loc></url>", i)
		}

		fmt.Fprint(w, "</urlset>")
	}))
	t.Cleanup(site.Close)

	result := discoverViaHandler(t, site.URL)
	if len(result.URLs) != maxURLCount || !result.Truncated {
		t.Fatalf("got %d urls, truncated = %v; want %d and true", len(result.URLs), result.Truncated, maxURLCount)
	}
}

func TestDiscoverRejectsDisallowedTargets(t *testing.T) {
	server, _, _ := newTestServer(t)

	for _, target := range []string{"http://169.254.169.254/latest/meta-data", "http://127.0.0.1:8080", "ftp://example.com"} {
		body := strings.NewReader(fmt.Sprintf(`{"url":%q}`, target))
		rec := httptest.NewRecorder()
		server.handleDiscover(rec, httptest.NewRequest(http.MethodPost, "/api/v1/discover", body))

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", target, rec.Code)
		}
	}
}

// The dial check is what stops DNS rebinding and redirects into the private
// network: every connection, including each redirect hop, goes through it.
func TestDiscoveryClientRefusesPrivateAddressAtDialTime(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "secret")
	}))
	t.Cleanup(internal.Close)

	resp, err := newDiscoveryClient(targetValidationModePublic).Get(internal.URL)
	if err == nil {
		_ = resp.Body.Close()

		t.Fatal("public discovery client connected to a loopback address")
	}

	if !errors.Is(err, errDiscoveryBlockedAddress) {
		t.Fatalf("err = %v, want errDiscoveryBlockedAddress", err)
	}
}
