// ZIP upload request handling -- multipart parsing, filename sanitization, and
// scanner-config fields. Exercises handlers_jobs_zip_upload.go.

package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
)

func TestZipUploadWithScannerConfigsField(t *testing.T) {
	server, store, publisher := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")
	writeField(t, writer, "scanner_configs", `{"axe":{"rules":["color-contrast"]}}`)
	writeField(t, writer, "highlight_style", "solid")
	writeField(t, writer, "browser", "webkit")
	writeField(t, writer, "screenshot", "true")
	writeField(t, writer, "unknown_field", "ignored")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(store.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(store.uploads))
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(publisher.envelopes))
	}

	payload, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok {
		t.Fatal("expected JobCreatedPayload")
	}

	if payload.Config.HighlightStyle != "solid" {
		t.Fatalf("expected highlight_style=solid, got %q", payload.Config.HighlightStyle)
	}

	if payload.Config.Browser != "webkit" {
		t.Fatalf("expected browser=webkit, got %q", payload.Config.Browser)
	}

	if !payload.Config.Screenshot {
		t.Fatal("expected screenshot=true")
	}

	if payload.Config.ScannerConfigs["axe"] == nil {
		t.Fatal("expected scanner_configs.axe to be set")
	}

	if !payload.Config.AllowPrivateTargets {
		t.Fatal("zip jobs must set allow_private_targets: scanners target the pod-internal loopback server")
	}
}

func TestZipUploadInvalidScannerConfigsJSON(t *testing.T) {
	server, _, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")
	writeField(t, writer, "scanner_configs", "not-valid-json")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestZipUploadMissingFile(t *testing.T) {
	server, _, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writeField(t, writer, "modules", "axe")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestZipUploadNonZipFile(t *testing.T) {
	server, _, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.tar.gz", []byte("not a zip"))
	writeField(t, writer, "modules", "axe")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobZipUploadMethodNotAllowed(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/zip", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- handleJobStatus success + method not allowed ---

func TestZipUploadSanitizesUploadedFilename(t *testing.T) {
	server, objectStore, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "../../../etc/passwd.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(objectStore.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(objectStore.uploads))
	}

	var uploadedPath string
	for key := range objectStore.uploads {
		uploadedPath = key
	}

	if strings.Contains(uploadedPath, "../") {
		t.Fatalf("uploaded object path must not contain path traversal segments: %q", uploadedPath)
	}

	if !strings.HasSuffix(uploadedPath, "/passwd.zip") {
		t.Fatalf("uploaded object path = %q, want suffix %q", uploadedPath, "/passwd.zip")
	}
}

func TestZipUploadDefaultsScreenshotCaptureOn(t *testing.T) {
	server, _, publisher := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.envelopes))
	}

	published, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok {
		t.Fatalf("published payload = %T, want *events.JobCreatedPayload", publisher.envelopes[0].Payload)
	}

	if !published.Config.Screenshot {
		t.Fatal("expected omitted multipart screenshot field to default to true")
	}
}
