package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func TestMapChangeToSSEPayload_Golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change jobstatus.Change
	}{
		{
			name: "scan_progress",
			change: jobstatus.Change{
				Signal: jobstatus.Signal{
					Kind: jobstatus.SignalScanPageCompleted,
					ScanPageCompleted: &events.ScanPageCompletedPayload{
						ScannerType: "axe",
					},
				},
				Snapshot: &status.JobRecord{
					State:       models.JobStateScanning,
					CurrentPage: 2,
					TotalPages:  5,
				},
			},
		},
		{
			name: "scan_complete",
			change: jobstatus.Change{
				Signal: jobstatus.Signal{
					Kind: jobstatus.SignalScanCompleted,
					ScanCompleted: &events.ScanCompletedPayload{
						ScannerType:       "lighthouse",
						TotalPagesScanned: 3,
						Summary:           events.ScanSummary{TotalViolations: 7},
						Timing:            &events.ScanTiming{TotalMs: 1234},
					},
				},
				Snapshot: &status.JobRecord{
					State:           models.JobStateScanning,
					CurrentPage:     3,
					TotalViolations: 7,
				},
			},
		},
		{
			name: "extraction_ready",
			change: jobstatus.Change{
				Signal: jobstatus.Signal{Kind: jobstatus.SignalExtractionReady},
				Snapshot: &status.JobRecord{
					State:      models.JobStateReady,
					TotalPages: 8,
				},
			},
		},
		{
			name: "job_failed",
			change: jobstatus.Change{
				Signal: jobstatus.Signal{Kind: jobstatus.SignalJobFailed},
				Snapshot: &status.JobRecord{
					State: models.JobStateFailed,
					Error: "boom",
				},
			},
		},
		{
			name: "job_complete",
			change: jobstatus.Change{
				Signal: jobstatus.Signal{Kind: jobstatus.SignalJobCompleted},
				Snapshot: &status.JobRecord{
					State: models.JobStateDone,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertSSEPayloadGolden(t, tt.name, mapChangeToSSEPayload(tt.change))
		})
	}
}

func TestTerminalDonePayloadFromMappedCompletePayload(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(mapChangeToSSEPayload(jobstatus.Change{
		Signal: jobstatus.Signal{Kind: jobstatus.SignalJobCompleted},
		Snapshot: &status.JobRecord{
			State: models.JobStateDone,
		},
	}))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	done, isTerminal, parseErr := terminalDonePayloadFromUpdate(payload)
	if parseErr != nil {
		t.Fatalf("terminalDonePayloadFromUpdate error = %v", parseErr)
	}

	if !isTerminal {
		t.Fatal("expected mapped complete payload to be terminal")
	}

	if done.State != models.JobStateDone {
		t.Fatalf("done.State = %q, want %q", done.State, models.JobStateDone)
	}
}

func assertSSEPayloadGolden(t *testing.T, name string, payload map[string]any) {
	t.Helper()

	got, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	got = append(got, '\n')

	goldenPath := filepath.Join("testdata", "sse_payloads", name+".golden.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", name, want, got)
	}
}
