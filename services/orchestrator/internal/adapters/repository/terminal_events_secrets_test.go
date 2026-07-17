package db

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

const (
	terminalUserCanary     = "terminal-user-canary-119a"
	terminalPasswordCanary = "terminal-password-canary-b4e3"
	terminalAICanary       = "terminal-ai-canary-73cd"
)

func sensitiveTerminalConfig(t *testing.T) models.JobConfig {
	t.Helper()

	auth := json.RawMessage(`{
		"mode":"form",
		"login_url":"https://example.com/login",
		"steps":[
			{"type":"fill","selector":"#user","value":"` + terminalUserCanary + `"},
			{"type":"fill","selector":"#password","value":"` + terminalPasswordCanary + `"}
		],
		"success":{"type":"load"}
	}`)

	return models.JobConfig{
		Modules: []string{"ai-navigator"},
		Auth:    auth,
		ScannerConfigs: map[string]map[string]any{
			"ai-navigator": {
				"goal": map[string]any{
					"objective": "Complete the flow using " + terminalAICanary,
					"inputValues": map[string]any{
						"email": terminalAICanary,
					},
				},
			},
		},
	}
}

func createSensitiveTerminalJob(t *testing.T, database *Database, id string, state models.JobState) {
	t.Helper()

	now := time.Now().UTC()
	if err := database.CreateJob(context.Background(), &models.Job{
		ID:        id,
		State:     state,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    sensitiveTerminalConfig(t),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
}

func assertTerminalJobConfigRedacted(t *testing.T, database *Database, jobID string) {
	t.Helper()

	job, err := database.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	serialized, err := json.Marshal(job.Config)
	if err != nil {
		t.Fatalf("Marshal(job.Config) error = %v", err)
	}

	for _, canary := range []string{
		terminalUserCanary,
		terminalPasswordCanary,
		terminalAICanary,
	} {
		if strings.Contains(string(serialized), canary) {
			t.Fatalf("terminal config leaked canary %q: %s", canary, serialized)
		}
	}

	if len(job.Config.Auth) != 0 {
		t.Fatalf("terminal config retained auth recipe: %s", job.Config.Auth)
	}

	goal, _ := job.Config.ScannerConfigs["ai-navigator"]["goal"].(map[string]any)

	inputValues, _ := goal["inputValues"].(map[string]any)
	if got := inputValues["email"]; got != redactedConfigValue {
		t.Fatalf("terminal AI input = %v, want %q", got, redactedConfigValue)
	}
}

func TestCompleteJobWithTerminalEventScrubsExecutionOnlyConfig(t *testing.T) {
	database := setupTestDB(t)
	createSensitiveTerminalJob(t, database, "job-complete-redacted", models.JobStateCompleting)

	err := database.CompleteJobWithTerminalEvent(
		context.Background(),
		"job-complete-redacted",
		&events.JobCompletedPayload{
			JobID:  "job-complete-redacted",
			Status: events.JobStatusSuccess,
			Artifacts: events.ArtifactLocations{
				ReportJSON: "job-complete-redacted/report.json",
				ReportHTML: "job-complete-redacted/report.html",
			},
		},
	)
	if err != nil {
		t.Fatalf("CompleteJobWithTerminalEvent() error = %v", err)
	}

	assertTerminalJobConfigRedacted(t, database, "job-complete-redacted")
}

func TestFailJobWithTerminalEventScrubsConfigAndFailureText(t *testing.T) {
	database := setupTestDB(t)
	createSensitiveTerminalJob(t, database, "job-failed-redacted", models.JobStateScanning)

	errorMessage := "browser failure while submitting " + terminalAICanary
	errorDetails := "credentials included " + terminalPasswordCanary
	payload := &events.JobFailedPayload{
		JobID:        "job-failed-redacted",
		Status:       events.JobStatusFailed,
		Stage:        events.JobFailStageScanning,
		Error:        errorMessage,
		ErrorDetails: errorDetails,
	}

	transitioned, err := database.FailJobWithTerminalEvent(
		context.Background(),
		"job-failed-redacted",
		events.JobFailStageScanning,
		errorMessage,
		errorDetails,
		payload,
	)
	if err != nil {
		t.Fatalf("FailJobWithTerminalEvent() error = %v", err)
	}

	if !transitioned {
		t.Fatal("first failure did not transition the job")
	}

	assertTerminalJobConfigRedacted(t, database, "job-failed-redacted")

	job, err := database.GetJob(context.Background(), "job-failed-redacted")
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if strings.Contains(job.Error+job.ErrorDetails, terminalAICanary) ||
		strings.Contains(job.Error+job.ErrorDetails, terminalPasswordCanary) {
		t.Fatalf("terminal failure text leaked a canary: error=%q details=%q", job.Error, job.ErrorDetails)
	}

	terminalEvents, err := database.ListUnpublishedTerminalEvents(
		context.Background(),
		"job-failed-redacted",
	)
	if err != nil {
		t.Fatalf("ListUnpublishedTerminalEvents() error = %v", err)
	}

	if len(terminalEvents) != 1 {
		t.Fatalf("terminal events = %d, want 1", len(terminalEvents))
	}

	if strings.Contains(terminalEvents[0].PayloadJSON, terminalAICanary) ||
		strings.Contains(terminalEvents[0].PayloadJSON, terminalPasswordCanary) {
		t.Fatalf("terminal outbox leaked a canary: %s", terminalEvents[0].PayloadJSON)
	}

	if strings.Contains(payload.Error+payload.ErrorDetails, terminalAICanary) ||
		strings.Contains(payload.Error+payload.ErrorDetails, terminalPasswordCanary) {
		t.Fatalf("publishable terminal payload leaked a canary: %+v", payload)
	}
}

func TestFailJobWithTerminalEventScrubsEncodedFailureText(t *testing.T) {
	database := setupTestDB(t)
	jobID := "job-failed-encoded-secret"

	const (
		secret            = "p@ss word+1"
		browserFormSecret = "a~b*c"
		browserFormValue  = "a%7Eb*c"
	)

	config := sensitiveTerminalConfig(t)
	config.ScannerConfigs["ai-navigator"]["goal"] = map[string]any{
		"objective": "Submit the form",
		"inputValues": map[string]any{
			"password":          secret,
			"recovery_question": browserFormSecret,
		},
	}

	now := time.Now().UTC()
	if err := database.CreateJob(context.Background(), &models.Job{
		ID:        jobID,
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	queryEncoded := url.QueryEscape(secret)
	percentEncoded := strings.ReplaceAll(queryEncoded, "+", "%20")
	payload := &events.JobFailedPayload{
		JobID:        jobID,
		Status:       events.JobStatusFailed,
		Stage:        events.JobFailStageScanning,
		Error:        "login failed at https://example.com/?password=" + queryEncoded,
		ErrorDetails: "alternate URLs contained " + percentEncoded + " and " + browserFormValue,
	}

	transitioned, err := database.FailJobWithTerminalEvent(
		context.Background(),
		jobID,
		payload.Stage,
		payload.Error,
		payload.ErrorDetails,
		payload,
	)
	if err != nil || !transitioned {
		t.Fatalf("FailJobWithTerminalEvent() transitioned=%v error=%v", transitioned, err)
	}

	job, err := database.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	outbox, err := database.ListUnpublishedTerminalEvents(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListUnpublishedTerminalEvents() error = %v", err)
	}

	combined := job.Error + job.ErrorDetails + payload.Error + payload.ErrorDetails

	if len(outbox) != 1 {
		t.Fatalf("terminal events = %d, want 1", len(outbox))
	}

	combined += outbox[0].PayloadJSON

	for _, value := range []string{
		secret,
		queryEncoded,
		percentEncoded,
		browserFormSecret,
		browserFormValue,
	} {
		if strings.Contains(combined, value) {
			t.Fatalf("terminal records leaked encoded secret %q: %s", value, combined)
		}
	}
}

func TestFailJobWithTerminalEventReusesCanonicalPayloadAfterConcurrentFailure(t *testing.T) {
	database := setupTestDB(t)
	createSensitiveTerminalJob(t, database, "job-failed-race", models.JobStateScanning)

	first := &events.JobFailedPayload{
		JobID:        "job-failed-race",
		Status:       events.JobStatusFailed,
		Stage:        events.JobFailStageScanning,
		Error:        "first " + terminalAICanary,
		ErrorDetails: "first " + terminalPasswordCanary,
	}

	transitioned, err := database.FailJobWithTerminalEvent(
		context.Background(),
		first.JobID,
		first.Stage,
		first.Error,
		first.ErrorDetails,
		first,
	)
	if err != nil || !transitioned {
		t.Fatalf("first failure: transitioned=%v err=%v", transitioned, err)
	}

	secondOnlyCanary := "second-failure-secret-canary"
	second := &events.JobFailedPayload{
		JobID:        first.JobID,
		Status:       events.JobStatusFailed,
		Stage:        events.JobFailStageScanning,
		Error:        "second " + secondOnlyCanary,
		ErrorDetails: "second " + terminalPasswordCanary,
	}

	transitioned, err = database.FailJobWithTerminalEvent(
		context.Background(),
		second.JobID,
		second.Stage,
		second.Error,
		second.ErrorDetails,
		second,
	)
	if err != nil || transitioned {
		t.Fatalf("idempotent failure: transitioned=%v err=%v", transitioned, err)
	}

	if strings.Contains(second.Error+second.ErrorDetails, secondOnlyCanary) ||
		strings.Contains(second.Error+second.ErrorDetails, terminalPasswordCanary) {
		t.Fatalf("idempotent caller retained unredacted text: %+v", second)
	}

	if second.Error != first.Error || second.ErrorDetails != first.ErrorDetails {
		t.Fatalf("idempotent payload = %+v, want canonical %+v", second, first)
	}
}

func TestFailJobWithTerminalEventConcurrentCallersPublishOnlyRedactedPayloads(t *testing.T) {
	database := setupTestDB(t)
	jobID := "job-failed-concurrently"
	createSensitiveTerminalJob(t, database, jobID, models.JobStateScanning)

	payloads := []*events.JobFailedPayload{
		{
			JobID: jobID, Status: events.JobStatusFailed, Stage: events.JobFailStageScanning,
			Error: "first " + terminalAICanary, ErrorDetails: "first " + terminalPasswordCanary,
		},
		{
			JobID: jobID, Status: events.JobStatusFailed, Stage: events.JobFailStageScanning,
			Error: "second " + terminalPasswordCanary, ErrorDetails: "second " + terminalAICanary,
		},
	}

	type result struct {
		transitioned bool
		err          error
	}

	results := make([]result, len(payloads))
	start := make(chan struct{})

	var wait sync.WaitGroup

	for index := range payloads {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()

			<-start

			payload := payloads[index]
			results[index].transitioned, results[index].err = database.FailJobWithTerminalEvent(
				context.Background(),
				jobID,
				payload.Stage,
				payload.Error,
				payload.ErrorDetails,
				payload,
			)
		}(index)
	}

	close(start)
	wait.Wait()

	transitionCount := 0

	for index, call := range results {
		if call.err != nil {
			t.Fatalf("failure caller %d: %v", index, call.err)
		}

		if call.transitioned {
			transitionCount++
		}

		if strings.Contains(payloads[index].Error+payloads[index].ErrorDetails, terminalAICanary) ||
			strings.Contains(payloads[index].Error+payloads[index].ErrorDetails, terminalPasswordCanary) {
			t.Fatalf("caller %d retained a canary: %+v", index, payloads[index])
		}
	}

	if transitionCount != 1 {
		t.Fatalf("transitioned callers = %d, want 1", transitionCount)
	}
}

func TestSanitizeLegacyTerminalRecordsBackfillsExistingSecrets(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()
	jobID := "job-legacy-terminal-secrets"
	createSensitiveTerminalJob(t, database, jobID, models.JobStateFailed)

	if _, err := database.execContext(ctx, `
		UPDATE jobs SET error = ?, error_details = ?, scanner_results = ? WHERE id = ?
	`,
		"failed with "+terminalAICanary,
		"details "+terminalPasswordCanary,
		`{"ai-navigator":{"error":"`+terminalUserCanary+`"}}`,
		jobID,
	); err != nil {
		t.Fatalf("seed legacy failure text: %v", err)
	}

	legacyPayload := `{
		"job_id":"` + jobID + `",
		"auth":{"mode":"form","steps":[{"type":"fill","value":"` + terminalPasswordCanary + `"}]},
		"pages":[{"pre_scan_actions":[{"type":"fill","value":"` + terminalUserCanary + `"}]}],
		"scanner_configs":{"ai-navigator":{"goal":{"inputValues":{"email":"` + terminalAICanary + `"}}}}
	}`
	if err := database.InsertJobEvent(ctx, &JobEventInsert{
		JobID:         jobID,
		Event:         events.EventJobCreated,
		Payload:       legacyPayload,
		HandlerStatus: "error",
		HandlerError:  "handler echoed " + terminalPasswordCanary,
	}); err != nil {
		t.Fatalf("seed legacy job event: %v", err)
	}

	terminalPayload := `{"job_id":"` + jobID + `","status":"failed","error":"` +
		terminalAICanary + `","error_details":"` + terminalPasswordCanary + `"}`
	if _, err := database.execContext(ctx, `
		INSERT INTO terminal_events (job_id, event, payload_json)
		VALUES (?, ?, ?)
	`, jobID, events.EventJobFailed, terminalPayload); err != nil {
		t.Fatalf("seed legacy terminal event: %v", err)
	}

	count, err := database.SanitizeLegacyTerminalRecords(ctx)
	if err != nil || count != 1 {
		t.Fatalf("SanitizeLegacyTerminalRecords() count=%d err=%v", count, err)
	}

	assertTerminalJobConfigRedacted(t, database, jobID)

	var jobText, auditPayload, handlerError, outboxPayload string
	if err = database.queryRowContext(ctx, `
		SELECT COALESCE(error, '') || COALESCE(error_details, '') || COALESCE(scanner_results, '')
		FROM jobs WHERE id = ?
	`, jobID).Scan(&jobText); err != nil {
		t.Fatalf("load sanitized job text: %v", err)
	}

	if err = database.queryRowContext(ctx, `
		SELECT COALESCE(payload_json, ''), COALESCE(handler_error, '')
		FROM job_events WHERE job_id = ?
	`, jobID).Scan(&auditPayload, &handlerError); err != nil {
		t.Fatalf("load sanitized audit event: %v", err)
	}

	if err = database.queryRowContext(ctx, `
		SELECT payload_json FROM terminal_events WHERE job_id = ? AND event = ?
	`, jobID, events.EventJobFailed).Scan(&outboxPayload); err != nil {
		t.Fatalf("load sanitized terminal event: %v", err)
	}

	combined := jobText + auditPayload + handlerError + outboxPayload
	for _, canary := range []string{terminalUserCanary, terminalPasswordCanary, terminalAICanary} {
		if strings.Contains(combined, canary) {
			t.Fatalf("legacy terminal records retained canary %q: %s", canary, combined)
		}
	}

	if !strings.Contains(auditPayload, `"pre_scan_actions":{"configured":true,"redacted":true}`) {
		t.Fatalf("legacy page actions were not summarized: %s", auditPayload)
	}

	count, err = database.SanitizeLegacyTerminalRecords(ctx)
	if err != nil || count != 0 {
		t.Fatalf("idempotent migration count=%d err=%v", count, err)
	}
}
