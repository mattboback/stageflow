package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
)

func postCreateProject(t *testing.T, server *Server, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleCreateProject(rr, req)

	return rr
}

func TestHandleCreateProjectRejectsPrivateURLByDefault(t *testing.T) {
	server, _, _ := newTestServer(t)

	rr := postCreateProject(t, server, `{"slug":"local-project","urls":["http://127.0.0.1:3010"]}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateProjectAcceptsPrivateURLWhenInstanceAllows(t *testing.T) {
	server, _, _ := newTestServer(t)
	server.config.AllowPrivateTargets = true

	rr := postCreateProject(t, server, `{"slug":"local-project","urls":["http://127.0.0.1:3010"]}`)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleProjectScanCarriesInstancePrivateTargetOptInToJob(t *testing.T) {
	server, _, publisher := newTestServer(t)
	server.config.AllowPrivateTargets = true

	rr := postCreateProject(t, server, `{"slug":"local-project","urls":["http://127.0.0.1:3010"]}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/local-project/scan", http.NoBody)
	scanRR := httptest.NewRecorder()
	server.handleProjectScan(scanRR, req, "local-project")

	if scanRR.Code != http.StatusCreated {
		t.Fatalf("project scan: expected 201, got %d: %s", scanRR.Code, scanRR.Body.String())
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 job.created published, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected envelope payload to be *events.JobCreatedPayload")
	}

	if !created.Config.AllowPrivateTargets {
		t.Fatalf("expected job config to carry allow_private_targets from instance setting")
	}
}
