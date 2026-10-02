package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
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
	discoveryMaxCandidates = 1000
	discoveryMaxWarnings   = 20
	discoveryUserAgent     = "StageFlow-Discovery/1.0 (+https://stageflow.org)"
	discoverySourceSitemap = "sitemap"
	discoverySourceCrawl   = "crawl"
)

var errDiscoveryBlockedAddress = errors.New("connection to a disallowed address was blocked")

type discoveryResult struct {
	URLs        []string `json:"urls"`
	Source      string   `json:"source"`
	Truncated   bool     `json:"truncated"`
	Sources     []string `json:"sources"`
	Complete    bool     `json:"complete"`
	StopReasons []string `json:"stopReasons"`
	Warnings    []string `json:"warnings"`
}

// The connect-time check remains mandatory: redirects and changed DNS records
// must never route discovery into the host's private network.
func newDiscoveryClient(mode targetValidationMode) *http.Client {
	dialer := &net.Dialer{Timeout: discoveryFetchTimeout, Control: func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}

		ip := net.ParseIP(host)
		if ip == nil || !isAllowedTargetIP(ip, mode) {
			return errDiscoveryBlockedAddress
		}

		return nil
	}}

	return &http.Client{Timeout: discoveryFetchTimeout, Transport: &http.Transport{
		Proxy: nil, DialContext: dialer.DialContext, TLSHandshakeTimeout: discoveryFetchTimeout,
		ResponseHeaderTimeout: discoveryFetchTimeout, DisableKeepAlives: true,
	}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > discoveryMaxRedirects {
			return errors.New("too many redirects")
		}

		if req.URL.User != nil || !sameDiscoveryOrigin(req.URL, via[0].URL) {
			return errors.New("redirect outside the discovery origin")
		}

		return nil
	}}
}

func sameDiscoveryOrigin(a, b *url.URL) bool {
	port := func(u *url.URL) string {
		if u.Port() != "" {
			return u.Port()
		}

		if u.Scheme == "https" {
			return "443"
		}

		return "80"
	}

	return (a.Scheme == "http" || a.Scheme == "https") && a.Scheme == b.Scheme &&
		strings.EqualFold(a.Hostname(), b.Hostname()) &&
		port(a) == port(b)
}

type discoveryCandidate struct {
	url    string
	depth  int
	source string
}
type discoverer struct {
	client   *http.Client
	start    *url.URL
	result   discoveryResult
	seen     map[string]bool
	queue    []discoveryCandidate
	robots   []robotsRule
	maxPages int
}

