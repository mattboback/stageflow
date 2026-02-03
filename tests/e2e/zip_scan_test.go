package e2e

import (
	"os"
	"testing"
)

func TestE2E_ZipScan(t *testing.T) {
	requireE2E(t)

	waitForAPI(t)

	zipPath := createTestZip(t)

	jobID := uploadZip(t, zipPath)
	t.Logf("Job submitted: %s", jobID)

	waitForJobCompletion(t, jobID)
	verifyResults(t, jobID)
}

func requireE2E(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	if os.Getenv("RUN_E2E") == "" {
		t.Skip("skipping E2E test; RUN_E2E not set")
	}
}
