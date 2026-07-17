package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestZipUploadCleansStagedFileWhenLaterFieldIsInvalid(t *testing.T) {
	t.Parallel()

	server, objectStore, publisher := newTestServer(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "scanner_configs", "not-json")

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	if len(publisher.envelopes) != 0 {
		t.Fatalf("published envelopes = %d, want 0", len(publisher.envelopes))
	}

	if len(objectStore.deletes) != 1 {
		t.Fatalf("staged deletes = %d, want 1", len(objectStore.deletes))
	}

	if len(objectStore.uploads) != 0 {
		t.Fatalf("orphaned uploads = %d, want 0", len(objectStore.uploads))
	}
}

func TestZipUploadCleansFirstFileWhenDuplicateFollows(t *testing.T) {
	t.Parallel()

	server, objectStore, publisher := newTestServer(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "first.zip", buildTestZip(t))
	addZipFile(t, writer, "second.zip", buildTestZip(t))

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	if len(publisher.envelopes) != 0 {
		t.Fatalf("published envelopes = %d, want 0", len(publisher.envelopes))
	}

	if len(objectStore.deletes) != 1 || len(objectStore.uploads) != 0 {
		t.Fatalf("cleanup mismatch: deletes=%d uploads=%d", len(objectStore.deletes), len(objectStore.uploads))
	}
}