func discoverPages(ctx context.Context, client *http.Client, start *url.URL, maxPages int) discoveryResult {
	ctx, cancel := context.WithTimeout(ctx, discoveryBudget)
	defer cancel()

	start = cloneDiscoveryURL(start)

	start.Fragment = ""
	if start.Path == "" {
		start.Path = "/"
	}

	d := &discoverer{
		maxPages: maxPages,
		client:   client,
		start:    start,
		seen:     make(map[string]bool),
		result: discoveryResult{
			URLs:        []string{},
			Source:      discoverySourceCrawl,
			Sources:     []string{},
			Complete:    true,
			StopReasons: []string{},
			Warnings:    []string{},
		},
	}

	sitemaps, robotsOK := d.readRobots(ctx)
	if !robotsOK {
		return d.result
	}
	// Enforce robots, same-origin scope and query exclusions before every redirect.
	guardedClient := *client
	originalRedirect := client.CheckRedirect
	guardedClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if originalRedirect != nil {
			if err := originalRedirect(req, via); err != nil {
				return err
			}
		}

		if len(via) > discoveryMaxRedirects || req.URL.User != nil || !sameDiscoveryOrigin(req.URL, start) {
			return errors.New("redirect outside discovery scope")
		}

		if _, ok := d.normalize(req.URL.String(), start); !ok || !robotsAllow(d.robots, req.URL) {
			return errors.New("redirect excluded by discovery policy")
		}

		return nil
	}
	d.client = &guardedClient
	d.addCandidate(start.String(), start, 0, discoverySourceCrawl)
	d.fromSitemaps(ctx, sitemaps)
	d.crawl(ctx)

	if ctx.Err() != nil {
		d.stop("time_limit", "Discovery reached its time limit; the URL list is incomplete.")
	}

	if len(d.result.Sources) > 0 && containsDiscoveryString(d.result.Sources, discoverySourceSitemap) {
		d.result.Source = discoverySourceSitemap
	}

	return d.result
}
func cloneDiscoveryURL(u *url.URL) *url.URL { copyURL := *u; return &copyURL }
func containsDiscoveryString(values []string, value string) bool {
	for _, x := range values {
		if x == value {
			return true
		}
	}

	return false
}
func (d *discoverer) warn(message string) {
	if len(d.result.Warnings) < discoveryMaxWarnings && !containsDiscoveryString(d.result.Warnings, message) {
		d.result.Warnings = append(d.result.Warnings, message)
	}
}
func (d *discoverer) stop(reason, message string) {
	d.result.Complete = false

	d.result.Truncated = true
	if !containsDiscoveryString(d.result.StopReasons, reason) {
		d.result.StopReasons = append(d.result.StopReasons, reason)
	}

	d.warn(message)
}
func (d *discoverer) normalize(raw string, base *url.URL) (string, bool) {
	parsed, err := base.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || !sameDiscoveryOrigin(parsed, d.start) {
		return "", false
	}

	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	if len(parsed.String()) > maxURLLength {
		return "", false
	}

	if (parsed.RawQuery != "" || parsed.ForceQuery) && parsed.String() != d.start.String() {
		return "", false
	}
	// Canonicalize the origin only; paths and query values remain case sensitive.
	parsed.Scheme, parsed.Host = d.start.Scheme, d.start.Host

	return parsed.String(), true
}
func (d *discoverer) addCandidate(raw string, base *url.URL, depth int, source string) {
	normalized, ok := d.normalize(raw, base)
	if !ok || !looksLikePage(normalized) || d.seen[normalized] {
		return
	}

	if len(d.seen) >= discoveryMaxCandidates {
		d.stop("candidate_limit", "Discovery reached its candidate limit.")
		return
	}

	d.seen[normalized] = true

	parsed, err := url.Parse(normalized)
	if err != nil {
		return
	}

	if !robotsAllow(d.robots, parsed) {
		d.warn("Some pages were excluded by robots.txt.")
		return
	}

	d.queue = append(d.queue, discoveryCandidate{url: normalized, depth: depth, source: source})
}
func (d *discoverer) fetch(ctx context.Context, target string) ([]byte, string, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, "", nil, err
	}

	req.Header.Set("User-Agent", discoveryUserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, discoveryMaxBodyBytes+1))
	if err != nil {
		return nil, "", nil, err
	}

	if len(body) > discoveryMaxBodyBytes {
		return nil, "", nil, errors.New("response exceeds the 5 MiB body limit")
	}

	return body, resp.Header.Get("Content-Type"), resp.Request.URL, nil
}
func (d *discoverer) fromSitemaps(ctx context.Context, candidates []string) {
	candidates = append(candidates, d.start.ResolveReference(&url.URL{Path: "/sitemap.xml"}).String())
	seen := make(map[string]bool)

	for len(candidates) > 0 && ctx.Err() == nil {
		raw := candidates[0]
		candidates = candidates[1:]

		target, ok := d.normalize(raw, d.start)
		if !ok || seen[target] || !d.sitemapAllowed(target) {
			continue
		}

		if len(seen) >= discoveryMaxSitemaps {
			d.stop("sitemap_limit", "Discovery reached its sitemap limit.")
			break
		}

		seen[target] = true

		children := d.readSitemap(ctx, target)
		for _, child := range children {
			if len(candidates) >= discoveryMaxCandidates {
				d.stop("candidate_limit", "Discovery reached its candidate limit.")
				break
			}

			if !seen[child] {
				candidates = append(candidates, child)
			}
		}
	}
}

