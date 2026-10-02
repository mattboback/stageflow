package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestSnapshotUpdatesGolden(t *testing.T) {
	cases := []struct {
		name       string
		prev, next *jobstore.JobRecord
		ready      bool
	}{
		{
			"scan_progress",
			&jobstore.JobRecord{State: models.JobStateScanning},
			&jobstore.JobRecord{
				State:            models.JobStateScanning,
				CurrentPage:      2,
				TotalPages:       5,
				ExpectedScanners: []string{"axe"},
				ScannerStatuses: map[string]models.ScannerStatus{
					"axe": {State: "running", PagesScanned: 2, TotalPages: 5},
				},
			},
			true,
		},
		{
			"scan_complete",
			&jobstore.JobRecord{State: models.JobStateScanning},
			&jobstore.JobRecord{
				State:            models.JobStateScanning,
				TotalViolations:  9,
				ExpectedScanners: []string{"lighthouse"},
				ScannerStatuses: map[string]models.ScannerStatus{
					"lighthouse": {State: "completed", PagesScanned: 3, Violations: 7, TotalMs: 1234},
				},
			},
			true,
		},
		{
			"extraction_ready",
			&jobstore.JobRecord{State: models.JobStateReady},
			&jobstore.JobRecord{State: models.JobStateReady, TotalPages: 8},
			false,
		},
		{
			"job_failed",
			&jobstore.JobRecord{State: models.JobStateScanning},
			&jobstore.JobRecord{State: models.JobStateFailed, Error: "boom"},
			true,
		},
		{
			"job_complete",
			&jobstore.JobRecord{State: models.JobStateScanning},
			&jobstore.JobRecord{State: models.JobStateDone},
			true,
		},
		{
			"one_scanner_failed",
			&jobstore.JobRecord{State: models.JobStateScanning},
			&jobstore.JobRecord{
				State:            models.JobStateScanning,
				ExpectedScanners: []string{"seo"},
				ScannerStatuses:  map[string]models.ScannerStatus{"seo": {State: "failed", Error: "browser closed"}},
			},
			true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			updates := snapshotUpdates(tt.prev, tt.next, &tt.ready)
			if len(updates) != 1 {
				t.Fatalf("updates=%v", updates)
			}

			assertSSEPayloadGolden(t, tt.name, updates[0])
		})
	}
}
func TestLastScannerAndDoneInOneTick(t *testing.T) {
	previous := &jobstore.JobRecord{State: models.JobStateScanning}
	next := &jobstore.JobRecord{
		State:            models.JobStateDone,
		ExpectedScanners: []string{"axe"},
		ScannerStatuses:  map[string]models.ScannerStatus{"axe": {State: "completed", PagesScanned: 2}},
	}
	ready := true

	updates := snapshotUpdates(previous, next, &ready)
	if len(updates) != 2 || updates[0]["type"] != "scanner_complete" || updates[1]["type"] != "complete" {
		t.Fatalf("wrong ordering: %v", updates)
	}

	assertSSEPayloadGolden(t, "last_scanner_and_done", map[string]any{"updates": updates})
	data, _ := json.Marshal(updates[0])

	_, terminal, err := terminalDonePayloadFromUpdate(data)
	if terminal || err != nil {
		t.Fatalf("scanner completion prematurely ends stream: %v %v", terminal, err)
	}
}
func TestReadyCannotBeSkipped(t *testing.T) {
	ready := false

	updates := snapshotUpdates(
		&jobstore.JobRecord{State: models.JobStateExtracting},
		&jobstore.JobRecord{State: models.JobStateScanning, TotalPages: 30},
		&ready,
	)
	if len(updates) != 2 || updates[0]["state"] != "READY_TO_SCAN" || !ready {
		t.Fatalf("updates=%v", updates)
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
