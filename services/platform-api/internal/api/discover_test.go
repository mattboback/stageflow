package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html></html>")
	})
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
		if r.URL.Path == "/robots.txt" {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path != "/sitemap.xml" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html></html>")

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
	if len(result.URLs) != discoveryMaxCrawlPages || !result.Truncated {
		t.Fatalf(
			"got %d urls, truncated = %v; want %d and true",
			len(result.URLs),
			result.Truncated,
			discoveryMaxCrawlPages,
		)
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

func TestDiscoverUnionsSitemapWithVerifiedPageRelativeLinks(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			http.NotFound(w, r)
		case "/sitemap.xml":
			fmt.Fprint(w, `<urlset><url><loc>/dir/page</loc></url><url><loc>/missing</loc></url></urlset>`)
		case "/dir/page":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<a href="child">Child</a><a href="/query?sort=new">Query</a><a href="/doc.pdf">PDF</a>`)
		case "/dir/child", "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)

	want := []string{site.URL + "/", site.URL + "/dir/page", site.URL + "/dir/child"}
	if !slices.Equal(result.URLs, want) || !slices.Contains(result.Sources, "sitemap") ||
		!slices.Contains(result.Sources, "crawl") {
		t.Fatalf("result=%+v", result)
	}

	if result.Complete || !slices.Contains(result.StopReasons, "page_fetch_failed") {
		t.Fatalf("missing page must be reported: %+v", result)
	}
}

func TestDiscoverPreservesExplicitSeedQueryAndExcludesDiscoveredQueries(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<a href="/?seed=1">Seed</a><a href="/?other=2">Other</a><a href="/about">About</a>`)
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL+"/?seed=1")
	if !slices.Equal(result.URLs, []string{site.URL + "/?seed=1", site.URL + "/about"}) {
		t.Fatalf("result=%+v", result)
	}
}

func TestDiscoverRespectsRobotsAndBlocksRedirectBeforeFetch(t *testing.T) {
	blockedFetches := 0

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			fmt.Fprint(w, "User-agent: *\nDisallow: /private\nAllow: /private/public\nDisallow: /secret*\n")
		case "/sitemap.xml":
			http.NotFound(w, r)
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(
				w,
				`<a href="/private">No</a><a href="/private/public">Yes</a><a href="/redirect">Redirect</a><a href="/secret-file">No</a>`,
			)
		case "/redirect":
			http.Redirect(w, r, "/private", http.StatusFound)
		case "/private", "/secret-file":
			blockedFetches++

			w.Header().Set("Content-Type", "text/html")
		default:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html></html>")
		}
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if blockedFetches != 0 || !slices.Equal(result.URLs, []string{site.URL + "/", site.URL + "/private/public"}) {
		t.Fatalf("blockedFetches=%d result=%+v", blockedFetches, result)
	}
}

func TestDiscoverRejectsCrossOriginRedirects(t *testing.T) {
	externalFetches := 0

	external := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { externalFetches++; fmt.Fprint(w, "external") }),
	)
	defer external.Close()

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, external.URL, http.StatusFound)
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if len(result.URLs) != 0 || externalFetches != 0 || result.Complete {
		t.Fatalf("cross-origin fetched=%d result=%+v", externalFetches, result)
	}
}

func TestDiscoveryOriginIncludesSchemeAndEffectivePort(t *testing.T) {
	base, _ := url.Parse("https://example.com/path")
	for raw, want := range map[string]bool{"https://EXAMPLE.com:443/": true, "http://example.com/": false, "https://example.com:444/": false, "https://other.example/": false} {
		candidate, _ := url.Parse(raw)
		if got := sameDiscoveryOrigin(candidate, base); got != want {
			t.Errorf("%s=%v want%v", raw, got, want)
		}
	}
}

func TestDiscoverSitemapFetchCapIsGlobal(t *testing.T) {
	sitemapFetches := 0

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/robots.txt":
			http.NotFound(w, r)
		case r.URL.Path == "/sitemap.xml":
			sitemapFetches++

			fmt.Fprint(w, "<sitemapindex>")

			for i := range 12 {
				fmt.Fprintf(w, "<sitemap><loc>/map-%d.xml</loc></sitemap>", i)
			}

			fmt.Fprint(w, "</sitemapindex>")
		case strings.HasPrefix(r.URL.Path, "/map-"):
			sitemapFetches++

			fmt.Fprint(w, "<urlset></urlset>")
		default:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html></html>")
		}
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if sitemapFetches != discoveryMaxSitemaps || !slices.Contains(result.StopReasons, "sitemap_limit") {
		t.Fatalf("fetches=%d result=%+v", sitemapFetches, result)
	}
}

func TestDiscoverReportsDepthLimitAndOnlyFetchedHTML(t *testing.T) {
	fetches := []string{}

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		fetches = append(fetches, r.URL.Path)

		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/":
			fmt.Fprint(w, `<a href="/one">One</a>`)
		case "/one":
			fmt.Fprint(w, `<a href="/two">Two</a>`)
		case "/two":
			fmt.Fprint(w, `<a href="/three">Three</a>`)
		}
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if !slices.Equal(fetches, []string{"/", "/one", "/two"}) || !slices.Contains(result.StopReasons, "depth_limit") {
		t.Fatalf("fetches=%v result=%+v", fetches, result)
	}
}

