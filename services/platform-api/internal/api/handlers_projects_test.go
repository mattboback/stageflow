package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
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
