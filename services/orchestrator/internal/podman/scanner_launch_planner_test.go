package podman

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

func TestScannerLaunchPlannerPlan(t *testing.T) {
	t.Parallel()

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerImage:       "localhost/stageflow/scanner-runner:latest",
		PageLoadTimeout:    15000,
		PodNetnsMode:       PodNetnsModeBridge,
		DefaultScannerUser: "0",
	})

	job := &models.Job{
		ID:        "job-123",
		InputType: models.JobInputTypeZip,
		Config: models.JobConfig{
			Screenshot: true,
			Modules:    []string{"axe"},
		},
	}

	ctx := logging.WithRequestID(context.Background(), "req-123")
	ctx = logging.WithRunID(ctx, "run-123")

	plan, err := planner.Plan(ctx, job, "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := plan.Image, "localhost/stageflow/scanner-runner:latest"; got != want {
		t.Fatalf("plan.Image = %q, want %q", got, want)
	}

	if got, want := plan.Env["PROVENANCE_PATH"], "/job/provenance.json"; got != want {
		t.Fatalf("plan.Env[PROVENANCE_PATH] = %q, want %q", got, want)
	}

	if got, want := plan.Env["REQUEST_ID"], "req-123"; got != want {
		t.Fatalf("plan.Env[REQUEST_ID] = %q, want %q", got, want)
	}

	axe, _ := scannercatalog.Lookup("axe")
	if want := int64(axe.MaxMemoryMB); plan.ResourceLimits.MemoryLimitMB != want ||
		plan.ResourceLimits.MemorySwapMB != want {
		t.Fatalf("plan.ResourceLimits = %+v, want the manifest's %d MB", plan.ResourceLimits, want)
	}

	if len(plan.Volumes) != 2 {
		t.Fatalf("len(plan.Volumes) = %d, want 2", len(plan.Volumes))
	}

	if plan.Volumes[0].Name != "stageflow_data" || plan.Volumes[0].Subpath != "jobs/job-123" ||
		!plan.Volumes[0].ReadOnly {
		t.Fatalf("workspace volume = %+v, want read-only workspace-job-123", plan.Volumes[0])
	}
}

func TestScannerLaunchPlannerPlanURLJobHostNetns(t *testing.T) {
	t.Parallel()

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerImage:       "localhost/stageflow/scanner-runner:latest",
		PageLoadTimeout:    15000,
		PodNetnsMode:       PodNetnsModeHost,
		DefaultScannerUser: "0",
	})

	job := &models.Job{
		ID:        "job-urls",
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"http://localhost:3000"},
		Config: models.JobConfig{
			Modules:             []string{"axe"},
			AllowPrivateTargets: true,
		},
	}

	plan, err := planner.Plan(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := plan.Env["PROVENANCE_PATH"], "/tmp/provenance.json"; got != want {
		t.Fatalf("plan.Env[PROVENANCE_PATH] = %q, want %q", got, want)
	}

	if got, want := plan.Env["ALLOW_PRIVATE_TARGETS"], "true"; got != want {
		t.Fatalf("plan.Env[ALLOW_PRIVATE_TARGETS] = %q, want %q", got, want)
	}

	var urls []string
	if unmarshalErr := json.Unmarshal([]byte(plan.Env["SCAN_URLS"]), &urls); unmarshalErr != nil {
		t.Fatalf("SCAN_URLS unmarshal error = %v", unmarshalErr)
	}

	if len(urls) != 1 || urls[0] != "http://localhost:3000" {
		t.Fatalf("SCAN_URLS = %v, want localhost URL", urls)
	}
}

func TestScannerLaunchPlannerUsesConfiguredImageForEveryScanner(t *testing.T) {
	t.Parallel()

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerImage:       "localhost/stageflow/scanner-runner:pr-1",
		PageLoadTimeout:    15000,
		PodNetnsMode:       PodNetnsModeBridge,
		DefaultScannerUser: "0",
	})

	job := &models.Job{
		ID:        "job-image",
		InputType: models.JobInputTypeZip,
		Config:    models.JobConfig{Modules: scannercatalog.IDs()},
	}

	for _, scannerType := range scannercatalog.IDs() {
		plan, err := planner.Plan(context.Background(), job, scannerType)
		if err != nil {
			t.Fatalf("Plan(%s) error = %v", scannerType, err)
		}

		if plan.Image != "localhost/stageflow/scanner-runner:pr-1" {
			t.Fatalf("Plan(%s).Image = %q", scannerType, plan.Image)
		}
	}
}

// The orchestrator only plans scanners admission accepted, which are catalog
// IDs; anything else fails that scanner instead of launching a container.
func TestScannerLaunchPlannerRejectsNonCatalogScanners(t *testing.T) {
	t.Parallel()

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{ScannerImage: "image"})
	job := &models.Job{ID: "job-unknown", InputType: models.JobInputTypeZip}

	for _, scannerType := range []string{"", "a11y", "AXE", "custom-scanner"} {
		if _, err := planner.Plan(context.Background(), job, scannerType); err == nil {
			t.Errorf("Plan(%q) planned a container", scannerType)
		}
	}
}
