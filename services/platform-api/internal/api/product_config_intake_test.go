package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/tests/statusfixture/events"
)

func TestProductConfigURLAndZIPIntake(t *testing.T) {
	verification := &models.VerificationConfig{
		SourceJobID: "source-job",
		PageID:      "page-v2-a",
		RuleID:      "color-contrast",
		Scanner:     "axe",
		SubjectID:   "occ-v2-a",
	}
	for _, inputType := range []string{"urls", "zip"} {
		for _, tc := range []productIntakeCase{
			{"default production", "", "", nil, []string{"axe"}, http.StatusCreated},
			{"explicit production", "production", "path", nil, []string{"axe"}, http.StatusCreated},
			{"explicit staging", "staging", "origin", nil, []string{"axe"}, http.StatusCreated},
			{"staging targeted verification", "staging", "origin", verification, []string{"axe"}, http.StatusCreated},
			{"unknown profile", "preview", "", nil, []string{"axe"}, http.StatusBadRequest},
			{"unknown identity scope", "staging", "host", nil, []string{"axe"}, http.StatusBadRequest},
			{"verification scanner mismatch", "staging", "origin", verification, []string{"seo"}, http.StatusBadRequest},
			{"verification multiple scanners", "staging", "origin", verification, []string{"axe", "seo"}, http.StatusBadRequest},
			{"verification missing source", "staging", "origin", &models.VerificationConfig{PageID: "page-v2-a", RuleID: "color-contrast", Scanner: "axe"}, []string{"axe"}, http.StatusBadRequest},
		} {
			t.Run(inputType+"/"+tc.name, func(t *testing.T) {
				runProductIntakeCase(t, inputType, tc)
			})
		}
	}
}

type productIntakeCase struct {
	name, profile, scope string
	verification         *models.VerificationConfig
	modules              []string
	status               int
}

func productURLRequest(t *testing.T, tc productIntakeCase) *http.Request {
	t.Helper()

	body, err := json.Marshal(
		map[string]any{
			"urls":           []string{"https://example.com/a"},
			"modules":        tc.modules,
			"profile":        tc.profile,
			"identity_scope": tc.scope,
			"verification":   tc.verification,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return req
}

func productZIPRequest(t *testing.T, tc productIntakeCase) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "profile", tc.profile)
	writeField(t, writer, "identity_scope", tc.scope)
	writeField(t, writer, "modules", strings.Join(tc.modules, ","))

	if tc.verification != nil {
		encoded, err := json.Marshal(tc.verification)
		if err != nil {
			t.Fatal(err)
		}

		writeField(t, writer, "verification", string(encoded))
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func assertProductIntakeConfig(t *testing.T, config models.JobConfig, tc productIntakeCase) {
	t.Helper()

	if config.Profile != tc.profile || config.IdentityScope != tc.scope {
		t.Fatalf("product config lost: %+v", config)
	}

	if tc.verification != nil && (config.Verification == nil || *config.Verification != *tc.verification) {
		t.Fatal("verification identity lost")
	}
}

func runProductIntakeCase(t *testing.T, inputType string, tc productIntakeCase) {
	t.Helper()
	server, _, publisher := newTestServer(t)

	req := productURLRequest(t, tc)
	if inputType == "zip" {
		req = productZIPRequest(t, tc)
	}

	rec := httptest.NewRecorder()
	if inputType == "urls" {
		server.handleJobURLSubmit(rec, req)
	} else {
		server.handleJobZipUpload(rec, req)
	}

	if rec.Code != tc.status {
		t.Fatalf("status=%d want %d: %s", rec.Code, tc.status, rec.Body.String())
	}

	if tc.status != http.StatusCreated {
		if len(publisher.envelopes) != 0 {
			t.Fatal("invalid product config launched a job")
		}

		return
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("published=%d", len(publisher.envelopes))
	}

	payload, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok {
		t.Fatalf("payload=%T", publisher.envelopes[0].Payload)
	}

	assertProductIntakeConfig(t, payload.Config, tc)
}
