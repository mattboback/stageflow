// Shared helpers for the api package's handler tests. They were previously
// defined in handlers_coverage_test.go while being used from
// baseline_storage_test.go and handlers_jobs_zip_upload_regression_test.go too,
// so their home did not reflect their scope.

package api

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

func applyTestSignal(t *testing.T, server *Server, signal jobstatus.Signal) {
	t.Helper()

	if _, err := server.jobStatus.Apply(context.Background(), signal); err != nil {
		t.Fatalf("apply signal: %v", err)
	}
}

// --- handleListScanners ---

func writeField(t *testing.T, w *multipart.Writer, key, value string) {
	t.Helper()

	if err := w.WriteField(key, value); err != nil {
		t.Fatalf("write field %s: %v", key, err)
	}
}

func addZipFile(t *testing.T, w *multipart.Writer, filename string, data []byte) {
	t.Helper()

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, writeErr := part.Write(data); writeErr != nil {
		t.Fatalf("write file: %v", writeErr)
	}
}
