package orchestrator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestMarshalPayloadReturnsValidJSONOnMarshalError(t *testing.T) {
	auditPayload := marshalPayload(map[string]any{"bad": make(chan int)})

	var decoded map[string]string
	if err := json.Unmarshal([]byte(auditPayload), &decoded); err != nil {
		t.Fatalf("audit payload should remain valid JSON after marshal error: %v; payload=%s", err, auditPayload)
	}

	if decoded["marshal_error"] == "" {
		t.Fatalf("audit payload should include marshal_error: %s", auditPayload)
	}
}

func TestMarshalPayloadRedactsInlineAuthStorageState(t *testing.T) {
	payload := &events.JobCreatedPayload{
		JobID:     "job-redact",
		InputType: "urls",
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
			Auth:    []byte(`{"mode":"storage_state","content_b64":"secret-bytes"}`),
		},
	}

	auditPayload := marshalPayload(payload)
	if strings.Contains(auditPayload, "secret-bytes") || strings.Contains(auditPayload, "content_b64") {
		t.Fatalf("audit payload leaked inline auth bytes: %s", auditPayload)
	}

	if !strings.Contains(auditPayload, `"content_redacted":true`) {
		t.Fatalf("audit payload should mark redaction: %s", auditPayload)
	}
}
