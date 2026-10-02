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
	authFormFixture   = `{"mode":"form","login_url":"https://app.example.com/login","steps":[{"type":"fill","selector":"input[name=email]","value":"demo@example.com"},{"type":"fill","selector":"input[name=password]","value":"throwaway-password"},{"type":"click","selector":"button[type=submit]"}],"success":{"type":"selector","selector":"[data-test=signed-in]","timeout":15000}}`
	authStorageFixed  = `{"mode":"storage_state"}`
	authStorageInline = `{"mode":"storage_state","content_b64":"eyJjb29raWVzIjpbXSwib3JpZ2lucyI6W119"}`
)

func newAuthPlanner(t *testing.T) *ScannerLaunchPlanner {
	t.Helper()

	return NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{
		NatsHost:           "nats",
		PageLoadTimeout:    15000,
		PodNetnsMode:       PodNetnsModeBridge,
		DefaultScannerUser: "0",
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

func TestPlanForwardsFormRecipeAsProvenanceAuth(t *testing.T) {
	t.Parallel()

	plan, err := newAuthPlanner(t).Plan(context.Background(), newAuthJob(t, authFormFixture), "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	authEnv, has := plan.Env["PROVENANCE_AUTH_JSON"]
	if !has {
		t.Fatal("PROVENANCE_AUTH_JSON not set")
	}

	if !strings.Contains(authEnv, `"mode":"form"`) || !strings.Contains(authEnv, `"login_url"`) {
		t.Fatalf("missing form recipe: %s", authEnv)
	}
}

// TestPlanRejectsFromEnvReferences is the orchestrator half of the from_env
// fix: whatever the name or step type, no recipe may pull a value out of the
// orchestrator's environment and into a scanner pod.
func TestPlanRejectsFromEnvReferences(t *testing.T) {
	t.Setenv("POSTGRES_PASSWORD", "host-secret-must-not-leak")

	for _, step := range []string{
		`{"type":"fill","selector":"#a","value":{"from_env":"POSTGRES_PASSWORD"}}`,
		`{"type":"select","selector":"#a","value":{"from_env":"DATABASE_URL"}}`,
		`{"type":"click","selector":"#a","value":{"from_env":"ORCHESTRATOR_API_TOKEN"}}`,
		`{"type":"fill","selector":"#a","value":{"from_env":"NATS_URL"}}`,
	} {
		t.Run(step, func(t *testing.T) {
			authJSON := `{"mode":"form","login_url":"https://x","steps":[` + step + `],"success":{"type":"load"}}`

			plan, err := newAuthPlanner(t).Plan(context.Background(), newAuthJob(t, authJSON), "axe")
			if err == nil {
				t.Fatalf("expected Plan() to refuse a from_env reference; env = %v", plan.Env)
			}

			if !strings.Contains(err.Error(), "from_env") {
				t.Errorf("error should name from_env; got: %v", err)
			}
		})
	}
}

func TestPlanStorageStateUsesMountedAuth(t *testing.T) {
	t.Parallel()

	planner := newAuthPlanner(t)

	plan, err := planner.Plan(context.Background(), newAuthJob(t, authStorageFixed), "axe")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	authEnv, has := plan.Env["PROVENANCE_AUTH_JSON"]
	if !has {
		t.Fatal("PROVENANCE_AUTH_JSON not set")
	}

	if authEnv != `{"mode":"storage_state"}` {
		t.Fatalf("execution session path exposed: %s", authEnv)
	}
}

func TestPlanStorageStateRejectsInlineContentAtLaunchTime(t *testing.T) {
	t.Parallel()

	planner := newAuthPlanner(t)

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

	planner := newAuthPlanner(t)

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
}
