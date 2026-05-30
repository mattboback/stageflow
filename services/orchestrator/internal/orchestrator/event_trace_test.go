package orchestrator

import (
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

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