func (d *discoverer) sitemapAllowed(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil {
		return false
	}

	if !robotsAllow(d.robots, parsed) {
		d.warn("Some sitemaps were excluded by robots.txt.")
		return false
	}

	return true
}

func (d *discoverer) readSitemap(ctx context.Context, target string) []string {
	body, _, finalURL, err := d.fetch(ctx, target)
	if err != nil {
		if err.Error() != "HTTP 404" && err.Error() != "HTTP 410" {
			d.warn("A sitemap could not be read; HTML link discovery continued.")
		}

		return nil
	}

	var doc sitemapDocument
	if xml.Unmarshal(body, &doc) != nil {
		d.warn("A sitemap was not valid XML; HTML link discovery continued.")
		return nil
	}

	switch doc.XMLName.Local {
	case "urlset":
		d.addSource(discoverySourceSitemap)

		for _, entry := range doc.URLs {
			d.addCandidate(entry.Loc, finalURL, 0, discoverySourceSitemap)
		}
	case "sitemapindex":
		children := []string{}

		for _, child := range doc.Sitemaps {
			if candidate, ok := d.normalize(child.Loc, finalURL); ok {
				children = append(children, candidate)
			}

			if len(children) >= discoveryMaxCandidates {
				d.stop("candidate_limit", "Discovery reached its candidate limit.")
				break
			}
		}

		return children
	default:
		d.warn("A sitemap response was not a sitemap; HTML link discovery continued.")
	}

	return nil
}

func (d *discoverer) addSource(source string) {
	if !containsDiscoveryString(d.result.Sources, source) {
		d.result.Sources = append(d.result.Sources, source)
	}
}

type sitemapDocument struct {
	XMLName  xml.Name
	URLs     []sitemapLoc `xml:"url"`
	Sitemaps []sitemapLoc `xml:"sitemap"`
}
type sitemapLoc struct {
	Loc string `xml:"loc"`
}

func (d *discoverer) crawl(ctx context.Context) {
	accepted := make(map[string]bool)

	fetches := 0
	for len(d.queue) > 0 && ctx.Err() == nil {
		if fetches >= discoveryMaxCrawlPages {
			d.stop("page_limit", "Discovery reached its 30 HTML-page fetch limit.")
			break
		}

		current := d.queue[0]
		d.queue = d.queue[1:]

		if accepted[current.url] {
			continue
		}

		fetches++

		body, finalURL, ok := d.readHTMLPage(ctx, current.url)
		if !ok {
			continue
		}

		final := finalURL.String()
		if !accepted[final] {
			if len(d.result.URLs) >= d.maxPages {
				d.stop("url_limit", "Discovery reached its URL limit.")
				break
			}

			accepted[final] = true
			d.seen[final] = true
			d.result.URLs = append(d.result.URLs, final)
		}

		d.addSource(current.source)
		d.queueHTMLLinks(body, finalURL, current.depth)
	}
}

func (d *discoverer) readHTMLPage(ctx context.Context, target string) ([]byte, *url.URL, bool) {
	body, contentType, finalURL, err := d.fetch(ctx, target)
	if err != nil {
		d.stop("page_fetch_failed", "A page could not be fetched; discovery coverage is incomplete.")
		return nil, nil, false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
		d.warn("Non-HTML links were excluded.")
		return nil, nil, false
	}

	final, ok := d.normalize(finalURL.String(), finalURL)
	if !ok {
		d.stop("redirect_excluded", "A redirect destination was excluded from discovery.")
		return nil, nil, false
	}

	if !robotsAllow(d.robots, finalURL) {
		d.warn("Some redirect destinations were excluded by robots.txt.")
		return nil, nil, false
	}

	normalizedURL, err := url.Parse(final)
	if err != nil {
		return nil, nil, false
	}

	return body, normalizedURL, true
}

