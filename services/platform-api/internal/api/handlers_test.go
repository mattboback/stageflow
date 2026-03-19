package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/sse"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
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
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitEmpty(t *testing.T) {
	server, _, _, _ := newTestServer(t)

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
	server, _, _, _ := newTestServer(t)

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
	server, _, _, _ := newTestServer(t)

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
	server, _, _, _ := newTestServer(t)

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
	server, _, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["http://127.0.0.1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	if len(publisher.envelopes) != 0 {
		t.Fatalf("expected no job.created published, got %d", len(publisher.envelopes))
	}
}

func TestHandleJobURLSubmitPrivateTargetFlagRequiresServerOptIn(t *testing.T) {
	server, _, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["http://127.0.0.1"],"allow_private_targets":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	if len(publisher.envelopes) != 0 {
		t.Fatalf("expected no job.created published, got %d", len(publisher.envelopes))
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

func TestHandleJobURLSubmitPrivateTargetSucceedsWhenBothOptInsEnabled(t *testing.T) {
	server, _, _, publisher := newTestServer(t)
	server.config.AllowPrivateTargets = true

	body := bytes.NewBufferString(`{"urls":["http://127.0.0.1"],"allow_private_targets":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 job.created published, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected envelope payload to be *events.JobCreatedPayload")
	}

	if len(created.URLs) != 1 || created.URLs[0] != "http://127.0.0.1" {
		t.Fatalf("expected local target to be preserved in payload, got %#v", created.URLs)
	}
}

func TestHandleJobURLSubmitAiNavigatorRequiresScannerConfig(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["https://example.com"],"modules":["ai-navigator"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var parsed httputil.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if parsed.Error.Field != "scanner_configs.ai-navigator" {
		t.Fatalf("expected field scanner_configs.ai-navigator, got %q", parsed.Error.Field)
	}
}

func TestHandleJobURLSubmitAiNavigatorAcceptsScannerConfig(t *testing.T) {
	server, _, _, publisher := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"ai-navigator"},
		"scanner_configs": map[string]any{
			"ai-navigator": map[string]any{
				"goal": map[string]any{"objective": "Reach checkout"},
				"vision": map[string]any{
					"model": "openai/gpt-4o-mini",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 job.created published, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected envelope payload to be *events.JobCreatedPayload")
	}

	cfg := created.Config.ScannerConfigs["ai-navigator"]
	if cfg == nil {
		t.Fatalf("expected scanner_configs.ai-navigator to be set")
	}

	goal, ok := cfg["goal"].(map[string]any)
	if !ok {
		t.Fatalf("expected goal to be an object")
	}

	if goal["objective"] != "Reach checkout" {
		t.Fatalf("expected goal.objective %q, got %v", "Reach checkout", goal["objective"])
	}
}

func TestHandleJobURLSubmitAiNavigatorUsesDefaultModel(t *testing.T) {
	t.Setenv("AI_NAVIGATOR_DEFAULT_MODEL", "openai/gpt-4o-mini")

	server, _, _, publisher := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"ai-navigator"},
		"scanner_configs": map[string]any{
			"ai-navigator": map[string]any{
				"goal": map[string]any{"objective": "Reach checkout"},
				"vision": map[string]any{
					"maxTokens": 250,
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 job.created published, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected envelope payload to be *events.JobCreatedPayload")
	}

	cfg := created.Config.ScannerConfigs["ai-navigator"]
	if cfg == nil {
		t.Fatalf("expected scanner_configs.ai-navigator to be set")
	}

	vision, ok := cfg["vision"].(map[string]any)
	if !ok {
		t.Fatalf("expected vision to be an object")
	}

	if vision["model"] != "openai/gpt-4o-mini" {
		t.Fatalf("expected vision.model %q, got %v", "openai/gpt-4o-mini", vision["model"])
	}
}

func TestZipUploadToArtifactFlow(t *testing.T) {
	server, storage, store, publisher := newTestServer(t)

	jobID := uploadZipAndGetJobID(t, server, buildTestZip(t), scannerTypeAxe)

	assertUploadCount(t, storage, 1)

	completeJobWithArtifacts(t, store, jobID, events.ArtifactLocations{
		ReportJSON: jobID + "/report.json",
		ReportHTML: jobID + "/report.html",
	})

	job := assertJobStatusDone(t, server, jobID)
	if job.Artifacts == nil || job.Artifacts.ReportHTML == "" {
		t.Fatalf("expected artifacts populated")
	}

	assertJobReportRedirect(t, server, jobID)
	assertJobResultsRedirect(t, server, jobID)

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected job.created published")
	}
}

func TestHandleJobStatusReturnsStructuredScreenshotArtifacts(t *testing.T) {
	server, objectStore, store, _ := newTestServer(t)

	jobID := "job-structured-screenshots"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	reportKey := jobID + "/report.json"
	results := buildStructuredScreenshotReport(jobID, time.Now().UTC())

	reportBytes, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	uploadKey := fmt.Sprintf("%s::%s", storage.BucketArtifacts, reportKey)
	objectStore.uploads[uploadKey] = reportBytes

	if completeErr := store.HandleJobCompleted(context.Background(), &events.JobCompletedPayload{
		JobID: jobID,
		Artifacts: events.ArtifactLocations{
			ReportJSON: reportKey,
			ReportHTML: jobID + "/report.html",
		},
	}); completeErr != nil {
		t.Fatalf("complete job: %v", completeErr)
	}

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
					{ElementId: strPtr("issue-abc-el-0"), ArtifactIds: []string{"ss-issue-abc"}},
					{ElementId: strPtr("issue-abc-el-1"), ArtifactIds: []string{"ss-issue-abc"}},
				},
			},
		},
		Artifacts: []report.ReportArtifact{
			{
				Id:   "ss-issue-abc",
				Type: "screenshot",
				Path: strPtr("screenshots/issue-abc.webp"),
			},
		},
		Errors: []report.ReportError{},
	}
}

func assertStructuredScreenshots(t *testing.T, screenshots []models.ScreenshotArtifact) {
	t.Helper()

	if len(screenshots) != 3 {
		t.Fatalf("expected 3 screenshots (2 violation + 1 overview), got %d", len(screenshots))
	}

	var (
		hasPrimary   bool
		hasSecondary bool
		hasOverview  bool
	)

	for _, shot := range screenshots {
		switch {
		case shot.Kind == models.ScreenshotKindViolation && shot.IssueID == "issue-abc":
			assertPrimaryScreenshot(t, shot)

			hasPrimary = true
		case shot.Kind == models.ScreenshotKindViolation && shot.IssueID == "issue-abc--occ-2":
			assertSecondaryScreenshot(t, shot)

			hasSecondary = true
		case shot.Kind == models.ScreenshotKindPageOverview:
			assertOverviewScreenshot(t, shot)

			hasOverview = true
		}
	}

	if !hasPrimary || !hasSecondary || !hasOverview {
		t.Fatalf(
			"missing expected screenshots: primary=%v secondary=%v overview=%v",
			hasPrimary,
			hasSecondary,
			hasOverview,
		)
	}
}

func assertPrimaryScreenshot(t *testing.T, shot models.ScreenshotArtifact) {
	t.Helper()

	if shot.ScannerID != "axe" || shot.PageID != "page-1" || shot.ArtifactID != "ss-issue-abc" {
		t.Fatalf("unexpected primary screenshot payload: %+v", shot)
	}

	if shot.OccurrenceIndex == nil || *shot.OccurrenceIndex != 0 {
		t.Fatalf("expected primary occurrence_index=0, got %+v", shot.OccurrenceIndex)
	}
}

func assertSecondaryScreenshot(t *testing.T, shot models.ScreenshotArtifact) {
	t.Helper()

	if shot.OccurrenceIndex == nil || *shot.OccurrenceIndex != 1 {
		t.Fatalf("expected secondary occurrence_index=1, got %+v", shot.OccurrenceIndex)
	}
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

func assertUploadCount(t *testing.T, storage *fakeStorage, expected int) {
	t.Helper()

	if len(storage.uploads) != expected {
		t.Fatalf("expected %d upload, got %d", expected, len(storage.uploads))
	}
}

func completeJobWithArtifacts(t *testing.T, store *status.Store, jobID string, artifacts events.ArtifactLocations) {
	t.Helper()

	payload := &events.JobCompletedPayload{
		JobID:     jobID,
		Artifacts: artifacts,
	}
	if err := store.HandleJobCompleted(context.Background(), payload); err != nil {
		t.Fatalf("complete job: %v", err)
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

func assertJobReportRedirect(t *testing.T, server *Server, jobID string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", rr.Code)
	}
}

func assertJobResultsRedirect(t *testing.T, server *Server, jobID string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", rr.Code)
	}
}

func TestJobReportFallbackWhenHTMLMissing(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-report-fallback"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	// Mark the job complete with an aggregated JSON report, but no HTML report artifact.
	if err := store.HandleJobCompleted(context.Background(), &events.JobCompletedPayload{
		JobID: jobID,
		Artifacts: events.ArtifactLocations{
			ReportJSON: jobID + "/report.json",
			ReportHTML: "",
		},
	}); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct == "" || !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content-type, got %q", ct)
	}

	if body := rr.Body.String(); !strings.Contains(body, "/api/v1/jobs/"+jobID+"/results") {
		t.Fatalf("expected results link in fallback HTML")
	}
}

func TestZipUploadAiNavigatorRequiresScannerConfig(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	zipBytes := buildTestZip(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "site.zip")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, writeErr := part.Write(zipBytes); writeErr != nil {
		t.Fatalf("write file: %v", writeErr)
	}

	if fieldErr := writer.WriteField("modules", "ai-navigator"); fieldErr != nil {
		t.Fatalf("failed to write field: %v", fieldErr)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("failed to close writer: %v", closeErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var parsed httputil.ErrorResponse
	if decodeErr := json.NewDecoder(rr.Body).Decode(&parsed); decodeErr != nil {
		t.Fatalf("decode error response: %v", decodeErr)
	}

	if parsed.Error.Field != "scanner_configs.ai-navigator" {
		t.Fatalf("expected field scanner_configs.ai-navigator, got %q", parsed.Error.Field)
	}
}

func TestZipUploadBodyTooLarge(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	rr := httptest.NewRecorder()
	handled := server.handleZipParseError(context.Background(), rr, &http.MaxBytesError{Limit: maxUploadSize})

	if !handled {
		t.Fatalf("expected oversized body error to be handled")
	}

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

// Helpers

type fakeStorage struct {
	uploads map[string][]byte
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{uploads: make(map[string][]byte)}
}

func (f *fakeStorage) UploadFile(_ context.Context, bucket, path string, reader io.Reader, _ int64) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s::%s", bucket, path)
	f.uploads[key] = data

	return nil
}

func (f *fakeStorage) GetPresignedURL(_ context.Context, bucket, path string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://storage.local/%s/%s?exp=%d", bucket, path, int(expiry.Seconds())), nil
}

func (f *fakeStorage) DownloadFile(_ context.Context, bucket, path string) (io.ReadCloser, error) {
	key := fmt.Sprintf("%s::%s", bucket, path)
	if data, ok := f.uploads[key]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	now := time.Now().UTC()
	results := report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       path,
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

func (f *fakeStorage) DeleteFile(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeStorage) FileExists(_ context.Context, bucket, path string) (bool, error) {
	key := fmt.Sprintf("%s::%s", bucket, path)
	_, ok := f.uploads[key]

	return ok, nil
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

type fakePublisher struct {
	envelopes []*events.Envelope
}

func (f *fakePublisher) PublishJobCreated(_ context.Context, envelope *events.Envelope) error {
	f.envelopes = append(f.envelopes, envelope)

	return nil
}

func newTestServer(t *testing.T) (*Server, *fakeStorage, *status.Store, *fakePublisher) {
	t.Helper()
	dir := t.TempDir()

	store, err := status.NewStore(&status.Config{Path: filepath.Join(dir, "status.db")})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	})

	storage := newFakeStorage()
	publisher := &fakePublisher{}

	registry, err := scannerregistry.InitializeRegistry(scannerregistry.DefaultConfig())
	if err != nil {
		t.Fatalf("init scanner registry: %v", err)
	}

	server := &Server{
		config: &ServerConfig{
			Storage:         storage,
			Publisher:       publisher,
			StatusReader:    store,
			ScannerRegistry: registry,
		},
		statusReader:    store,
		pendingJobs:     newPendingJobCache(),
		sseHub:          sse.NewHub(),
		scannerRegistry: registry,
		ipResolver:      defaultSecurityTestResolver(t),
	}

	return server, storage, store, publisher
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
