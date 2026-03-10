package jobs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/logging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	scanners "github.com/mattboback/stageflow/packages/shared-go/scannerregistry"
)

func TestScannerLaunchPlannerPlan(t *testing.T) {
	t.Parallel()

	registry := scanners.NewRegistry("localhost/stageflow/scanner-runner:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
		Requirements: scanners.Requirements{
			MaxMemoryMB: 4096,
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerRegistry:    registry,
		NatsHost:           "nats",
		MinioHost:          "minio",
		MinioAccessKey:     "minioadmin",
		MinioSecretKey:     "secret",
		PageLoadTimeout:    15000,
		ScrollTimeout:      300,
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

	if got, want := plan.Env["NATS_URL"], "nats://nats:4222"; got != want {
		t.Fatalf("plan.Env[NATS_URL] = %q, want %q", got, want)
	}

	if got, want := plan.Env["PROVENANCE_PATH"], "/workspace/provenance.json"; got != want {
		t.Fatalf("plan.Env[PROVENANCE_PATH] = %q, want %q", got, want)
	}

	if got, want := plan.Env["A11Y_HIGHLIGHT_STYLE"], "dashed"; got != want {
		t.Fatalf("plan.Env[A11Y_HIGHLIGHT_STYLE] = %q, want %q", got, want)
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

	if plan.Volumes[0].Name != "workspace-job-123" || !plan.Volumes[0].ReadOnly {
		t.Fatalf("workspace volume = %+v, want read-only workspace-job-123", plan.Volumes[0])
	}
}

func TestScannerLaunchPlannerPlanURLJobHostNetns(t *testing.T) {
	t.Parallel()

	registry := scanners.NewRegistry("localhost/stageflow/scanner-runner:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "ai-navigator",
		Name:    "ai navigator",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	planner := NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		ScannerRegistry:      registry,
		NatsHost:             "nats",
		MinioHost:            "minio",
		MinioAccessKey:       "minioadmin",
		MinioSecretKey:       "secret",
		PageLoadTimeout:      15000,
		ScrollTimeout:        300,
		PodNetnsMode:         PodNetnsModeHost,
		DefaultScannerUser:   "0",
		OpenRouterAPIKey:     "openrouter-key",
		OpenRouterAppTitle:   "StageFlow",
		OpenRouterAppReferer: "https://stageflow.org",
	})

	job := &models.Job{
		ID:        "job-urls",
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"http://localhost:3000"},
		Config: models.JobConfig{
			Modules:             []string{"ai-navigator"},
			HighlightStyle:      "solid",
			AllowPrivateTargets: true,
			ScannerConfigs: map[string]map[string]any{
				"ai-navigator": {"model": "openai/gpt-5"},
			},
		},
	}

	plan, err := planner.Plan(context.Background(), job, "ai-navigator")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := plan.Env["NATS_URL"], "nats://127.0.0.1:4222"; got != want {
		t.Fatalf("plan.Env[NATS_URL] = %q, want %q", got, want)
	}

	if got, want := plan.Env["MINIO_ENDPOINT"], "127.0.0.1:9000"; got != want {
		t.Fatalf("plan.Env[MINIO_ENDPOINT] = %q, want %q", got, want)
	}

	if got, want := plan.Env["PROVENANCE_PATH"], "/results/ai-navigator/provenance.json"; got != want {
		t.Fatalf("plan.Env[PROVENANCE_PATH] = %q, want %q", got, want)
	}

	if got, want := plan.Env["ALLOW_PRIVATE_TARGETS"], "true"; got != want {
		t.Fatalf("plan.Env[ALLOW_PRIVATE_TARGETS] = %q, want %q", got, want)
	}

	if got, want := plan.Env["OPENROUTER_API_KEY"], "openrouter-key"; got != want {
		t.Fatalf("plan.Env[OPENROUTER_API_KEY] = %q, want %q", got, want)
	}

	if got, want := plan.Env["A11Y_HIGHLIGHT_STYLE"], "solid"; got != want {
		t.Fatalf("plan.Env[A11Y_HIGHLIGHT_STYLE] = %q, want %q", got, want)
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
		NatsHost:            "nats",
		MinioHost:           "minio",
		MinioAccessKey:      "minioadmin",
		MinioSecretKey:      "secret",
		PageLoadTimeout:     15000,
		ScrollTimeout:       300,
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
