package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	scanners "github.com/mattboback/stageflow/packages/shared-go/scannerregistry"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func TestStartScanning_StateTransitions(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	tests := []struct {
		name         string
		initialState models.JobState
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid transition from READY",
			initialState: models.JobStateReady,
			expectError:  false,
		},
		{
			name:         "already in SCANNING state",
			initialState: models.JobStateScanning,
			expectError:  false, // Should not error, just continue
		},
		{
			name:         "invalid transition from DONE",
			initialState: models.JobStateDone,
			expectError:  true,
			// Error message includes job ID prefix
		},
		{
			name:         "invalid transition from FAILED",
			initialState: models.JobStateFailed,
			expectError:  true,
			// Error message includes job ID prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &models.Job{
				ID:        "job-" + tt.name,
				State:     tt.initialState,
				InputType: "zip",
				InputPath: "test.zip",
				PodID:     "pod-123",
				Config:    models.JobConfig{Modules: []string{"axe"}},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			insertJob(t, database, job)

			err := orch.startScanning(context.Background(), job)

			//nolint:nestif // Test error validation requires nested checks
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Verify state was updated
				updatedJob, getErr := database.GetJob(context.Background(), job.ID)
				if getErr != nil {
					t.Fatalf("failed to get job: %v", getErr)
				}

				if updatedJob.State != models.JobStateScanning {
					t.Fatalf("expected state SCANNING, got %s", updatedJob.State)
				}
			}
		})
	}
}

func TestStartScanning_SetExpectedScanners(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateReady,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe", "lighthouse", "seo"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startScanning(context.Background(), job)
	if err != nil {
		t.Fatalf("startScanning failed: %v", err)
	}

	// Verify expected scanners were set
	updatedJob, err := database.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	expectedScanners := orch.getScannerTypes(job.Config.Modules)
	if len(updatedJob.ExpectedScanners) != len(expectedScanners) {
		t.Fatalf("expected %d scanners, got %d", len(expectedScanners), len(updatedJob.ExpectedScanners))
	}

	// Verify CompletedScanners is initialized as empty
	if len(updatedJob.CompletedScanners) != 0 {
		t.Fatalf("expected empty CompletedScanners, got %d", len(updatedJob.CompletedScanners))
	}

	// Verify ScannerResults is initialized
	if updatedJob.ScannerResults == nil {
		t.Fatal("expected ScannerResults to be initialized")
	}
}