func (d *discoverer) queueHTMLLinks(body []byte, finalURL *url.URL, depth int) {
	links, baseHref, linksLimited := extractDiscoveryLinks(body)
	if linksLimited {
		d.stop("candidate_limit", "Discovery reached its candidate limit.")
	}

	base := finalURL
	if parsed, err := finalURL.Parse(
		baseHref,
	); baseHref != "" && err == nil && parsed.User == nil &&
		sameDiscoveryOrigin(parsed, d.start) {
		base = parsed
	}

	for _, href := range links {
		if depth >= discoveryCrawlDepth {
			d.checkDepthLimit(href, base)
			continue
		}

		d.addCandidate(href, base, depth+1, discoverySourceCrawl)
	}
}

func (d *discoverer) checkDepthLimit(href string, base *url.URL) {
	candidate, valid := d.normalize(href, base)
	if !valid || !looksLikePage(candidate) || d.seen[candidate] {
		return
	}

	parsed, err := url.Parse(candidate)
	if err == nil && robotsAllow(d.robots, parsed) {
		d.stop("depth_limit", "Discovery reached its link-depth limit.")
	}
}

func extractDiscoveryLinks(body []byte) ([]string, string, bool) {
	links := []string{}
	base := ""
	limited := false

	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return links, base, limited
		}

		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}

		token := tokenizer.Token()

		href, found := discoveryTokenHref(token)
		if !found {
			continue
		}

		if token.Data == "base" && base == "" {
			base = href
		}

		if token.Data != "a" {
			continue
		}

		if len(links) >= discoveryMaxCandidates {
			limited = true
			continue
		}

		links = append(links, href)
	}
}

func discoveryTokenHref(token html.Token) (string, bool) {
	if token.Data != "a" && token.Data != "base" {
		return "", false
	}

	for _, attr := range token.Attr {
		if attr.Key == "href" {
			return attr.Val, true
		}
	}

	return "", false
}

func looksLikePage(link string) bool {
	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	if strings.HasPrefix(parsed.Path, "/cdn-cgi/") {
		return false
	}

	switch strings.ToLower(path.Ext(parsed.Path)) {
	case "", ".html", ".htm", ".php", ".asp", ".aspx":
		return true
	}

	return false
}

// Robots rules use the longest matching directive; Allow wins equal lengths.
// A specific StageFlow group takes precedence over wildcard groups.
type robotsRule struct {
	pattern     *regexp.Regexp
	specificity int
	allow       bool
}

func (d *discoverer) readRobots(ctx context.Context) ([]string, bool) {
	target := d.start.ResolveReference(&url.URL{Path: "/robots.txt"}).String()

	body, _, _, err := d.fetch(ctx, target)
	if err != nil {
		if err.Error() == "HTTP 404" || err.Error() == "HTTP 410" {
			return nil, true
		}

		d.stop("robots_unavailable", "robots.txt could not be checked; discovery did not fetch site pages.")

		return nil, false
	}

	rules, bodySitemaps, parseErr := parseDiscoveryRobots(body)
	if parseErr != nil {
		d.stop("robots_unavailable", "robots.txt exceeded parsing limits; discovery did not fetch site pages.")
		return nil, false
	}

	d.robots = rules

	return bodySitemaps, true
}

type discoveryRobotsGroup struct {
	agents []string
	rules  []robotsRule
}

type discoveryRobotsParser struct {
	groups    []discoveryRobotsGroup
	current   discoveryRobotsGroup
	hasRules  bool
	ruleCount int
	sitemaps  []string
}

