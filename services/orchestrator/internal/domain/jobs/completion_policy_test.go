package jobs

import (
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestDecideScanFailureCompletion(t *testing.T) {
	t.Parallel()

	t.Run("waits when more scanners are still running", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{
			ScannerResults: map[string]*models.ScannerResult{
				"axe": {Success: false},
			},
		}

		got := DecideScanFailureCompletion(job, false)
		if got != ScanFailureWait {
			t.Fatalf("DecideScanFailureCompletion() = %s, want %s", got, ScanFailureWait)
		}
	})

	t.Run("completes when at least one scanner succeeded", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{
			ScannerResults: map[string]*models.ScannerResult{
				"axe":        {Success: true},
				"lighthouse": {Success: false},
			},
		}

		got := DecideScanFailureCompletion(job, true)
		if got != ScanFailureCompleteWithPartialResults {
			t.Fatalf("DecideScanFailureCompletion() = %s, want %s", got, ScanFailureCompleteWithPartialResults)
		}
	})

	t.Run("fails when all scanners failed", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{
			ScannerResults: map[string]*models.ScannerResult{
				"axe":        {Success: false},
				"lighthouse": {Success: false},
			},
		}

		got := DecideScanFailureCompletion(job, true)
		if got != ScanFailureFailJob {
			t.Fatalf("DecideScanFailureCompletion() = %s, want %s", got, ScanFailureFailJob)
		}
	})
}

func TestSelectPrimaryScanner(t *testing.T) {
	t.Parallel()

	t.Run("prefers first successful expected scanner", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{ExpectedScanners: []string{"lighthouse", "axe"}}

		got, ok := SelectPrimaryScanner(job, []string{"axe", "lighthouse"})
		if !ok {
			t.Fatal("expected primary scanner")
		}

		if got != "lighthouse" {
			t.Fatalf("SelectPrimaryScanner() = %q, want %q", got, "lighthouse")
		}
	})

	t.Run("falls back to alphabetical order", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{ExpectedScanners: []string{"seo"}}

		got, ok := SelectPrimaryScanner(job, []string{"pa11y", "axe"})
		if !ok {
			t.Fatal("expected primary scanner")
		}

		if got != "axe" {
			t.Fatalf("SelectPrimaryScanner() = %q, want %q", got, "axe")
		}
	})

	t.Run("sorts successful scanners in place when falling back to alphabetical order", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{ExpectedScanners: []string{"seo"}}
		successfulScanners := []string{"pa11y", "axe"}

		got, ok := SelectPrimaryScanner(job, successfulScanners)
		if !ok {
			t.Fatal("expected primary scanner")
		}

		if got != "axe" {
			t.Fatalf("SelectPrimaryScanner() = %q, want %q", got, "axe")
		}

		if successfulScanners[0] != "axe" || successfulScanners[1] != "pa11y" {
			t.Fatalf("SelectPrimaryScanner() left successfulScanners = %v, want %v",
				successfulScanners, []string{"axe", "pa11y"})
		}
	})
}
