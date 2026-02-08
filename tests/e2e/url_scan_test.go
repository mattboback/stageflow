package e2e

import (
	"testing"
	"time"
)

func TestE2E_URLScan(t *testing.T) {
	requireE2E(t)
	waitForAPI(t)

	modules := fetchEnabledModules(t)

	start := time.Now()

	jobID := submitURLScan(t, modules)
	t.Logf("URL scan job submitted: %s", jobID)

	waitForURLJobCompletion(t, jobID)

	r := fetchAndParseReport(t, jobID)
	t.Logf("Report parsed successfully (version=%s, scanners=%d, pages=%d, issues=%d)",
		r.Version, len(r.Scanners), len(r.Pages), len(r.Issues))

	wallClock := time.Since(start)

	t.Run("report_structure", func(t *testing.T) {
		validateReportStructure(t, r, jobID, modules)
	})

	t.Run("summary_consistency", func(t *testing.T) {
		validateSummaryConsistency(t, r)
	})

	for _, scannerID := range modules {
		t.Run("scanner/"+scannerID, func(t *testing.T) {
			validateScannerResult(t, r, scannerID)
		})
	}

	logPerformanceSummary(t, r, wallClock)
}