func parseDiscoveryRobots(body []byte) ([]robotsRule, []string, error) {
	parser := discoveryRobotsParser{sitemaps: []string{}}
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(body, []byte("\ufeff"))))
	scanner.Buffer(make([]byte, 4096), discoveryMaxBodyBytes)

	for scanner.Scan() {
		if err := parser.readLine(scanner.Text()); err != nil {
			return nil, nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	parser.flush()

	return selectDiscoveryRobotsRules(parser.groups), parser.sitemaps, nil
}

func (p *discoveryRobotsParser) flush() {
	if len(p.current.agents) > 0 {
		p.groups = append(p.groups, p.current)
	}

	p.current = discoveryRobotsGroup{}
	p.hasRules = false
}

func (p *discoveryRobotsParser) readLine(line string) error {
	line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])

	key, value, found := strings.Cut(line, ":")
	if !found {
		return nil
	}

	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)

	switch key {
	case "sitemap":
		if len(p.sitemaps) < discoveryMaxCandidates {
			p.sitemaps = append(p.sitemaps, value)
		}
	case "user-agent":
		if len(p.groups) >= discoveryMaxCandidates || len(p.current.agents) >= discoveryMaxCandidates {
			return errors.New("too many robots groups")
		}

		if p.hasRules {
			p.flush()
		}

		p.current.agents = append(p.current.agents, strings.ToLower(value))
	case "allow", "disallow":
		p.hasRules = true
		if len(value) > maxURLLength || p.ruleCount >= discoveryMaxCandidates {
			return errors.New("robots directives exceed limits")
		}

		if value == "" || !strings.HasPrefix(value, "/") {
			return nil
		}

		rule, err := compileDiscoveryRobotsRule(value, key == "allow")
		if err != nil {
			return err
		}

		p.ruleCount++
		p.current.rules = append(p.current.rules, rule)
	}

	return nil
}

func compileDiscoveryRobotsRule(value string, allow bool) (robotsRule, error) {
	end := strings.HasSuffix(value, "$")
	raw := normalizeRobotsOctets(strings.TrimSuffix(value, "$"))

	pieces := strings.Split(raw, "*")
	for i := range pieces {
		pieces[i] = regexp.QuoteMeta(pieces[i])
	}

	expression := "^" + strings.Join(pieces, ".*")
	if end {
		expression += "$"
	}

	pattern, err := regexp.Compile(expression)
	if err != nil {
		return robotsRule{}, err
	}

	return robotsRule{pattern: pattern, specificity: len(strings.ReplaceAll(raw, "*", "")), allow: allow}, nil
}

func selectDiscoveryRobotsRules(groups []discoveryRobotsGroup) []robotsRule {
	best := -1
	selected := []robotsRule{}

	for _, group := range groups {
		specificity := discoveryRobotsGroupSpecificity(group.agents)
		if specificity < 0 || specificity < best {
			continue
		}

		if specificity > best {
			selected = nil
			best = specificity
		}

		selected = append(selected, group.rules...)
	}

	return selected
}

func discoveryRobotsGroupSpecificity(agents []string) int {
	specificity := -1
	productToken := "stageflow-discovery"

	for _, agent := range agents {
		if agent == "*" && specificity < 0 {
			specificity = 0
		}

		if agent != "" && agent != "*" && strings.Contains(productToken, agent) && len(agent) > specificity {
			specificity = len(agent)
		}
	}

	return specificity
}

func robotsAllow(rules []robotsRule, target *url.URL) bool {
	value := target.EscapedPath()
	if value == "" {
		value = "/"
	}

	if target.RawQuery != "" {
		value += "?" + target.RawQuery
	}

	value = normalizeRobotsOctets(value)

	best := -1
	allowed := true

	for _, rule := range rules {
		if rule.pattern.MatchString(value) && rule.specificity >= best {
			if rule.specificity > best || rule.allow {
				best = rule.specificity
				allowed = rule.allow
			}
		}
	}

	return allowed
}

// Compare the same URI octets when a link percent-encodes an unreserved
// character, or a robots directive contains a literal UTF-8 character.
func normalizeRobotsOctets(value string) string {
	const hex = "0123456789ABCDEF"

	var out strings.Builder

	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '%' && i+2 < len(value) {
			decoded, err := url.PathUnescape(value[i : i+3])
			if err == nil {
				ch = decoded[0]
				if robotsUnreserved(ch) {
					out.WriteByte(ch)
				} else {
					out.WriteByte('%')
					out.WriteByte(hex[ch>>4])
					out.WriteByte(hex[ch&15])
				}

				i += 2

				continue
			}
		}

		if ch >= 128 {
			out.WriteByte('%')
			out.WriteByte(hex[ch>>4])
			out.WriteByte(hex[ch&15])

			continue
		}

		out.WriteByte(ch)
	}

	return out.String()
}

func robotsUnreserved(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
		strings.ContainsRune("-._~", rune(ch))
}
