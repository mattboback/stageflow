package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestJobFileRequiresLiveJobAndAllowlistedPath(t *testing.T) {
	server, files, _ := newTestServer(t)
	if err := files.Write(context.Background(), "job/report.json", strings.NewReader(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}

	originalReader := server.statusReader
	server.statusReader = &staticJobReader{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job/files/report.json", http.NoBody)
	response := httptest.NewRecorder()
	server.handleJobStatus(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("orphan file returned %d", response.Code)
	}

	server.statusReader = originalReader
	seedReadableJob(t, server, "job", models.JobStateDone)

	response = httptest.NewRecorder()
	server.handleJobStatus(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"ok":true}` {
		t.Fatalf("public report: %d %s", response.Code, response.Body.String())
	}

	for _, path := range []string{"auth/storage-state.json", "input.zip", "site/logo.svg", "axe/progress.jsonl", "axe/outcome.json"} {
		response = httptest.NewRecorder()
		server.handleJobStatus(
			response,
			httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job/files/"+path, http.NoBody),
		)

		if response.Code != http.StatusNotFound {
			t.Fatalf("private path %s returned %d", path, response.Code)
		}
	}
}

func TestPublicArtifactURLServesOnlyThisJobsPublicFiles(t *testing.T) {
	t.Parallel()

	for key, want := range map[string]string{
		"job/report.json":                          "/api/v1/jobs/job/files/report.json",
		"job/axe/results.json":                     "/api/v1/jobs/job/files/axe/results.json",
		"job/axe/page-1/screenshots/overview.webp": "/api/v1/jobs/job/files/axe/page-1/screenshots/overview.webp",
	} {
		if got, err := publicArtifactURL("job", key); err != nil || got != want {
			t.Errorf("publicArtifactURL(%q) = %q, %v; want %q", key, got, err, want)
		}
	}

	for _, key := range []string{
		"",
		"report.json",
		"other/report.json",
		"job/auth.json",
		"job/../other/report.json",
		"job/axe/page-1/../../report.json",
		"job/axe/page-1//screenshots/overview.webp",
		"job/axe/page-1/./screenshots/overview.webp",
		"/job/report.json",
		"job//report.json",
	} {
		if got, err := publicArtifactURL("job", key); err == nil {
			t.Errorf("publicArtifactURL(%q) = %q, want an error", key, got)
		}
	}

	if got, err := publicArtifactURL("", "/report.json"); err == nil {
		t.Errorf("empty job ID produced %q", got)
	}
}
