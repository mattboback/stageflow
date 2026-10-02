package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/internal/archive"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestHandleHealth(t *testing.T) {
	server := &Server{
		config: &ServerConfig{},
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestHandleJobStatusNotFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitEmpty(t *testing.T) {
	server, _, _ := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitTooManyURLs(t *testing.T) {
	server, _, _ := newTestServer(t)

	urls := make([]string, maxURLCount+1)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/%d", i)
	}

	body, err := json.Marshal(map[string]any{"urls": urls})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitURLTooLong(t *testing.T) {
	server, _, _ := newTestServer(t)

	longURL := strings.Repeat("a", maxURLLength+1)

	body, err := json.Marshal(map[string]any{"urls": []string{longURL}})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitBodyTooLarge(t *testing.T) {
	server, _, _ := newTestServer(t)

	payload := strings.Repeat("a", maxURLSubmitBodySize)
	body := fmt.Sprintf(`{"urls":["https://example.com/%s"]}`, payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitPrivateTargetWithoutOptInKeepsDefaultBlocking(t *testing.T) {
	server, _, jobs := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["http://127.0.0.1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	if len(jobs.admitted) != 0 {
		t.Fatalf("expected no job admitted, got %d", len(jobs.admitted))
	}
}

func TestHandleJobURLSubmitPrivateTargetFlagRequiresServerOptIn(t *testing.T) {
	server, _, jobs := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["http://127.0.0.1"],"allow_private_targets":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	if len(jobs.admitted) != 0 {
		t.Fatalf("expected no job admitted, got %d", len(jobs.admitted))
	}

	var parsed httputil.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if parsed.Error.Code != httputil.ErrCodeValidationFailed {
		t.Fatalf("expected validation error code, got %q", parsed.Error.Code)
	}

	if parsed.Error.Field != "allow_private_targets" {
		t.Fatalf("expected field allow_private_targets, got %q", parsed.Error.Field)
	}
}

func TestHandleJobURLSubmitPrivateTargetSucceedsForTokenCallerWhenServerAllows(t *testing.T) {
	server, _, jobs := newTestServer(t)
	server.config.AllowPrivateTargets = true
	server.config.APIToken = "caller-token"

	body := bytes.NewBufferString(`{"urls":["http://127.0.0.1"],"allow_private_targets":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "caller-token")

	rr := httptest.NewRecorder()

	server.identifyAPIKeyCaller(server.handleJobURLSubmit)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	if len(jobs.admitted) != 1 {
		t.Fatalf("expected 1 job admitted, got %d", len(jobs.admitted))
	}

	created := jobs.admitted[0]

	if len(created.URLs) != 1 || created.URLs[0] != "http://127.0.0.1" {
		t.Fatalf("expected local target to be preserved in payload, got %#v", created.URLs)
	}
}

func TestHandleJobURLSubmitWritesStorageStateBeforeAdmission(t *testing.T) {
	server, storage, jobs := newTestServer(t)

	storageState := `{"cookies":[],"origins":[]}`
	encoded := base64.StdEncoding.EncodeToString([]byte(storageState))

	jobs.onAdmit = func(job *models.Job) error {
		expectedKey := job.ID + "/auth.json"

		uploaded := storage.uploads[expectedKey]
		if string(uploaded) != storageState {
			t.Fatalf("auth state was not written before admission; got %q, want %q", uploaded, storageState)
		}

		return nil
	}

	body, err := json.Marshal(map[string]any{
		"urls": []string{"https://example.com"},
		"auth": map[string]any{
			"mode": "storage_state",
			"storage_state": map[string]any{
				"content_b64": encoded,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(jobs.admitted) != 1 {
		t.Fatalf("expected 1 job admitted, got %d", len(jobs.admitted))
	}

	created := jobs.admitted[0]

	authWire := string(created.Config.Auth)
	if strings.Contains(authWire, "content_b64") || strings.Contains(authWire, encoded) {
		t.Fatalf("admitted auth leaked storage-state content: %s", authWire)
	}

	expectedKey := created.ID + "/auth.json"

	if authWire != `{"mode":"storage_state"}` {
		t.Fatalf("dispatch leaked session metadata: %s", authWire)
	}

	uploaded := storage.uploads[expectedKey]
	if string(uploaded) != storageState {
		t.Fatalf("uploaded storage state = %q, want %q", uploaded, storageState)
	}
}

func TestHandleJobURLSubmitKeepsAuthOnUncertainAdmission(t *testing.T) {
	server, storage, jobs := newTestServer(t)
	jobs.err = errors.New("database down")

	storageState := `{"cookies":[],"origins":[]}`
	encoded := base64.StdEncoding.EncodeToString([]byte(storageState))

	body, err := json.Marshal(map[string]any{
		"urls": []string{"https://example.com"},
		"auth": map[string]any{
			"mode": "storage_state",
			"storage_state": map[string]any{
				"content_b64": encoded,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(storage.deletes) != 0 || len(storage.uploads) != 1 {
		t.Fatalf("uncertain admission lost auth: deletes=%d uploads=%d", len(storage.deletes), len(storage.uploads))
	}
}

func TestHandleJobZipUploadKeepsInputOnUncertainAdmission(t *testing.T) {
	server, storage, jobs := newTestServer(t)
	jobs.err = errors.New("database down")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "site.zip")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, writeErr := part.Write(buildTestZip(t)); writeErr != nil {
		t.Fatalf("write file: %v", writeErr)
	}

	if fieldErr := writer.WriteField("modules", scannerTypeAxe); fieldErr != nil {
		t.Fatalf("write modules field: %v", fieldErr)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("close writer: %v", closeErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(storage.deletes) != 0 || len(storage.uploads) != 1 {
		t.Fatalf("uncertain admission lost ZIP: deletes=%d uploads=%d", len(storage.deletes), len(storage.uploads))
	}
}

func TestZipUploadToArtifactFlow(t *testing.T) {
	server, storage, jobs := newTestServer(t)

	jobID := uploadZipAndGetJobID(t, server, buildTestZip(t), scannerTypeAxe)

	assertUploadCount(t, storage, 1)

	testJobs(server).finish(t, jobID, jobID+"/report.json")

	job := assertJobStatusDone(t, server, jobID)
	if job.Artifacts == nil || job.Artifacts.ReportJSON == "" {
		t.Fatalf("expected artifacts populated")
	}

	assertJobResultsRedirect(t, server, jobID)

	if len(jobs.admitted) != 1 {
		t.Fatalf("expected the job to be admitted")
	}
}

func TestHandleJobStatusReturnsStructuredScreenshotArtifacts(t *testing.T) {
	server, objectStore, _ := newTestServer(t)

	jobID := "job-structured-screenshots"
	testJobs(server).put(scanningURLJob(jobID))

	reportKey := jobID + "/report.json"
	results := buildStructuredScreenshotReport(jobID, time.Now().UTC())

	reportBytes, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	objectStore.uploads[reportKey] = reportBytes

	testJobs(server).finish(t, jobID, reportKey)

	job := assertJobStatusDone(t, server, jobID)
	if job.Artifacts == nil {
		t.Fatalf("expected artifacts in job status")
	}

	assertStructuredScreenshots(t, job.Artifacts.Screenshots)
}

func buildStructuredScreenshotReport(jobID string, now time.Time) report.UnifiedReportV2 {
	infoCount := 0

	return report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       jobID,
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 1,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  1,
				Moderate: 0,
				Minor:    0,
				Info:     &infoCount,
			},
			ByScanner:       map[string]int{"axe": 1},
			PagesScanned:    1,
			PagesWithIssues: 1,
		},
		Scanners: []report.ScannerSummary{
			{Id: "axe", Name: strPtr("Accessibility"), Status: report.ScannerStatusSuccess},
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "http://example.com",
				IssueCount: 1,
				DurationMs: 15,
				PageOverview: &report.PageOverview{
					ScreenshotFilename: "page-overview-page-1.webp",
					PageWidth:          1280,
					PageHeight:         720,
					Elements:           []report.PageOverviewElement{},
				},
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "issue-abc",
				Scanner:      "axe",
				RuleId:       "color-contrast",
				Severity:     report.IssueSeveritySerious,
				Title:        "Contrast issue",
				Description:  "Insufficient color contrast",
				PageId:       "page-1",
				PageUrl:      "http://example.com",
				ElementCount: 2,
				Occurrences: []report.IssueOccurrence{
					{ElementId: strPtr("issue-abc-el-0")},
					{ElementId: strPtr("issue-abc-el-1")},
				},
			},
		},
		Artifacts: []report.ReportArtifact{},
		Errors:    []report.ReportError{},
	}
}

func assertStructuredScreenshots(t *testing.T, screenshots []models.ScreenshotArtifact) {
	t.Helper()

	if len(screenshots) != 1 {
		t.Fatalf("expected 1 page overview screenshot, got %d", len(screenshots))
	}

	if screenshots[0].Kind != models.ScreenshotKindPageOverview {
		t.Fatalf("expected a page overview screenshot, got kind %q", screenshots[0].Kind)
	}

	assertOverviewScreenshot(t, screenshots[0])
}

func assertOverviewScreenshot(t *testing.T, shot models.ScreenshotArtifact) {
	t.Helper()

	if shot.ScannerID != "axe" || shot.PageID != "page-1" {
		t.Fatalf("unexpected overview screenshot payload: %+v", shot)
	}

	if shot.ArtifactID != "page-overview:axe:page-1" {
		t.Fatalf("unexpected overview artifact_id: %q", shot.ArtifactID)
	}
}

func uploadZipAndGetJobID(t *testing.T, server *Server, zipBytes []byte, modules string) string {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "site.zip")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, writeErr := part.Write(zipBytes); writeErr != nil {
		t.Fatalf("write file: %v", writeErr)
	}

	if modules != "" {
		if fieldErr := writer.WriteField("modules", modules); fieldErr != nil {
			t.Fatalf("failed to write field: %v", fieldErr)
		}
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("failed to close writer: %v", closeErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var resp map[string]any
	if decodeErr := json.NewDecoder(rr.Body).Decode(&resp); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}

	jobIDValue, ok := resp["job_id"]
	if !ok {
		t.Fatalf("job_id missing")
	}

	jobID, ok := jobIDValue.(string)
	if !ok || jobID == "" {
		t.Fatalf("job_id should be a non-empty string")
	}

	return jobID
}

func assertUploadCount(t *testing.T, storage *fakeFiles, expected int) {
	t.Helper()

	if len(storage.uploads) != expected {
		t.Fatalf("expected %d upload, got %d", expected, len(storage.uploads))
	}
}

func assertJobStatusDone(t *testing.T, server *Server, jobID string) models.JobStatus {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID, http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var job models.JobStatus
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("decode job: %v", err)
	}

	if job.State != models.JobStateDone {
		t.Fatalf("expected DONE, got %s", job.State)
	}

	return job
}

func assertJobResultsRedirect(t *testing.T, server *Server, jobID string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestZipUploadBodyTooLarge(t *testing.T) {
	server, _, _ := newTestServer(t)

	rr := httptest.NewRecorder()
	handled := server.handleZipParseError(context.Background(), rr, &http.MaxBytesError{Limit: maxUploadSize})

	if !handled {
		t.Fatalf("expected oversized body error to be handled")
	}

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

// fakeFiles keeps job files in memory, keyed "<job ID>/<path>".
type fakeFiles struct {
	uploads map[string][]byte
	deletes []string
}

func newFakeFiles() *fakeFiles {
	return &fakeFiles{uploads: make(map[string][]byte)}
}

func (f *fakeFiles) Write(_ context.Context, key string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	f.uploads[key] = data

	return nil
}

func (f *fakeFiles) WriteInput(ctx context.Context, jobID string, reader io.Reader) error {
	return f.Write(ctx, jobID+"/input.zip", reader)
}

func (f *fakeFiles) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if data, ok := f.uploads[key]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	now := time.Now().UTC()
	results := report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       key,
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 0,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  0,
				Moderate: 0,
				Minor:    0,
			},
			ByScanner:       map[string]int{"axe": 0},
			PagesScanned:    1,
			PagesWithIssues: 0,
		},
		Scanners: []report.ScannerSummary{
			{Id: "axe", Name: strPtr("Accessibility"), Status: report.ScannerStatusSuccess},
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "http://example.com",
				IssueCount: 0,
				DurationMs: 0,
				StartedAt:  &now,
				FinishedAt: &now,
			},
		},
		Issues:    []report.IssueDetail{},
		Artifacts: []report.ReportArtifact{},
		Errors:    []report.ReportError{},
	}

	data, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	f.uploads[key] = data

	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeFiles) Remove(_ context.Context, key string) error {
	f.deletes = append(f.deletes, key)
	delete(f.uploads, key)

	return nil
}

func (f *fakeFiles) RemoveJob(_ context.Context, jobID string) error {
	for key := range f.uploads {
		if strings.HasPrefix(key, jobID+"/") {
			f.deletes = append(f.deletes, key)
			delete(f.uploads, key)
		}
	}

	return nil
}

func TestReadLimitedReportJSON(t *testing.T) {
	data, err := readLimitedReportJSON(strings.NewReader("12345"), 5)
	if err != nil {
		t.Fatalf("readLimitedReportJSON exact limit: %v", err)
	}

	if string(data) != "12345" {
		t.Fatalf("data = %q", data)
	}

	_, err = readLimitedReportJSON(strings.NewReader("123456"), 5)
	if err == nil {
		t.Fatal("expected over-limit error")
	}

	if !strings.Contains(err.Error(), "exceeds 5 byte limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func newTestServer(t *testing.T) (*Server, *fakeFiles, *fakeJobs) {
	t.Helper()

	storage := newFakeFiles()
	jobs := newFakeJobs()

	server := NewServer(&ServerConfig{
		StatusReader: jobs,
		Files:        storage,
		Admitter:     jobs,
		JobDeleter:   jobs,
		CheckDisk:    func(context.Context) error { return nil },
		ValidateZIP: func(_ context.Context, id string) error {
			data, exists := storage.uploads[id+"/input.zip"]
			if !exists {
				return errors.New("input ZIP missing")
			}

			return archive.Validate(bytes.NewReader(data), int64(len(data)))
		},
	})
	server.ipResolver = defaultSecurityTestResolver(t)

	return server, storage, jobs
}

func buildTestZip(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer

	zw := zip.NewWriter(&buf)

	w, err := zw.Create("index.html")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}

	content := []byte("<html><body>Hello</body></html>")
	if _, writeErr := w.Write(content); writeErr != nil {
		t.Fatalf("write zip entry: %v", writeErr)
	}

	if closeErr := zw.Close(); closeErr != nil {
		t.Fatalf("close zip: %v", closeErr)
	}

	return buf.Bytes()
}
