// ZIP upload request handling -- multipart parsing, filename sanitization, and
// scanner-config fields. Exercises handlers_jobs_zip_upload.go.

package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestZipUploadRejectsNonChromiumBrowser(t *testing.T) {
	server, _, jobs := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "browser", "firefox")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "browser") {
		t.Fatalf("expected a browser validation error, got %s", rr.Body.String())
	}

	if len(jobs.admitted) != 0 {
		t.Fatalf("expected no job to be admitted, got %d", len(jobs.admitted))
	}
}

func TestZipUploadWithScannerConfigsField(t *testing.T) {
	server, store, jobs := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")

	const optionsJSON = `{"disabledRules":["color-contrast"],"runOnlyTags":["wcag2a"],"dynamicContentWaitMs":500,"rules":["legacy"]}`
	writeField(t, writer, "scanner_configs", `{"axe":`+optionsJSON+`}`)
	writeField(t, writer, "browser", "chromium")
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

	if len(jobs.admitted) != 1 {
		t.Fatalf("expected 1 admitted job, got %d", len(jobs.admitted))
	}

	payload := jobs.admitted[0]

	if !payload.Config.Screenshot {
		t.Fatal("expected screenshot=true")
	}

	var want map[string]any
	if err := json.Unmarshal([]byte(optionsJSON), &want); err != nil {
		t.Fatal(err)
	}

	if got := payload.Config.ScannerConfigs["axe"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("scanner options changed at ZIP intake: got %#v, want %#v", got, want)
	}

	if payload.Config.AllowPrivateTargets {
		t.Fatal("zip jobs must not set allow_private_targets: uploaded pages could then load internal addresses")
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

	if !strings.HasSuffix(uploadedPath, "/input.zip") {
		t.Fatalf("uploaded object path = %q, want suffix %q", uploadedPath, "/input.zip")
	}
}

func TestZipUploadDefaultsScreenshotCaptureOn(t *testing.T) {
	server, _, jobs := newTestServer(t)

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

	if len(jobs.admitted) != 1 {
		t.Fatalf("expected 1 admitted job, got %d", len(jobs.admitted))
	}

	admitted := jobs.admitted[0]

	if !admitted.Config.Screenshot {
		t.Fatal("expected omitted multipart screenshot field to default to true")
	}
}
