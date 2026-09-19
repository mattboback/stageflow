package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

const (
	discoveryBudget        = 45 * time.Second
	discoveryFetchTimeout  = 10 * time.Second
	discoveryMaxBodyBytes  = 5 << 20
	discoveryMaxRedirects  = 5
	discoveryMaxSitemaps   = 10
	discoveryMaxCrawlPages = 30
	discoveryCrawlDepth    = 2
	discoveryUserAgent     = "StageFlow-Discovery/1.0 (+https://stageflow.org)"

	discoverySourceSitemap = "sitemap"
	discoverySourceCrawl   = "crawl"
)

var errDiscoveryBlockedAddress = errors.New("connection to a disallowed address was blocked")

type discoveryResult struct {
	URLs      []string `json:"urls"`
	Source    string   `json:"source"`
	Truncated bool     `json:"truncated"`
}

// newDiscoveryClient returns the only HTTP client allowed to fetch a
// user-supplied URL from this service. Validating the hostname up front is not
// enough: DNS can answer differently at connect time and redirects can point
// anywhere, so the dialer checks the address actually being connected to.
func newDiscoveryClient(mode targetValidationMode) *http.Client {
	dialer := &net.Dialer{
		Timeout: discoveryFetchTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}

			ip := net.ParseIP(host)
			if ip == nil || !isAllowedTargetIP(ip, mode) {
				return errDiscoveryBlockedAddress
			}

			return nil
		},
	}

	return &http.Client{
		Timeout: discoveryFetchTimeout,
		Transport: &http.Transport{
			// No proxy: a proxy would connect on our behalf and bypass the dial check.
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   discoveryFetchTimeout,
			ResponseHeaderTimeout: discoveryFetchTimeout,
			DisableKeepAlives:     true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > discoveryMaxRedirects {
				return errors.New("too many redirects")
			}

			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("redirect to a non-HTTP scheme")
			}

			return nil
		},
	}
}

type discoverer struct {
	client *http.Client
	start  *url.URL
}

// discoverPages lists same-host pages for a site: sitemaps first, then a shallow
// link crawl when the site publishes none.
func discoverPages(ctx context.Context, client *http.Client, start *url.URL) discoveryResult {
	ctx, cancel := context.WithTimeout(ctx, discoveryBudget)
	defer cancel()

	d := &discoverer{client: client, start: start}

	if found := d.fromSitemaps(ctx); len(found) > 0 {
		return d.result(found, discoverySourceSitemap)
	}

	return d.result(d.crawl(ctx), discoverySourceCrawl)
}

func (d *discoverer) result(found []string, source string) discoveryResult {
	urls := make([]string, 0, len(found)+1)
	seen := make(map[string]bool)

	for _, candidate := range append([]string{d.start.String()}, found...) {
		normalized, ok := d.normalize(candidate)
		if !ok || seen[normalized] {
			continue
		}

		seen[normalized] = true

		urls = append(urls, normalized)
	}

	truncated := len(urls) > maxURLCount
	if truncated {
		urls = urls[:maxURLCount]
	}

	return discoveryResult{URLs: urls, Source: source, Truncated: truncated}
}

// normalize resolves a link against the start URL and keeps it only when it is
// a same-host HTTP(S) page.
func (d *discoverer) normalize(raw string) (string, bool) {
	parsed, err := d.start.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}

	if !strings.EqualFold(parsed.Host, d.start.Host) {
		return "", false
	}

	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return parsed.String(), true
}

func (d *discoverer) fetch(ctx context.Context, target string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", discoveryUserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, discoveryMaxBodyBytes))
	if err != nil {
		return nil, "", err
	}

	return body, resp.Header.Get("Content-Type"), nil
}

