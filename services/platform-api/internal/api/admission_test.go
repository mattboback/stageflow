package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
)

func TestQueueCapacityRejectsAllCallersAndCleansAuthFiles(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		t.Run(map[bool]string{false: "public", true: "token"}[trusted], func(t *testing.T) {
			server, files, store := newTestServer(t)
			server.config.APIToken = "trusted"
			store.err = jobstore.ErrQueueFull

			req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString(storageStateBody))
			if trusted {
				req.Header.Set("X-Api-Key", "trusted")
			}

			response := httptest.NewRecorder()
			server.Router().ServeHTTP(response, req)

			if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "30" {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}

			if len(store.admitted) != 0 || len(files.uploads) != 0 {
				t.Fatalf("rejected job left state: jobs=%d files=%d", len(store.admitted), len(files.uploads))
			}
		})
	}
}

func TestLowDiskRefusesBeforeUploadAndAgainBeforeAdmission(t *testing.T) {
	for _, rejectAt := range []int{1, 2} {
		t.Run(map[int]string{1: "before upload", 2: "before admission"}[rejectAt], func(t *testing.T) {
			server, files, store := newTestServer(t)
			calls := 0
			server.config.CheckDisk = func(context.Context) error {
				calls++
				if calls == rejectAt {
					return artifacts.ErrLowDisk
				}

				return nil
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString(storageStateBody))
			response := httptest.NewRecorder()
			server.handleJobURLSubmit(response, req)

			if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "30" {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}

			if len(store.admitted) != 0 || len(files.uploads) != 0 {
				t.Fatal("low-disk rejection admitted a job or retained credentials")
			}
		})
	}
}

func uploadArchiveRequest(t *testing.T, data []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "input.zip")
	if err != nil {
		t.Fatal(err)
	}

	if _, writeErr := part.Write(data); writeErr != nil {
		t.Fatal(writeErr)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

func TestInvalidAndBombArchivesFailBeforeAdmission(t *testing.T) {
	var compressed bytes.Buffer

	writer := zip.NewWriter(&compressed)

	part, err := writer.Create("large.txt")
	if err != nil {
		t.Fatal(err)
	}

	if _, copyErr := io.Copy(part, bytes.NewReader(bytes.Repeat([]byte("x"), 2*1024*1024))); copyErr != nil {
		t.Fatal(copyErr)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	for name, data := range map[string][]byte{"invalid": []byte("not a ZIP"), "expansion bomb": compressed.Bytes()} {
		t.Run(name, func(t *testing.T) {
			server, files, store := newTestServer(t)
			response := httptest.NewRecorder()
			server.handleJobZipUpload(response, uploadArchiveRequest(t, data))

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}

			if len(store.admitted) != 0 || len(files.uploads) != 0 {
				t.Fatal("invalid input was admitted or retained")
			}
		})
	}
}

func TestZipAdmissionCleansCertainRejectionAndPreservesUncertainFailures(t *testing.T) {
	for name, failure := range map[string]error{"full": jobstore.ErrQueueFull, "canceled": context.Canceled, "store unavailable": errors.New("database unavailable")} {
		t.Run(name, func(t *testing.T) {
			server, files, store := newTestServer(t)
			store.err = failure
			response := httptest.NewRecorder()
			server.handleJobZipUpload(response, uploadArchiveRequest(t, buildTestZip(t)))

			expectedFiles := 1
			if errors.Is(failure, jobstore.ErrQueueFull) {
				expectedFiles = 0
			}

			if response.Code == http.StatusCreated || len(store.admitted) != 0 || len(files.uploads) != expectedFiles {
				t.Fatalf(
					"unexpected intake state: status=%d jobs=%d files=%d",
					response.Code,
					len(store.admitted),
					len(files.uploads),
				)
			}
		})
	}
}

func TestAdmittedJobIsImmediatelyPendingWithSelectedScanners(t *testing.T) {
	server, _, store := newTestServer(t)
	response := httptest.NewRecorder()
	server.handleJobURLSubmit(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/jobs/urls",
			bytes.NewBufferString(`{"urls":["https://example.com"],"modules":["axe","seo"]}`),
		),
	)

	if response.Code != http.StatusCreated {
		t.Fatal(response.Body.String())
	}

	var submitted struct {
		ID string `json:"job_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &submitted); err != nil {
		t.Fatal(err)
	}

	snapshot, err := server.statusReader.GetJob(context.Background(), submitted.ID)
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.State != "PENDING" || len(snapshot.ExpectedScanners) != 2 || len(store.admitted) != 1 {
		t.Fatalf("job missing pending durable snapshot: %#v", snapshot)
	}
}

func TestOrphanPublicFileCannotBeServedWithoutJobRow(t *testing.T) {
	server, files, _ := newTestServer(t)
	files.uploads[artifacts.BucketArtifacts+"::missing/report.json"] = []byte(`{"orphan":true}`)
	response := httptest.NewRecorder()
	server.Router().
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/files/report.json", http.NoBody))

	if response.Code != http.StatusNotFound {
		t.Fatalf("orphan artifact visible: %d %s", response.Code, response.Body.String())
	}
}

func TestLowDiskZIPRequestDoesNotReadBodyOrCreateFiles(t *testing.T) {
	server, files, store := newTestServer(t)
	server.config.CheckDisk = func(context.Context) error { return artifacts.ErrLowDisk }
	request := uploadArchiveRequest(t, buildTestZip(t))
	response := httptest.NewRecorder()
	server.handleJobZipUpload(response, request)

	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "30" {
		t.Fatalf("status=%d", response.Code)
	}

	if len(files.uploads) != 0 || len(store.admitted) != 0 {
		t.Fatal("low-disk admission started upload")
	}
}

func TestDeleteCannotRemoveArtifactsWhileCleanupOwnsJob(t *testing.T) {
	server, files, store := newTestServer(t)
	seedReadableJob(t, server, "pending-cleanup", "DONE")

	store.deleteErr = jobstore.ErrCleanupPending
	files.uploads[artifacts.BucketArtifacts+"::pending-cleanup/report.json"] = []byte(`{"retained":true}`)
	response := httptest.NewRecorder()
	server.handleJobDelete(
		response,
		httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/pending-cleanup", http.NoBody),
	)

	if response.Code != http.StatusConflict || len(files.uploads) != 1 {
		t.Fatalf("cleanup-owned artifacts removed: %d files=%d", response.Code, len(files.uploads))
	}
}

func TestCancellationBeforeAdmissionCleansUploadedAuth(t *testing.T) {
	server, files, store := newTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checks := 0
	server.config.CheckDisk = func(context.Context) error {
		checks++
		if checks == 2 {
			if len(files.uploads) != 1 {
				t.Fatal("auth was not uploaded before final admission check")
			}

			cancel()
		}

		return nil
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString(storageStateBody)).
		WithContext(ctx)
	response := httptest.NewRecorder()
	server.handleJobURLSubmit(response, request)

	if response.Code != http.StatusRequestTimeout || len(store.admitted) != 0 || len(files.uploads) != 0 {
		t.Fatalf(
			"pre-admission cancellation retained state: status=%d jobs=%d files=%d",
			response.Code,
			len(store.admitted),
			len(files.uploads),
		)
	}
}
