package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name string, contents string) string {
	t.Helper()

	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}

	return p
}

func TestLoadAuthStateFile_HappyPath(t *testing.T) {
	tmp := t.TempDir()
	body := `{"cookies":[{"name":"sid","value":"abc","domain":".example.com","path":"/","expires":-1,"httpOnly":true,"secure":true,"sameSite":"Lax"}],"origins":[]}`
	path := writeFile(t, tmp, "state.json", body)

	got, err := loadAuthStateFile(path)
	requireNoErr(t, err)

	requireEqual(t, got.Mode, "storage_state", "mode")

	if got.StorageState == nil {
		t.Fatalf("StorageState is nil")
	}

	decoded, err := base64.StdEncoding.DecodeString(got.StorageState.ContentBase64)
	requireNoErr(t, err)

	if string(decoded) != body {
		t.Fatalf("decoded body mismatch:\n got=%q\nwant=%q", string(decoded), body)
	}

	if got.Form != nil {
		t.Fatalf("expected Form to be nil for storage_state mode")
	}
}

func TestLoadAuthStateFile_RejectsMissing(t *testing.T) {
	_, err := loadAuthStateFile(filepath.Join(t.TempDir(), "no-such-file.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadAuthStateFile_RejectsEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := writeFile(t, tmp, "empty.json", "")

	_, err := loadAuthStateFile(path)
	if err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}

func TestLoadAuthStateFile_RejectsNonJSON(t *testing.T) {
	tmp := t.TempDir()
	path := writeFile(t, tmp, "bad.json", "not json at all")

	_, err := loadAuthStateFile(path)
	if err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("expected JSON error, got %v", err)
	}
}

func TestLoadAuthStateFile_RejectsOversize(t *testing.T) {
	tmp := t.TempDir()
	bigCookie := strings.Repeat("x", maxAuthStateBytes+1024)
	body := `{"cookies":[],"origins":[],"pad":"` + bigCookie + `"}`
	path := writeFile(t, tmp, "huge.json", body)

	_, err := loadAuthStateFile(path)
	if err == nil || !strings.Contains(err.Error(), "exceeds the") {
		t.Fatalf("expected size-limit error, got %v", err)
	}
}

func TestLoadAuthRecipeFile_YAMLForm(t *testing.T) {
	tmp := t.TempDir()
	yaml := `
mode: form
login_url: https://app.example.com/login
steps:
  - type: fill
    selector: input[name=email]
    value:
      from_env: STAGEFLOW_AUTH_USER
  - type: fill
    selector: input[name=password]
    value:
      from_env: STAGEFLOW_AUTH_PASSWORD
  - type: click
    selector: button[type=submit]
success:
  type: selector
  selector: '[data-test=signed-in]'
  timeout: 15000
`

	path := writeFile(t, tmp, "recipe.yaml", yaml)

	got, err := loadAuthRecipeFile(path)
	requireNoErr(t, err)

	requireEqual(t, got.Mode, "form", "mode")

	if got.Form == nil {
		t.Fatal("Form is nil")
	}

	if got.Form.LoginURL != "https://app.example.com/login" {
		t.Fatalf("login_url = %q", got.Form.LoginURL)
	}

	if len(got.Form.Steps) != 3 {
		t.Fatalf("steps len = %d, want 3", len(got.Form.Steps))
	}

	emailStep := got.Form.Steps[0]
	if emailStep["type"] != "fill" {
		t.Fatalf("steps[0].type = %v, want fill", emailStep["type"])
	}

	emailValue, ok := emailStep["value"].(map[string]any)
	if !ok {
		t.Fatalf("steps[0].value not a map: %T", emailStep["value"])
	}

	if emailValue["from_env"] != "STAGEFLOW_AUTH_USER" {
		t.Fatalf("from_env = %v", emailValue["from_env"])
	}

	// Round-trip through JSON to confirm it survives the wire shape.
	wire, err := json.Marshal(got)
	requireNoErr(t, err)

	if !strings.Contains(string(wire), `"from_env":"STAGEFLOW_AUTH_USER"`) {
		t.Fatalf("from_env reference missing from JSON: %s", wire)
	}
}

func TestLoadAuthRecipeFile_JSONForm(t *testing.T) {
	tmp := t.TempDir()
	body := `{
  "mode": "form",
  "login_url": "https://demo.local/login",
  "steps": [
    { "type": "fill", "selector": "input[name=email]", "value": "demo@example.com" },
    { "type": "click", "selector": "button[type=submit]" }
  ],
  "success": { "type": "load" }
}`
	path := writeFile(t, tmp, "recipe.json", body)

	got, err := loadAuthRecipeFile(path)
	requireNoErr(t, err)

	requireEqual(t, got.Mode, "form", "mode")

	if got.Form.Steps[0]["value"] != "demo@example.com" {
		t.Fatalf("expected literal string value, got %v", got.Form.Steps[0]["value"])
	}
}

func TestLoadAuthRecipeFile_RejectsMissingMode(t *testing.T) {
	tmp := t.TempDir()
	path := writeFile(t, tmp, "recipe.yaml", "login_url: https://x\nsteps: []\nsuccess: {type: load}\n")

	_, err := loadAuthRecipeFile(path)
	if err == nil || !strings.Contains(err.Error(), `mode must be "form"`) {
		t.Fatalf("expected mode error, got %v", err)
	}
}

func TestLoadAuthRecipeFile_RejectsBadEnvVarName(t *testing.T) {
	tmp := t.TempDir()
	body := `mode: form
login_url: https://x
steps:
  - type: fill
    selector: '#email'
    value:
      from_env: lower_case_not_allowed
success:
  type: load
`
	path := writeFile(t, tmp, "recipe.yaml", body)

	_, err := loadAuthRecipeFile(path)
	if err == nil || !strings.Contains(err.Error(), "from_env") {
		t.Fatalf("expected from_env validation error, got %v", err)
	}
}

func TestLoadAuthRecipeFile_RejectsExtraValueKey(t *testing.T) {
	tmp := t.TempDir()
	body := `mode: form
login_url: https://x
steps:
  - type: fill
    selector: '#email'
    value:
      from_env: USER
      literal: surprise
success:
  type: load
`
	path := writeFile(t, tmp, "recipe.yaml", body)

	_, err := loadAuthRecipeFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected key") {
		t.Fatalf("expected unexpected-key error, got %v", err)
	}
}

func TestLoadAuthRecipeFile_RejectsEmptySteps(t *testing.T) {
	tmp := t.TempDir()
	body := `mode: form
login_url: https://x
steps: []
success:
  type: load
`
	path := writeFile(t, tmp, "recipe.yaml", body)

	_, err := loadAuthRecipeFile(path)
	if err == nil || !strings.Contains(err.Error(), "at least one step") {
		t.Fatalf("expected empty-steps error, got %v", err)
	}
}

func TestLoadAuthInputFromFlags_MutuallyExclusive(t *testing.T) {
	_, err := loadAuthInputFromFlags("/x", "/y")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually-exclusive error, got %v", err)
	}
}

func TestLoadAuthInputFromFlags_NeitherSet(t *testing.T) {
	got, err := loadAuthInputFromFlags("", "")
	requireNoErr(t, err)

	if got != nil {
		t.Fatalf("expected nil JobAuthInput when no flags set, got %#v", got)
	}
}
