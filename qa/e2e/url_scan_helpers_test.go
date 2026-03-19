package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func fetchEnabledModules(t *testing.T) []string {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiBaseURL+"/scanners", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create scanners request: %v", err)
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to fetch scanners: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Scanners endpoint returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Scanners []struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		} `json:"scanners"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		t.Fatalf("Failed to decode scanners response: %v", decodeErr)
	}

	var enabled []string

	for _, s := range result.Scanners {
		if s.Enabled {
			enabled = append(enabled, s.ID)
		}
	}

	if len(enabled) == 0 {
		t.Fatal("No enabled scanners found")
	}

	t.Logf("Enabled scanners: %v", enabled)

	return enabled
}

func submitURLScan(t *testing.T, modules []string) string {
	t.Helper()

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": modules,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal URL scan payload: %v", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		apiBaseURL+"/jobs/urls",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to create URL scan request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to submit URL scan: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("Failed to read URL scan error body: %v", readErr)
		}

		t.Fatalf("URL scan submission failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		JobID string `json:"job_id"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		t.Fatalf("Failed to decode URL scan response: %v", decodeErr)
	}

	if result.JobID == "" {
		t.Fatal("URL scan response missing job_id")
	}

	return result.JobID
}

func waitForURLJobCompletion(t *testing.T, jobID string) {
	t.Helper()

	timeout := time.NewTimer(6 * time.Minute)
	defer timeout.Stop()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	client := httpClient()
	start := time.Now()
	lastState := ""

	for {
		select {
		case <-timeout.C:
			t.Fatalf("Timed out waiting for URL scan job completion after %s", time.Since(start).Truncate(time.Second))
		case <-ticker.C:
			status, err := fetchJobStatus(client, jobID)
			if err != nil {
				t.Logf("Failed to get job status: %v", err)

				continue
			}

			if status.State != lastState {
				t.Logf("Job state: %s (elapsed: %s)", status.State, time.Since(start).Truncate(time.Second))
				lastState = status.State
			}

			if isDoneState(status.State) {
				return
			}

			if isFailedState(status.State) {
				t.Fatalf("URL scan job failed: %s", status.Error)
			}
		}
	}
}

func fetchAndParseReport(t *testing.T, jobID string) report.UnifiedReportV2 {
	t.Helper()

	reportBody := fetchReportBody(t, jobID)

	var r report.UnifiedReportV2
	if unmarshalErr := json.Unmarshal(reportBody, &r); unmarshalErr != nil {
		// Log a snippet of the body for debugging.
		snippet := string(reportBody)
		if len(snippet) > 500 {
			snippet = snippet[:500] + "..."
		}

		t.Fatalf("Failed to unmarshal report into UnifiedReportV2: %v\nBody snippet: %s", unmarshalErr, snippet)
	}

	return r
}

func fetchReportBody(t *testing.T, jobID string) []byte {
	t.Helper()

	endpoint := fmt.Sprintf("%s/jobs/%s/results", apiBaseURL, jobID)

	resp := doGet(t, newNoRedirectClient(), endpoint)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return mustReadAll(t, resp.Body, "report body")
	}

	if resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if location == "" {
			t.Fatal("Results returned redirect but no Location header")
		}

		return fetchReportBodyFromRedirect(t, location)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Fatalf("Results endpoint returned status %d: %s", resp.StatusCode, string(bodyBytes))

	return nil
}

func newNoRedirectClient() *http.Client {
	// Use a client that stops on redirects so we can rewrite the MinIO host.
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func doGet(t *testing.T, client *http.Client, endpoint string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to fetch %s: %v", endpoint, err)
	}

	return resp
}

func fetchReportBodyFromRedirect(t *testing.T, location string) []byte {
	t.Helper()

	rewrittenLocation, redirectHost := rewriteMinIOHost(location)
	t.Logf("Following redirect to: %s", rewrittenLocation)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rewrittenLocation, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create redirect request: %v", err)
	}

	if redirectHost != "" {
		req.Host = redirectHost
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to follow redirect: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Redirect returned status %d", resp.StatusCode)
	}

	return mustReadAll(t, resp.Body, "redirected report body")
}

