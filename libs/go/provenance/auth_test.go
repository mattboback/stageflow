package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// loadAuthFromFixture reads a Provenance fixture and returns its auth block.
// The fixtures are shared with the TypeScript implementation so the Go and TS
// CollectFromEnvReferences must agree on identical inputs.
func loadAuthFromFixture(t *testing.T, name string) *Auth {
	t.Helper()

	root := mustFindRepoRoot(t)
	path := filepath.Join(root, "libs", "contracts", "provenance", "fixtures", name)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	var doc struct {
		Auth *Auth `json:"auth"`
	}
	if unmarshalErr := json.Unmarshal(data, &doc); unmarshalErr != nil {
		t.Fatalf("parse fixture %s: %v", path, unmarshalErr)
	}

	return doc.Auth
}

func mustFindRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.work")); statErr == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root (go.work) not found from cwd")
		}

		dir = parent
	}
}

func TestCollectFromEnvReferences_FromFormFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.auth-form.json")
	if auth == nil {
		t.Fatal("expected auth block in fixture")
	}

	got := CollectFromEnvReferences(auth)
	want := []string{"STAGEFLOW_AUTH_PASSWORD", "STAGEFLOW_AUTH_USER"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v (must match the TypeScript secrets-resolver output)", got, want)
	}
}

func TestCollectFromEnvReferences_FromStorageStateFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.auth-storage-state.json")

	got := CollectFromEnvReferences(auth)
	if len(got) != 0 {
		t.Fatalf("storage_state must produce no env refs; got %v", got)
	}
}

func TestCollectFromEnvReferences_FormLiteralFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.auth-form-literal.json")

	got := CollectFromEnvReferences(auth)
	if len(got) != 0 {
		t.Fatalf("literal-only recipe must produce no env refs; got %v", got)
	}
}

func TestCollectFromEnvReferences_NoAuthFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.no-auth.json")
	if auth != nil {
		t.Fatalf("expected nil auth, got %#v", auth)
	}

	got := CollectFromEnvReferences(auth)
	if len(got) != 0 {
		t.Fatalf("absent auth must produce no env refs; got %v", got)
	}
}

// TestCollectFromEnvReferences_DedupesAndSortsAcrossFormSteps mirrors the
// TypeScript test that walks multiple from_env references and asserts the
// deduplicated, sorted output.
func TestCollectFromEnvReferences_DedupesAndSortsAcrossFormSteps(t *testing.T) {
	t.Parallel()

	user := "STAGEFLOW_AUTH_USER"
	password := "STAGEFLOW_AUTH_PASSWORD"
	tenant := "TENANT_TOKEN"

	auth := &Auth{
		Mode: AuthModeForm,
		Form: &FormRecipe{
			LoginURL: "https://app.example.com/login",
			Steps: []FormStep{
				{Type: "fill", Selector: "#email", Value: &FormStepValue{FromEnv: &user}},
				{Type: "fill", Selector: "#password", Value: &FormStepValue{FromEnv: &password}},
				// Duplicate to confirm dedup.
				{Type: "select", Selector: "#tenant", Value: &FormStepValue{FromEnv: &tenant}},
				{Type: "select", Selector: "#tenant2", Value: &FormStepValue{FromEnv: &tenant}},
				{Type: "click", Selector: "button"},
			},
			Success: map[string]any{"type": "load"},
		},
	}

	got := CollectFromEnvReferences(auth)
	want := []string{password, user, tenant}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestValidateAuth_AcceptsFormFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.auth-form.json")

	if err := ValidateAuth(auth); err != nil {
		t.Fatalf("expected fixture to validate: %v", err)
	}
}

func TestValidateAuth_AcceptsStorageStateFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.auth-storage-state.json")

	if err := ValidateAuth(auth); err != nil {
		t.Fatalf("expected fixture to validate: %v", err)
	}
}

func TestValidateAuth_RejectsBadFromEnvName(t *testing.T) {
	t.Parallel()

	bad := "lower-case-not-allowed"
	auth := &Auth{
		Mode: AuthModeForm,
		Form: &FormRecipe{
			LoginURL: "https://x",
			Steps: []FormStep{
				{Type: "fill", Selector: "#a", Value: &FormStepValue{FromEnv: &bad}},
			},
			Success: map[string]any{"type": "load"},
		},
	}

	if err := ValidateAuth(auth); err == nil {
		t.Fatal("expected validation error for bad from_env name")
	}
}

func TestValidateAuth_RejectsStorageStateWithoutKeyOrBytes(t *testing.T) {
	t.Parallel()

	auth := &Auth{
		Mode:         AuthModeStorageState,
		StorageState: &StorageStateBlock{},
	}

	if err := ValidateAuth(auth); err == nil {
		t.Fatal("expected validation error for empty storage_state")
	}
}

func TestAuthRoundTripFromFormFixture(t *testing.T) {
	t.Parallel()

	auth := loadAuthFromFixture(t, "provenance.auth-form.json")
	if auth == nil {
		t.Fatal("nil auth")
	}

	// Marshal back to JSON and re-decode; must produce structurally identical
	// envelope. Using Compact() to assert no extra fields appear.
	wire, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundtrip Auth
	if unmarshalErr := json.Unmarshal(wire, &roundtrip); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	if !reflect.DeepEqual(CollectFromEnvReferences(auth), CollectFromEnvReferences(&roundtrip)) {
		t.Fatal("from_env refs differ after round-trip")
	}
}

func TestAuthCompactStripsContentB64(t *testing.T) {
	t.Parallel()

	auth := &Auth{
		Mode: AuthModeStorageState,
		StorageState: &StorageStateBlock{
			ArtifactKey:   "job-1/auth/storage-state.json",
			ContentBase64: "aGVsbG8gd29ybGQ=",
		},
	}

	out := auth.Compact()

	if _, hasContent := out["content_b64"]; hasContent {
		t.Fatalf("Compact() must strip content_b64; got %v", out)
	}

	if out["artifact_key"] != "job-1/auth/storage-state.json" {
		t.Fatalf("Compact() lost artifact_key; got %v", out)
	}
}
