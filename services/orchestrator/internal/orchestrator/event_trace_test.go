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

	if !strings.Contains(auditPayload, `"auth":{"configured":true,"mode":"storage_state"}`) {
		t.Fatalf("audit payload should retain only safe auth metadata: %s", auditPayload)
	}
}

func TestMarshalPayloadRedactsLiteralFormAuthAndAIInputValues(t *testing.T) {
	const (
		userCanary     = "audit-user-canary-7d11"
		passwordCanary = "audit-password-canary-f913"
		aiInputCanary  = "audit-ai-input-canary-23ab"
	)

	payload := &events.JobCreatedPayload{
		JobID:     "job-redact-all",
		InputType: "urls",
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"example-scanner"},
			Auth: json.RawMessage(`{
				"mode":"form",
				"login_url":"https://example.com/login",
				"steps":[
					{"type":"fill","selector":"#user","value":"` + userCanary + `"},
					{"type":"fill","selector":"#password","value":"` + passwordCanary + `"}
				],
				"success":{"type":"load"}
			}`),
			ScannerConfigs: map[string]map[string]any{
				"example-scanner": {
					"goal": map[string]any{
						"objective":        "Submit the form with " + aiInputCanary,
						"successCriterion": "Welcome " + passwordCanary,
						"inputValues": map[string]any{
							"email": aiInputCanary,
						},
					},
				},
			},
		},
	}

	auditPayload := marshalPayload(payload)
	for _, canary := range []string{userCanary, passwordCanary, aiInputCanary} {
		if strings.Contains(auditPayload, canary) {
			t.Fatalf("audit payload leaked canary %q: %s", canary, auditPayload)
		}
	}

	if !strings.Contains(auditPayload, `"auth":{"configured":true,"mode":"form"}`) {
		t.Fatalf("audit payload should retain safe form metadata: %s", auditPayload)
	}

	if !strings.Contains(auditPayload, `"inputValues":{"email":"[REDACTED]"}`) {
		t.Fatalf("audit payload should preserve only AI input keys: %s", auditPayload)
	}
}

func TestMarshalPayloadSummarizesPagePreScanActions(t *testing.T) {
	const canary = "page-action-canary-f821"

	auditPayload := marshalPayload(map[string]any{
		"pages": []any{map[string]any{
			"url": "https://example.com",
			"pre_scan_actions": []any{map[string]any{
				"type":  "fill",
				"value": canary,
			}},
		}},
	})

	if strings.Contains(auditPayload, canary) {
		t.Fatalf("audit payload leaked page-action canary: %s", auditPayload)
	}

	if !strings.Contains(auditPayload, `"pre_scan_actions":{"configured":true,"redacted":true}`) {
		t.Fatalf("audit payload did not retain safe page-action metadata: %s", auditPayload)
	}
}

func TestMarshalPayloadRedactsProducerFailureText(t *testing.T) {
	const canary = "p%40ss+word%2B1"

	auditPayload := marshalPayload(&events.ScanFailedPayload{
		JobID:        "job-failure-audit",
		ScannerType:  "axe",
		Error:        "navigation failed at https://example.com/login?password=" + canary,
		ErrorDetails: "submitted value " + canary,
	})

	if strings.Contains(auditPayload, canary) {
		t.Fatalf("audit payload leaked producer failure text: %s", auditPayload)
	}

	if !strings.Contains(auditPayload, `"error":"[REDACTED]"`) ||
		!strings.Contains(auditPayload, `"error_details":"[REDACTED]"`) {
		t.Fatalf("audit payload did not redact producer failure fields: %s", auditPayload)
	}
}
