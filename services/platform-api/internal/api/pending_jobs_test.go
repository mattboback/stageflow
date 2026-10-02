package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestURLIntakeStatusExistsBeforeEventDelivery(t *testing.T) {
	server, _, publisher := newTestServer(t)
	publisher.snapshots = nil // Publishing has not delivered anything to the worker.
	publisher.onPublish = func(envelope *events.Envelope) error {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+envelope.JobID, http.NoBody)
		response := httptest.NewRecorder()
		server.handleJobStatus(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("immediate status before delivery: %d %s", response.Code, response.Body.String())
		}

		var status models.JobStatus
		if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}

		if status.State != models.JobStatePending {
			t.Fatalf("seed state %s", status.State)
		}

		return nil
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs/urls",
		bytes.NewBufferString(`{"urls":["https://example.com"]}`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	server.handleJobURLSubmit(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("submit: %d %s", response.Code, response.Body.String())
	}
}

func TestZIPIntakeStatusExistsBeforeEventDelivery(t *testing.T) {
	server, _, publisher := newTestServer(t)
	publisher.snapshots = nil
	publisher.onPublish = func(envelope *events.Envelope) error {
		record, err := server.statusReader.GetJob(context.Background(), envelope.JobID)
		if err != nil || record.State != models.JobStatePending || record.InputType != "zip" {
			t.Fatalf("ZIP status before delivery: %#v %v", record, err)
		}

		return nil
	}

	request := &zipJobRequest{
		jobID:   "00000000-0000-4000-8000-000000000123",
		zipPath: "00000000-0000-4000-8000-000000000123/input.zip",
		modules: []string{"axe"},
	}
	if err := server.enqueueZipJob(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}
