package podman

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
)

func TestScannerLaunchPlannerPlan(t *testing.T) {
	t.Parallel()

	registry := scanners.NewRegistry("localhost/stageflow/scanner-runner:latest")
	if checkErr1 := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
		Requirements: scanners.Requirements{
			MaxMemoryMB: 4096,
		},
	}); checkErr1 != nil {
		t.Fatalf("Register() error = %v", checkErr1)
	}

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerRegistry:    registry,
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

	if got, want := plan.ResourceLimits.MemoryLimitMB, int64(4096); got != want {
		t.Fatalf("plan.ResourceLimits.MemoryLimitMB = %d, want %d", got, want)
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

	registry := scanners.NewRegistry("localhost/stageflow/scanner-runner:latest")
	if checkErr2 := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
	}); checkErr2 != nil {
		t.Fatalf("Register() error = %v", checkErr2)
	}

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerRegistry:    registry,
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

	if plan.ResourceLimits.MemoryLimitMB != 2048 {
		t.Fatalf("plan.ResourceLimits.MemoryLimitMB = %d, want 2048", plan.ResourceLimits.MemoryLimitMB)
	}
}

func TestScannerLaunchPlannerPlanWithoutRegistryUsesDefaultScannerImage(t *testing.T) {
	t.Parallel()

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		DefaultScannerImage: "localhost/stageflow/scanner-runner:latest",
		PageLoadTimeout:     15000,
		PodNetnsMode:        PodNetnsModeBridge,
		DefaultScannerUser:  "0",
	})

	job := &models.Job{
		ID:        "job-no-registry",
		InputType: models.JobInputTypeZip,
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	plan, err := planner.Plan(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := plan.Image, "localhost/stageflow/scanner-runner:latest"; got != want {
		t.Fatalf("plan.Image = %q, want %q", got, want)
	}
}