func TestDiscoveryRobotsSpecificGroupAndLongestAllow(t *testing.T) {
	rules, _, err := parseDiscoveryRobots(
		[]byte("User-agent: *\nDisallow: /\nUser-agent: StageFlow\nDisallow: /hidden*\nAllow: /hidden/public$\n"),
	)
	if err != nil {
		t.Fatal(err)
	}

	for raw, want := range map[string]bool{"https://example.com/": true, "https://example.com/hidden": false, "https://example.com/hidden/public": true, "https://example.com/hidden/public/more": false} {
		target, _ := url.Parse(raw)
		if got := robotsAllow(rules, target); got != want {
			t.Errorf("%s=%v want%v", raw, got, want)
		}
	}
}

func TestDiscoverRobotsFailureDoesNotFetchPages(t *testing.T) {
	pageFetches := 0

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/robots.txt" {
			pageFetches++
		}

		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if pageFetches != 0 || len(result.URLs) != 0 || result.Complete ||
		!slices.Contains(result.StopReasons, "robots_unavailable") {
		t.Fatalf("result=%+v pageFetches=%d", result, pageFetches)
	}
}

func TestDiscoverUsesFinalRedirectURLAndHTMLBase(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt", "/sitemap.xml":
			http.NotFound(w, r)
		case "/start":
			http.Redirect(w, r, "/docs/", http.StatusFound)
		case "/docs/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<a href="child">Child</a><a href="/bad-type">Not HTML</a>`)
		case "/docs/child":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<base href="/assets/"><a href="page">Page</a>`)
		case "/assets/page":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html></html>")
		case "/bad-type":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"a":"<a href='/fake'>Fake</a>"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL+"/start")
	if !slices.Equal(result.URLs, []string{site.URL + "/docs/", site.URL + "/docs/child", site.URL + "/assets/page"}) {
		t.Fatalf("result=%+v", result)
	}
}

func TestDiscoverRejectsEmbeddedCredentialsBeforeFetch(t *testing.T) {
	fetchedCredentials := false

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); ok {
			fetchedCredentials = true
		}

		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<a href="http://user:secret@%s/private">Credential URL</a>`, r.Host)
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if fetchedCredentials || len(result.URLs) != 1 {
		t.Fatalf("credentials=%v result=%+v", fetchedCredentials, result)
	}
}

func TestDiscoverDoesNotFollowNewQueryRedirect(t *testing.T) {
	queryFetches := 0

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			queryFetches++
		}

		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/?view=login", http.StatusFound)
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if queryFetches != 0 || len(result.URLs) != 0 || result.Complete {
		t.Fatalf("queryFetches=%d result=%+v", queryFetches, result)
	}
}

func TestDiscoverOversizeHTMLIsExcluded(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, strings.Repeat("x", discoveryMaxBodyBytes+1))
	}))
	defer site.Close()

	result := discoverViaHandler(t, site.URL)
	if len(result.URLs) != 0 || result.Complete {
		t.Fatalf("result=%+v", result)
	}
}

func TestDiscoverRejectsUnknownFieldsAndTrailingBodies(t *testing.T) {
	server, _, _ := newTestServer(t)

	for _, body := range []string{`{"url":"https://example.com","allow_private_targets":true}`, `{"url":"https://example.com"} {"url":"https://other.example"}`} {
		rec := httptest.NewRecorder()
		server.handleDiscover(rec, httptest.NewRequest(http.MethodPost, "/api/v1/discover", strings.NewReader(body)))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status%d body%s", rec.Code, rec.Body.String())
		}
	}
}

func TestDiscoveryBodyLimitAndRobotsParsingLimits(t *testing.T) {
	rules, _, err := parseDiscoveryRobots(
		[]byte("User-agent: *\nDisallow: /" + strings.Repeat("x", maxURLLength) + "\n"),
	)
	if err == nil || len(rules) != 0 {
		t.Fatalf("oversize robots rule must fail closed")
	}
}

func TestDiscoveryRobotsNormalizesPercentEncodedOctets(t *testing.T) {
	rules, _, err := parseDiscoveryRobots(
		[]byte("\ufeffUser-agent: *\nDisallow: /private\nDisallow: /café\nDisallow: /encoded%2Fslash\n"),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/private", "/%70rivate", "/caf%C3%A9", "/encoded%2fslash"} {
		target, parseErr := url.Parse("https://example.com" + path)
		if parseErr != nil {
			t.Fatal(parseErr)
		}

		if robotsAllow(rules, target) {
			t.Errorf("robots allowed %s", path)
		}
	}

	target, err := url.Parse("https://example.com/encoded/slash")
	if err != nil {
		t.Fatal(err)
	}

	if !robotsAllow(rules, target) {
		t.Fatal("reserved encoded slash was incorrectly decoded")
	}
}
