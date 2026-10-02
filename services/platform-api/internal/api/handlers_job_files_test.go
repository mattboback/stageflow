package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestJobFileRequiresLiveJobAndAllowlistedPath(t *testing.T) {
	server, files, _ := newTestServer(t)
	if err := files.UploadFile(
		context.Background(),
		artifacts.BucketArtifacts,
		"job/report.json",
		strings.NewReader(`{"ok":true}`),
		11,
	); err != nil {
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