func (d *discoverer) fromSitemaps(ctx context.Context) []string {
	candidates := d.robotsSitemaps(ctx)
	candidates = append(candidates, d.start.ResolveReference(&url.URL{Path: "/sitemap.xml"}).String())

	var pages []string

	fetched := make(map[string]bool)

	for _, candidate := range candidates {
		sitemapURL, ok := d.normalize(candidate)
		if !ok || fetched[sitemapURL] {
			continue
		}

		fetched[sitemapURL] = true

		locs, children := d.readSitemap(ctx, sitemapURL)
		pages = append(pages, locs...)

		// An index is followed one level only; nested indexes are ignored.
		for _, child := range children {
			childURL, ok := d.normalize(child)
			if !ok || fetched[childURL] || len(fetched) > discoveryMaxSitemaps {
				continue
			}

			fetched[childURL] = true

			childLocs, _ := d.readSitemap(ctx, childURL)
			pages = append(pages, childLocs...)
		}

		if len(pages) > 0 {
			return pages
		}
	}

	return nil
}

func (d *discoverer) robotsSitemaps(ctx context.Context) []string {
	body, _, err := d.fetch(ctx, d.start.ResolveReference(&url.URL{Path: "/robots.txt"}).String())
	if err != nil {
		return nil
	}

	var sitemaps []string

	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), ":")
		if found && strings.EqualFold(strings.TrimSpace(key), "sitemap") {
			sitemaps = append(sitemaps, strings.TrimSpace(value))
		}
	}

	return sitemaps
}

type sitemapDocument struct {
	XMLName  xml.Name
	URLs     []sitemapLoc `xml:"url"`
	Sitemaps []sitemapLoc `xml:"sitemap"`
}

type sitemapLoc struct {
	Loc string `xml:"loc"`
}

// readSitemap trusts the document, not the status code: single-page apps answer
// /sitemap.xml with 200 and their HTML shell, which fails the root-element check.
func (d *discoverer) readSitemap(ctx context.Context, sitemapURL string) (pages, children []string) {
	body, _, err := d.fetch(ctx, sitemapURL)
	if err != nil {
		return nil, nil
	}

	var doc sitemapDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, nil
	}

	switch doc.XMLName.Local {
	case "urlset":
		for _, entry := range doc.URLs {
			pages = append(pages, entry.Loc)
		}
	case "sitemapindex":
		for _, entry := range doc.Sitemaps {
			children = append(children, entry.Loc)
		}
	}

	return pages, children
}

func (d *discoverer) crawl(ctx context.Context) []string {
	type queued struct {
		url   string
		depth int
	}

	start, _ := d.normalize(d.start.String())
	queue := []queued{{url: start, depth: 0}}
	seen := map[string]bool{start: true}

	var pages []string

	for fetches := 0; len(queue) > 0 && fetches < discoveryMaxCrawlPages && ctx.Err() == nil; fetches++ {
		current := queue[0]
		queue = queue[1:]

		body, contentType, err := d.fetch(ctx, current.url)
		if err != nil || !strings.Contains(contentType, "html") {
			continue
		}

		for _, href := range extractLinks(body) {
			link, ok := d.normalize(href)
			if !ok || seen[link] || !looksLikePage(link) {
				continue
			}

			seen[link] = true

			pages = append(pages, link)
			if current.depth+1 < discoveryCrawlDepth {
				queue = append(queue, queued{url: link, depth: current.depth + 1})
			}
		}
	}

	return pages
}

func extractLinks(body []byte) []string {
	var links []string

	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return links
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data != "a" {
				continue
			}

			for _, attr := range token.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
	}
}

// looksLikePage skips links to files a page scanner cannot load as a document.
func looksLikePage(link string) bool {
	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	// Cloudflare injects /cdn-cgi/ links (email obfuscation) into proxied pages.
	if strings.HasPrefix(parsed.Path, "/cdn-cgi/") {
		return false
	}

	switch strings.ToLower(path.Ext(parsed.Path)) {
	case "", ".html", ".htm", ".php", ".asp", ".aspx":
		return true
	}

	return false
}
