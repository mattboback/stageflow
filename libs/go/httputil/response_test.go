package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "hello"}

	RespondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("expected message 'hello', got %q", result["message"])
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	RespondError(w, http.StatusBadRequest, "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var result ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error.Message != "invalid input" || result.Error.Code != "INVALID_REQUEST" {
		t.Errorf("expected INVALID_REQUEST %q, got %+v", "invalid input", result.Error)
	}
}

func TestRespondOK(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]int{"count": 42}

	RespondOK(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["count"] != 42 {
		t.Errorf("expected count 42, got %d", result["count"])
	}
}

func TestRespondCreated(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"id": "abc123"}

	RespondCreated(w, data)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["id"] != "abc123" {
		t.Errorf("expected id 'abc123', got %q", result["id"])
	}
}

func TestRespondNotFound(t *testing.T) {
	w := httptest.NewRecorder()

	RespondNotFound(w, "Job")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var result ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Error.Message != "Job not found" || result.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND %q, got %+v", "Job not found", result.Error)
	}
}