func rewriteMinIOHost(location string) (rewritten string, hostOverride string) {
	const (
		minioHost = "minio:9000"
		localHost = "localhost:9000"
	)

	if !strings.Contains(location, minioHost) {
		return location, ""
	}

	return strings.Replace(location, minioHost, localHost, 1), minioHost
}

func mustReadAll(t *testing.T, r io.Reader, context string) []byte {
	t.Helper()

	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", context, err)
	}

	return body
}

func validateReportStructure(t *testing.T, r report.UnifiedReportV2, jobID string, modules []string) {
	t.Helper()

	if !strings.HasPrefix(r.Version, "2.") {
		t.Errorf("Expected version 2.x.x, got %s", r.Version)
	}

	if r.Meta.JobId != jobID {
		t.Errorf("Expected meta.jobId=%s, got %s", jobID, r.Meta.JobId)
	}

	if len(r.Scanners) != len(modules) {
		t.Errorf("Expected %d scanners, got %d", len(modules), len(r.Scanners))
	}

	if len(r.Pages) < 1 {
		t.Error("Expected at least 1 page in report")
	}

	if r.Issues == nil {
		t.Error("Issues slice should not be nil")
	}
}

func validateScannerResult(t *testing.T, r report.UnifiedReportV2, scannerID string) {
	t.Helper()

	scanner := findScanner(r, scannerID)
	if scanner == nil {
		t.Fatalf("Scanner %q not found in report", scannerID)
	}

	validateScannerStatus(t, scanner, scannerID)
	logScannerMetrics(t, scanner, scannerID)
	validateScannerSpecifics(t, r, scannerID)
}

func findScanner(r report.UnifiedReportV2, scannerID string) *report.ScannerSummary {
	for i := range r.Scanners {
		if r.Scanners[i].Id == scannerID {
			return &r.Scanners[i]
		}
	}

	return nil
}

func validateScannerStatus(t *testing.T, scanner *report.ScannerSummary, scannerID string) {
	t.Helper()

	if scanner.Status != report.ScannerStatusSuccess {
		t.Errorf("Scanner %q status=%s, expected success", scannerID, scanner.Status)

		if scanner.Error != nil {
			t.Logf("Scanner %q error: %s", scannerID, *scanner.Error)
		}
	}
}

func logScannerMetrics(t *testing.T, scanner *report.ScannerSummary, scannerID string) {
	t.Helper()

	if scanner.DurationMs != nil {
		t.Logf("Scanner %q duration: %.0fms", scannerID, *scanner.DurationMs)
	}

	if scanner.IssueCount != nil {
		t.Logf("Scanner %q issues: %d", scannerID, *scanner.IssueCount)
	}
}

func validateScannerSpecifics(t *testing.T, r report.UnifiedReportV2, scannerID string) {
	t.Helper()

	switch scannerID {
	case "lighthouse":
		validateLighthouseResults(t, r)
	case "axe":
		validateAxeResults(t, r)
	}
}

func validateLighthouseResults(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	if len(r.Summary.LighthouseCategories) == 0 {
		t.Error("Lighthouse scanner present but no lighthouseCategories in summary")
	}

	for _, cat := range r.Summary.LighthouseCategories {
		t.Logf("Lighthouse category %s: score=%.2f", cat.Id, cat.AvgScore)
	}
}

func validateAxeResults(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	axeIssues := 0

	for _, issue := range r.Issues {
		if issue.Scanner == "axe" {
			axeIssues++

			if len(issue.WcagTags) == 0 {
				t.Logf("Axe issue %q has no WCAG tags (may be best-practice only)", issue.RuleId)
			}
		}
	}

	t.Logf("Axe issues found: %d", axeIssues)
}

func validateSummaryConsistency(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	validateSummaryTotals(t, r)
	validateSummarySeveritySums(t, r)
	validateSummaryByScannerCounts(t, r)
	validateSummaryIssuePageReferences(t, r)
	validateSummaryPageIssueCounts(t, r)
}

