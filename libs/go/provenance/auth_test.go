package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// loadAuthFromFixture reads a Provenance fixture and returns its auth block.
// The fixtures use the canonical contract shape.
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
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root (go.mod) not found from cwd")
		}

		dir = parent
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

	if !reflect.DeepEqual(auth.Compact(), roundtrip.Compact()) {
		t.Fatal("auth fields differ after round-trip")
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
