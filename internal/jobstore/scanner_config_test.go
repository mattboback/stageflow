package jobstore

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestScannerConfigsSurviveAdmissionAndClaim(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	job := pendingJob("scanner-options")

	const configsJSON = `{"axe":{"disabledRules":["color-contrast"],"runOnlyTags":[],"dynamicContentWaitMs":500,"legacyOption":{"value":true}},"seo":{"custom":null}}`
	if err := json.Unmarshal([]byte(configsJSON), &job.Config.ScannerConfigs); err != nil {
		t.Fatal(err)
	}

	if err := db.AdmitJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	stored, err := db.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(stored.Config.ScannerConfigs, job.Config.ScannerConfigs) {
		t.Fatalf(
			"admission changed scanner options: got %#v, want %#v",
			stored.Config.ScannerConfigs,
			job.Config.ScannerConfigs,
		)
	}

	claimed, err := db.ClaimNext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if claimed == nil || !reflect.DeepEqual(claimed.Config.ScannerConfigs, job.Config.ScannerConfigs) {
		t.Fatalf("worker claim lost stored scanner options: %#v", claimed)
	}
}