func validateSummaryTotals(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	// totalIssues == len(issues)
	if r.Summary.TotalIssues != len(r.Issues) {
		t.Errorf("summary.totalIssues=%d but len(issues)=%d", r.Summary.TotalIssues, len(r.Issues))
	}

	// pagesScanned == len(pages)
	if r.Summary.PagesScanned != len(r.Pages) {
		t.Errorf("summary.pagesScanned=%d but len(pages)=%d", r.Summary.PagesScanned, len(r.Pages))
	}

	// pagesWithIssues <= pagesScanned
	if r.Summary.PagesWithIssues > r.Summary.PagesScanned {
		t.Errorf("pagesWithIssues=%d > pagesScanned=%d", r.Summary.PagesWithIssues, r.Summary.PagesScanned)
	}
}

func validateSummarySeveritySums(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	// Severity sums add up.
	sev := r.Summary.BySeverity

	infoCount := 0
	if sev.Info != nil {
		infoCount = *sev.Info
	}

	severitySum := sev.Critical + sev.Serious + sev.Moderate + sev.Minor + infoCount
	if severitySum != r.Summary.TotalIssues {
		t.Errorf("severity sum=%d (critical=%d serious=%d moderate=%d minor=%d info=%d) != totalIssues=%d",
			severitySum, sev.Critical, sev.Serious, sev.Moderate, sev.Minor, infoCount, r.Summary.TotalIssues)
	}
}

func validateSummaryByScannerCounts(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	// byScanner counts: sum must be >= totalIssues (cross-scanner dedup can
	// cause per-scanner discovery counts to exceed deduplicated total).
	if r.Summary.ByScanner == nil {
		return
	}

	byScannerSum := 0

	for id, count := range r.Summary.ByScanner {
		t.Logf("byScanner[%s]=%d", id, count)
		byScannerSum += count
	}

	if byScannerSum < r.Summary.TotalIssues {
		t.Errorf("byScanner sum=%d < totalIssues=%d (should be >=)", byScannerSum, r.Summary.TotalIssues)
		return
	}

	if byScannerSum > r.Summary.TotalIssues {
		t.Logf(
			"byScanner sum=%d > totalIssues=%d (cross-scanner dedup detected)",
			byScannerSum,
			r.Summary.TotalIssues,
		)
	}
}

func validateSummaryIssuePageReferences(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	// Verify all issues reference valid page IDs.
	pageIDs := make(map[string]bool, len(r.Pages))
	for _, p := range r.Pages {
		pageIDs[p.Id] = true
	}

	for _, issue := range r.Issues {
		if !pageIDs[issue.PageId] {
			t.Errorf("Issue %q references unknown pageId=%q", issue.Id, issue.PageId)
		}
	}
}

func validateSummaryPageIssueCounts(t *testing.T, r report.UnifiedReportV2) {
	t.Helper()

	// Page-level issue counts should sum to totalIssues.
	pageIssueSum := 0
	for _, p := range r.Pages {
		pageIssueSum += p.IssueCount
	}

	if pageIssueSum < r.Summary.TotalIssues {
		t.Errorf("page issueCount sum=%d < totalIssues=%d", pageIssueSum, r.Summary.TotalIssues)
		return
	}

	if pageIssueSum > r.Summary.TotalIssues {
		t.Logf("page issueCount sum=%d > totalIssues=%d (page counts not yet recomputed after dedup)",
			pageIssueSum, r.Summary.TotalIssues)
	}
}

func logPerformanceSummary(t *testing.T, r report.UnifiedReportV2, wallClock time.Duration) {
	t.Helper()

	t.Logf("PERF_SUMMARY total_wall_clock_ms=%d", wallClock.Milliseconds())

	if r.Meta.DurationMs != nil {
		t.Logf("PERF_SUMMARY report_duration_ms=%.0f", *r.Meta.DurationMs)
	}

	for _, s := range r.Scanners {
		durationStr := "unknown"
		if s.DurationMs != nil {
			durationStr = fmt.Sprintf("%.0f", *s.DurationMs)
		}

		t.Logf("PERF_SUMMARY scanner=%s status=%s duration_ms=%s", s.Id, s.Status, durationStr)
	}
}
