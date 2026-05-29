package jobs

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

const (
	authStateKey      = "job-auth-test/auth/storage-state.json"
	authFormFixture   = `{"mode":"form","login_url":"https://app.example.com/login","steps":[{"type":"fill","selector":"input[name=email]","value":{"from_env":"STAGEFLOW_AUTH_USER"}},{"type":"fill","selector":"input[name=password]","value":{"from_env":"STAGEFLOW_AUTH_PASSWORD"}},{"type":"click","selector":"button[type=submit]"}],"success":{"type":"selector","selector":"[data-test=signed-in]","timeout":15000}}`
	authStorageFixed  = `{"mode":"storage_state","artifact_key":"` + authStateKey + `"}`
	authStorageInline = `{"mode":"storage_state","content_b64":"eyJjb29raWVzIjpbXSwib3JpZ2lucyI6W119"}`
)

func newAuthPlanner(t *testing.T, hostEnv map[string]string) *ScannerLaunchPlanner {
	t.Helper()

	return NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		NatsHost:           "nats",
		MinioHost:          "minio",
		MinioAccessKey:     "k",
		MinioSecretKey:     "s",
		PageLoadTimeout:    15000,
		ScrollTimeout:      300,
		PodNetnsMode:       PodNetnsModeBridge,
		DefaultScannerUser: "0",
		HostEnv: func(name string) string {
			return hostEnv[name]
		},
	})
}

func newAuthJob(t *testing.T, authJSON string) *models.Job {
	t.Helper()

	return &models.Job{
		ID:        "job-auth-test",
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://app.example.com/profile"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
			Auth:    json.RawMessage(authJSON),
		},
	}
}

func TestPlanForwardsOnlyFromEnvAllowList(t *testing.T) {
	t.Parallel()

	host := map[string]string{
		"STAGEFLOW_AUTH_USER":     "demo@example.com",
		"STAGEFLOW_AUTH_PASSWORD": "redacted-password-NOT-leaked",
		"OTHER_HOST_SECRET":       "must-not-leak",
		"AWS_SESSION_TOKEN":       "must-not-leak-either",
	}

	planner := newAuthPlanner(t, host)

	plan, err := planner.Plan(context.Background(), newAuthJob(t, authFormFixture), "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// Allow-listed env vars are forwarded.
	if got := plan.Env["STAGEFLOW_AUTH_USER"]; got != "demo@example.com" {
		t.Errorf("STAGEFLOW_AUTH_USER = %q, want demo@example.com", got)
	}

	if got := plan.Env["STAGEFLOW_AUTH_PASSWORD"]; got != "redacted-password-NOT-leaked" {
		t.Errorf("STAGEFLOW_AUTH_PASSWORD = %q, want forwarded literal", got)
	}

	// Anything outside the recipe's allow-list stays out of the pod env.
	for _, name := range []string{"OTHER_HOST_SECRET", "AWS_SESSION_TOKEN"} {
		if _, leaked := plan.Env[name]; leaked {
			t.Errorf("env var %q leaked into pod env from orchestrator host", name)
		}
	}

	// PROVENANCE_AUTH_JSON carries from_env references but never resolved values.
	authEnv, has := plan.Env["PROVENANCE_AUTH_JSON"]
	if !has {
		t.Fatal("PROVENANCE_AUTH_JSON not set")
	}

	if !strings.Contains(authEnv, `"from_env":"STAGEFLOW_AUTH_USER"`) {
		t.Errorf("PROVENANCE_AUTH_JSON missing from_env reference: %s", authEnv)
	}

	if strings.Contains(authEnv, "demo@example.com") || strings.Contains(authEnv, "redacted-password-NOT-leaked") {
		t.Errorf("resolved credentials leaked into PROVENANCE_AUTH_JSON: %s", authEnv)
	}
}

func TestPlanFailsFastWhenFromEnvIsUnset(t *testing.T) {
	t.Parallel()

	// Only USER is set; PASSWORD is missing on the host.
	host := map[string]string{"STAGEFLOW_AUTH_USER": "demo@example.com"}

	planner := newAuthPlanner(t, host)

	_, err := planner.Plan(context.Background(), newAuthJob(t, authFormFixture), "axe")
	if err == nil {
		t.Fatal("expected Plan() to fail when a from_env reference is unset on the host")
	}

	if !strings.Contains(err.Error(), "STAGEFLOW_AUTH_PASSWORD") {
		t.Errorf("error should name the missing env var; got: %v", err)
	}

	if !strings.Contains(err.Error(), "from_env references not set") {
		t.Errorf("error should be the structured 'from_env references not set' message; got: %v", err)
	}
}

func TestPlanStorageStateForwardsArtifactKey(t *testing.T) {
	t.Parallel()

	host := map[string]string{"OTHER_HOST_SECRET": "must-not-leak"}

	planner := newAuthPlanner(t, host)

	plan, err := planner.Plan(context.Background(), newAuthJob(t, authStorageFixed), "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	authEnv, has := plan.Env["PROVENANCE_AUTH_JSON"]
	if !has {
		t.Fatal("PROVENANCE_AUTH_JSON not set")
	}

	if !strings.Contains(authEnv, `"artifact_key":"`+authStateKey+`"`) {
		t.Errorf("PROVENANCE_AUTH_JSON missing artifact_key: %s", authEnv)
	}

	// storage_state mode must not forward any host env vars.
	for _, name := range []string{"OTHER_HOST_SECRET", "STAGEFLOW_AUTH_USER", "STAGEFLOW_AUTH_PASSWORD"} {
		if _, leaked := plan.Env[name]; leaked {
			t.Errorf("env var %q leaked into pod env in storage_state mode", name)
		}
	}
}

func TestPlanStorageStateRejectsInlineContentAtLaunchTime(t *testing.T) {
	t.Parallel()

	planner := newAuthPlanner(t, nil)

	_, err := planner.Plan(context.Background(), newAuthJob(t, authStorageInline), "axe")
	if err == nil {
		t.Fatal("expected Plan() to refuse inline content_b64 at scanner-launch time")
	}

	if !strings.Contains(err.Error(), "content_b64") {
		t.Errorf("error should mention content_b64; got: %v", err)
	}
}

func TestPlanWithoutAuthIsByteIdentical(t *testing.T) {
	t.Parallel()

	planner := newAuthPlanner(t, map[string]string{"STAGEFLOW_AUTH_USER": "set-but-recipe-absent"})

	job := &models.Job{
		ID:        "job-no-auth",
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	plan, err := planner.Plan(context.Background(), job, "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if _, has := plan.Env["PROVENANCE_AUTH_JSON"]; has {
		t.Errorf("PROVENANCE_AUTH_JSON must not be set when auth is absent")
	}

	if _, leaked := plan.Env["STAGEFLOW_AUTH_USER"]; leaked {
		t.Errorf("host env var leaked into pod when no auth recipe")
	}
}

func TestPlanRejectsCollisionWithReservedEnvName(t *testing.T) {
	t.Parallel()

	// A pathological recipe that tries to overwrite NATS_URL via from_env.
	authJSON := `{"mode":"form","login_url":"https://x","steps":[{"type":"fill","selector":"#a","value":{"from_env":"NATS_URL"}}],"success":{"type":"load"}}`

	planner := newAuthPlanner(t, map[string]string{"NATS_URL": "evil://attacker"})

	_, err := planner.Plan(context.Background(), newAuthJob(t, authJSON), "axe")
	if err == nil {
		t.Fatal("expected Plan() to reject from_env collision with reserved scanner env vars")
	}

	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("error should mention reserved name; got: %v", err)
	}
}