func TestStartScanning_NoScannersResolved(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	orch.scannerRegistry = scanners.NewRegistry("localhost/stageflow/scanner-runner:latest")

	job := &models.Job{
		ID:        "job-no-scanners",
		State:     models.JobStateReady,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startScanning(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for empty scanner resolution, got nil")
	}

	updatedJob, err := database.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if updatedJob.State != models.JobStateReady {
		t.Fatalf("expected state READY, got %s", updatedJob.State)
	}
}

func TestStartScanning_WatchdogTimeout(t *testing.T) {
	// This test verifies that the watchdog goroutine is started
	// We can't easily test the actual timeout without waiting, but we can verify
	// that the function completes and the goroutine is spawned
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Set a very short timeout for testing
	orch.scanTimeout = 100 * time.Millisecond

	job := &models.Job{
		ID:        "job-timeout",
		State:     models.JobStateReady,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startScanning(context.Background(), job)
	if err != nil {
		t.Fatalf("startScanning failed: %v", err)
	}

	// The watchdog goroutine should be running
	// We just verify the function returned successfully
}

func TestGetScannerTypes_ModuleResolution(t *testing.T) {
	orch, _, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	tests := []struct {
		name           string
		modules        []string
		expectedCount  int
		expectContains []string
	}{
		{
			name:           "single module",
			modules:        []string{"axe"},
			expectedCount:  1,
			expectContains: []string{"axe"},
		},
		{
			name:           "multiple modules",
			modules:        []string{"axe", "lighthouse"},
			expectedCount:  2,
			expectContains: []string{"axe", "lighthouse"},
		},
		{
			name:           "empty modules",
			modules:        []string{},
			expectedCount:  1, // Scanner registry returns default scanner for empty modules
			expectContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanners := orch.getScannerTypes(tt.modules)

			if len(scanners) != tt.expectedCount {
				t.Fatalf("expected %d scanners, got %d", tt.expectedCount, len(scanners))
			}

			for _, expected := range tt.expectContains {
				found := false

				for _, scanner := range scanners {
					if scanner == expected {
						found = true
						break
					}
				}

				if !found {
					t.Fatalf("expected scanner %q not found in result", expected)
				}
			}
		})
	}
}

func TestStartScannersWithTimeout_Timeout(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Override mock to delay container creation
	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		// Wait longer than timeout
		time.Sleep(200 * time.Millisecond)
		return &podman.ContainerCreateResponse{ID: "container-delayed"}, nil
	}

	job := &models.Job{
		ID:        "job-timeout",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	ctx := context.Background()
	scannerTypes := []string{"axe"}

	// Set short timeout
	orch.scanTimeout = 50 * time.Millisecond

	err := orch.startScannersWithTimeout(ctx, job, scannerTypes)

	// Should timeout
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if err.Error() != "scanner start timed out after 50ms" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartScannersWithTimeout_ConcurrentFailure(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Override mock to fail for specific scanner
	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		if req.Name == "scanner-lighthouse-job-fail" {
			return nil, errors.New("failed to create lighthouse container")
		}

		return &podman.ContainerCreateResponse{ID: "container-success"}, nil
	}

	job := &models.Job{
		ID:        "job-fail",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe", "lighthouse"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	ctx := context.Background()
	scannerTypes := []string{"axe", "lighthouse"}

	err := orch.startScannersWithTimeout(ctx, job, scannerTypes)

	// Should fail with the first error encountered
	if err == nil {
		t.Fatal("expected error from failed scanner")
	}
}

func TestStartSingleScanner_EnvironmentVariables(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Capture environment variables passed to container
	var capturedEnv map[string]string

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedEnv = req.Env
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-env",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config: models.JobConfig{
			Modules:        []string{"axe"},
			Screenshot:     true,
			HighlightStyle: "solid",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify required environment variables
	requiredVars := []string{
		"JOB_ID",
		"SCANNER_TYPE",
		"NATS_URL",
		"MINIO_ENDPOINT",
		"MINIO_ACCESS_KEY",
		"MINIO_SECRET_KEY",
		"MINIO_USE_SSL",
		"MINIO_ARTIFACT_BUCKET",
		"PROVENANCE_PATH",
		"RESULTS_DIR",
		"PAGE_LOAD_TIMEOUT",
		"A11Y_SCROLL_TIMEOUT",
		"A11Y_SHOT_ENABLED",
		"A11Y_HIGHLIGHT_STYLE",
	}

	for _, varName := range requiredVars {
		if _, exists := capturedEnv[varName]; !exists {
			t.Fatalf("expected environment variable %s not found", varName)
		}
	}

	// Verify specific values
	if capturedEnv["JOB_ID"] != job.ID {
		t.Fatalf("expected JOB_ID=%s, got %s", job.ID, capturedEnv["JOB_ID"])
	}

	if capturedEnv["SCANNER_TYPE"] != "axe" {
		t.Fatalf("expected SCANNER_TYPE=axe, got %s", capturedEnv["SCANNER_TYPE"])
	}

	if capturedEnv["A11Y_SHOT_ENABLED"] != "true" {
		t.Fatalf("expected A11Y_SHOT_ENABLED=true, got %s", capturedEnv["A11Y_SHOT_ENABLED"])
	}

	if capturedEnv["A11Y_HIGHLIGHT_STYLE"] != "solid" {
		t.Fatalf("expected A11Y_HIGHLIGHT_STYLE=solid, got %s", capturedEnv["A11Y_HIGHLIGHT_STYLE"])
	}
}

func TestStartSingleScanner_URLJobProvenance(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedEnv map[string]string

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedEnv = req.Env
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-url",
		State:     models.JobStateScanning,
		InputType: inputTypeURLs,
		URLs:      []string{"https://example.com", "https://test.com"},
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify SCAN_URLS is set
	if scanURLs, exists := capturedEnv["SCAN_URLS"]; !exists {
		t.Fatal("expected SCAN_URLS environment variable for URL job")
	} else {
		var urls []string
		if unmarshalErr := json.Unmarshal([]byte(scanURLs), &urls); unmarshalErr != nil {
			t.Fatalf("failed to unmarshal SCAN_URLS: %v", unmarshalErr)
		}

		if len(urls) != 2 {
			t.Fatalf("expected 2 URLs, got %d", len(urls))
		}
	}

	// Verify provenance path is in results dir for URL jobs
	if provenancePath, exists := capturedEnv["PROVENANCE_PATH"]; !exists {
		t.Fatal("expected PROVENANCE_PATH environment variable")
	} else {
		expectedPath := "/results/axe/provenance.json"
		if provenancePath != expectedPath {
			t.Fatalf("expected PROVENANCE_PATH=%s, got %s", expectedPath, provenancePath)
		}
	}
}

func TestStartSingleScanner_SetsAllowPrivateTargetsEnvVar(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedEnv map[string]string

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedEnv = req.Env
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-url-private",
		State:     models.JobStateScanning,
		InputType: inputTypeURLs,
		URLs:      []string{"https://example.com"},
		PodID:     "pod-123",
		Config: models.JobConfig{
			Modules:             []string{"axe"},
			AllowPrivateTargets: true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	if capturedEnv["ALLOW_PRIVATE_TARGETS"] != "true" {
		t.Fatalf("expected ALLOW_PRIVATE_TARGETS=true, got %q", capturedEnv["ALLOW_PRIVATE_TARGETS"])
	}
}

func TestStartSingleScanner_AINavigatorAPIKey(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Set the API key environment variable
	os.Setenv("OPENROUTER_API_KEY", "test-api-key-123")

	defer os.Unsetenv("OPENROUTER_API_KEY")

	os.Setenv("OPENROUTER_APP_TITLE", "StageFlow Test")

	defer os.Unsetenv("OPENROUTER_APP_TITLE")

	os.Setenv("OPENROUTER_APP_REFERER", "https://example.com")

	defer os.Unsetenv("OPENROUTER_APP_REFERER")

	var capturedEnv map[string]string

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedEnv = req.Env
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-ai",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"ai-navigator"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "ai-navigator")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify API key is passed
	if apiKey, exists := capturedEnv["OPENROUTER_API_KEY"]; !exists {
		t.Fatal("expected OPENROUTER_API_KEY for ai-navigator")
	} else if apiKey != "test-api-key-123" {
		t.Fatalf("expected API key test-api-key-123, got %s", apiKey)
	}

	if title, exists := capturedEnv["OPENROUTER_APP_TITLE"]; !exists {
		t.Fatal("expected OPENROUTER_APP_TITLE for ai-navigator")
	} else if title != "StageFlow Test" {
		t.Fatalf("expected OPENROUTER_APP_TITLE StageFlow Test, got %s", title)
	}

	if referer, exists := capturedEnv["OPENROUTER_APP_REFERER"]; !exists {
		t.Fatal("expected OPENROUTER_APP_REFERER for ai-navigator")
	} else if referer != "https://example.com" {
		t.Fatalf("expected OPENROUTER_APP_REFERER https://example.com, got %s", referer)
	}
}

func TestStartSingleScanner_ScannerConfig(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedEnv map[string]string

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedEnv = req.Env
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	scannerConfig := map[string]any{
		"maxPages": 10,
		"timeout":  30,
	}

	job := &models.Job{
		ID:        "job-config",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config: models.JobConfig{
			Modules: []string{"axe"},
			ScannerConfigs: map[string]map[string]any{
				"axe": scannerConfig,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify SCANNER_OPTIONS is set
	if options, exists := capturedEnv["SCANNER_OPTIONS"]; !exists {
		t.Fatal("expected SCANNER_OPTIONS environment variable")
	} else {
		var parsedConfig map[string]any
		if unmarshalErr := json.Unmarshal([]byte(options), &parsedConfig); unmarshalErr != nil {
			t.Fatalf("failed to unmarshal SCANNER_OPTIONS: %v", unmarshalErr)
		}

		if parsedConfig["maxPages"].(float64) != 10 {
			t.Fatalf("expected maxPages=10, got %v", parsedConfig["maxPages"])
		}
	}
}

func TestStartSingleScanner_VolumeMounts(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedMounts []podman.VolumeMount

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedMounts = req.Mounts
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-volumes",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify workspace and results volumes are mounted
	if len(capturedMounts) != 2 {
		t.Fatalf("expected 2 volume mounts, got %d", len(capturedMounts))
	}

	// Verify workspace mount is read-only
	workspaceMount := capturedMounts[0]
	if workspaceMount.Destination != "/workspace" {
		t.Fatalf("expected workspace mount at /workspace, got %s", workspaceMount.Destination)
	}

	if !workspaceMount.ReadOnly {
		t.Fatal("expected workspace mount to be read-only")
	}

	// Verify results mount is writable
	resultsMount := capturedMounts[1]
	if resultsMount.Destination != "/results" {
		t.Fatalf("expected results mount at /results, got %s", resultsMount.Destination)
	}

	if resultsMount.ReadOnly {
		t.Fatal("expected results mount to be writable")
	}
}

func TestStartSingleScanner_ResourceLimits(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedLimits *podman.ResourceLimits

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedLimits = req.ResourceLimits
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-limits",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify resource limits are set
	if capturedLimits == nil {
		t.Fatal("expected resource limits to be set")
	}

	// Default limit should be 2048 MB
	if capturedLimits.MemoryLimitMB != 2048 {
		t.Fatalf("expected memory limit 2048, got %d", capturedLimits.MemoryLimitMB)
	}

	if capturedLimits.MemorySwapMB != 2048 {
		t.Fatalf("expected memory swap 2048, got %d", capturedLimits.MemorySwapMB)
	}
}

func TestStartSingleScanner_ContainerLabels(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedLabels map[string]string

	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.createContainerFunc = func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
		capturedLabels = req.Labels
		return &podman.ContainerCreateResponse{ID: "container-123"}, nil
	}

	job := &models.Job{
		ID:        "job-labels",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"lighthouse"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "lighthouse")
	if err != nil {
		t.Fatalf("startSingleScanner failed: %v", err)
	}

	// Verify labels are set correctly
	expectedLabels := map[string]string{
		"managed_by":   "orchestrator",
		"job_id":       job.ID,
		"component":    "scanner",
		"scanner_type": "lighthouse",
	}

	for key, expectedValue := range expectedLabels {
		if actualValue, exists := capturedLabels[key]; !exists {
			t.Fatalf("expected label %s not found", key)
		} else if actualValue != expectedValue {
			t.Fatalf("label %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}

func TestGetScannerImage_WithRegistry(t *testing.T) {
	orch, _, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Default scanner image should be used
	image := orch.getScannerImage("axe")
	if image != orch.scannerImage {
		t.Fatalf("expected default scanner image, got %s", image)
	}
}

func TestGetScannerImage_WithoutRegistry(t *testing.T) {
	orch, _, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Set registry to nil to test fallback
	orch.scannerRegistry = nil

	image := orch.getScannerImage("axe")
	if image != orch.scannerImage {
		t.Fatalf("expected default scanner image, got %s", image)
	}
}

func TestNewScannerLaunchPlannerUsesRegistryDefaultImageWhenNoOverride(t *testing.T) {
	t.Parallel()

	registry := scanners.NewRegistry("registry/default:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	orch := NewOrchestrator(&Config{
		PodmanClient:    &mockPodmanClient{},
		Database:        newInMemoryDB(t),
		Publisher:       &mockPublisher{},
		ScannerRegistry: registry,
		ScannerImage:    "localhost/stageflow/scanner-runner:latest",
	})

	job := &models.Job{
		ID:        "job-registry-default-image",
		InputType: models.JobInputTypeZip,
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	plan, err := orch.newScannerLaunchPlanner().Plan(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := plan.Image, "registry/default:latest"; got != want {
		t.Fatalf("plan.Image = %q, want %q", got, want)
	}
}

func TestNewScannerLaunchPlannerUsesExplicitScannerImageOverride(t *testing.T) {
	t.Parallel()

	registry := scanners.NewRegistry("registry/default:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	orch := NewOrchestrator(&Config{
		PodmanClient:         &mockPodmanClient{},
		Database:             newInMemoryDB(t),
		Publisher:            &mockPublisher{},
		ScannerRegistry:      registry,
		ScannerImage:         "explicit/override:latest",
		ScannerImageOverride: "explicit/override:latest",
	})

	job := &models.Job{
		ID:        "job-scanner-image-override",
		InputType: models.JobInputTypeZip,
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	plan, err := orch.newScannerLaunchPlanner().Plan(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := plan.Image, "explicit/override:latest"; got != want {
		t.Fatalf("plan.Image = %q, want %q", got, want)
	}
}

func TestStartSingleScanner_VolumeCreationFailure(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Override mock to fail volume inspection
	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.inspectVolumeFunc = func(_ context.Context, _ string) (*podman.VolumeInfo, error) {
		return nil, errors.New("volume not found")
	}
	mockPodman.createVolumeFunc = func(_ context.Context, _ string) error {
		return errors.New("failed to create volume")
	}

	job := &models.Job{
		ID:        "job-vol-fail",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err == nil {
		t.Fatal("expected error for volume creation failure")
	}
}

func TestStartSingleScanner_ContainerStartFailure(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	// Override mock to fail container start
	mockPodman := orch.podmanClient.(*mockPodmanClient)
	mockPodman.startContainerFunc = func(_ context.Context, _ string) error {
		return errors.New("failed to start container")
	}

	job := &models.Job{
		ID:        "job-start-fail",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.startSingleScanner(context.Background(), job, "axe")
	if err == nil {
		t.Fatal("expected error for container start failure")
	}

	if err.Error() != "failed to start axe scanner container: failed to start container" {
		t.Fatalf("unexpected error message: %v", err)
	}
}
