package authintake

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	testsupport.RequireNoErr(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func TestLoadStateFile_PreservesBytesForServerValidation(t *testing.T) {
	for _, body := range []string{`{"cookies":[{"value":"state-secret"}],"origins":[]}`, "invalid JSON", ""} {
		path := writeFile(t, t.TempDir(), "state.json", body)
		got, err := LoadStateFile(path)
		testsupport.RequireNoErr(t, err)
		testsupport.RequireEqual(t, got.Mode, "storage_state", "mode")

		decoded, err := base64.StdEncoding.DecodeString(got.StorageState.ContentBase64)
		testsupport.RequireNoErr(t, err)
		testsupport.RequireEqual(t, string(decoded), body, "storage state bytes")
	}
}

func TestLoadAuthFiles_BoundedRead(t *testing.T) {
	for _, state := range []bool{false, true} {
		for _, path := range []string{
			"",
			filepath.Join(t.TempDir(), "missing"),
			writeFile(t, t.TempDir(), "oversize", strings.Repeat("x", maxAuthFileBytes+1)),
		} {
			var err error
			if state {
				_, err = LoadStateFile(path)
			} else {
				_, err = LoadRecipeFile(path)
			}

			if err == nil {
				t.Fatal("expected file intake failure")
			}
		}
	}
}

func TestLoadRecipeFile_ResolvesYAMLAndJSONExactly(t *testing.T) {
	const secret = "  password\n雪 ${OTHER_SECRET} &+%  "
	t.Setenv("STAGEFLOW_AUTH_SECRET", secret)
	t.Setenv("STAGEFLOW_AUTH_EMPTY", "")

	for _, body := range []string{
		`mode: form
login_url: https://app.example.com/login
steps:
  - {type: fill, selector: '#password', value: {from_env: STAGEFLOW_AUTH_SECRET}}
  - {type: select, selector: '#empty', value: {from_env: STAGEFLOW_AUTH_EMPTY}}
  - {type: click, selector: '#go', value: {from_env: STAGEFLOW_AUTH_SECRET}}
  - {type: fill, selector: '#literal', value: '  $STAGEFLOW_AUTH_SECRET  '}
success: {type: load}
`,
		`{"mode":"form","login_url":"https://app.example.com/login","steps":[
{"type":"fill","selector":"#password","value":{"from_env":"STAGEFLOW_AUTH_SECRET"}},
{"type":"select","selector":"#empty","value":{"from_env":"STAGEFLOW_AUTH_EMPTY"}},
{"type":"click","selector":"#go","value":{"from_env":"STAGEFLOW_AUTH_SECRET"}},
{"type":"fill","selector":"#literal","value":"  $STAGEFLOW_AUTH_SECRET  "}],"success":{"type":"load"}}`,
	} {
		path := writeFile(t, t.TempDir(), "recipe", body)
		got, err := LoadRecipeFile(path)
		testsupport.RequireNoErr(t, err)
		testsupport.RequireEqual(t, got.Mode, "form", "mode")
		testsupport.RequireEqual(t, got.Form.LoginURL, "https://app.example.com/login", "login URL")

		for i, want := range []string{secret, "", secret, "  $STAGEFLOW_AUTH_SECRET  "} {
			if got.Form.Steps[i]["value"] != want {
				t.Fatalf("step %d literal was changed", i)
			}
		}

		wire, err := json.Marshal(got)
		testsupport.RequireNoErr(t, err)

		if strings.Contains(string(wire), "from_env") {
			t.Fatal("environment reference survived serialization")
		}

		original, err := os.ReadFile(path)
		testsupport.RequireNoErr(t, err)
		testsupport.RequireEqual(t, string(original), body, "source recipe")
	}
}

func TestLoadRecipeFile_ServerOwnsSemanticValidation(t *testing.T) {
	for _, body := range []string{
		`{"login_url":"https://x","steps":[],"success":{"type":"load"}}`,
		`{"mode":"unknown","steps":[{"type":"unknown","value":"12"}]}`,
		`{"mode":"form","steps":[{"type":"fill","value":{"unknown":"value"}}]}`,
	} {
		path := writeFile(t, t.TempDir(), "recipe.json", body)
		got, err := LoadRecipeFile(path)
		testsupport.RequireNoErr(t, err)

		var original struct {
			Mode string `json:"mode"`
		}
		testsupport.RequireNoErr(t, json.Unmarshal([]byte(body), &original))
		testsupport.RequireEqual(t, got.Mode, original.Mode, "mode passed to server")
	}
}

func TestLoadRecipeFile_SafeResolutionErrors(t *testing.T) {
	const canary = "private-recipe-canary"
	t.Setenv("STAGEFLOW_AUTH_UNUSED", canary)
	t.Setenv("STAGEFLOW_AUTH_MISSING_TEST", "")
	testsupport.RequireNoErr(t, os.Unsetenv("STAGEFLOW_AUTH_MISSING_TEST"))

	for _, body := range []string{
		`steps: [{value: {from_env: STAGEFLOW_AUTH_MISSING_TEST}}]`,
		`steps: [{value: {from_env: "", extra: ` + canary + `}}]`,
		`steps: [{value: {from_env: [` + canary + `]}}]`,
		`steps: [{value: {from_env: STAGEFLOW_AUTH_UNUSED, extra: ` + canary + `}}]`,
		`steps: [` + canary,
		`steps: ` + canary,
		`login_url: {` + canary + `: value}`,
		`steps: [{value: {123: ` + canary + `}}]`,
	} {
		path := writeFile(t, t.TempDir(), "recipe", body)

		_, err := LoadRecipeFile(path)
		if err == nil {
			t.Fatal("expected decode/resolution failure")
		}

		if strings.Contains(err.Error(), canary) {
			t.Fatal("error disclosed recipe content")
		}
	}
}

func TestLoadRecipeFile_RejectsCoercedScalars(t *testing.T) {
	for _, value := range []string{"123456", "00123", "1e3", "true", "2024-01-01", "null", "~"} {
		path := writeFile(t, t.TempDir(), "recipe.yaml", "mode: form\nsteps:\n  - {type: fill, value: "+value+"}\n")

		_, err := LoadRecipeFile(path)
		if err == nil {
			t.Fatalf("unquoted %s: expected a string-value error", value)
		}

		if !strings.Contains(err.Error(), "steps[0]") || strings.Contains(err.Error(), value) {
			t.Fatalf("unquoted %s: unexpected error %q", value, err)
		}
	}

	for _, value := range []string{"'123456'", `"00123"`, "'2024-01-01'", "plain-text"} {
		path := writeFile(t, t.TempDir(), "recipe.yaml", "mode: form\nsteps:\n  - {type: fill, value: "+value+"}\n")

		got, err := LoadRecipeFile(path)
		testsupport.RequireNoErr(t, err)
		testsupport.RequireEqual(t, got.Form.Steps[0]["value"], any(strings.Trim(value, `'"`)), "quoted value")
	}
}

func TestLoadRecipeFile_UnsetErrorNamesVariable(t *testing.T) {
	t.Setenv("STAGEFLOW_AUTH_NAMED_TEST", "")
	testsupport.RequireNoErr(t, os.Unsetenv("STAGEFLOW_AUTH_NAMED_TEST"))

	path := writeFile(t, t.TempDir(), "recipe.yaml", "steps: [{value: {from_env: STAGEFLOW_AUTH_NAMED_TEST}}]\n")

	_, err := LoadRecipeFile(path)
	if err == nil || !strings.Contains(err.Error(), "STAGEFLOW_AUTH_NAMED_TEST") {
		t.Fatalf("expected the unset variable to be named, got %v", err)
	}
}

func TestLoadFromFlags(t *testing.T) {
	_, _, err := LoadFromFlags("/x", "/y")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatal("expected mutually-exclusive error")
	}

	got, hasAuth, err := LoadFromFlags("", "")
	testsupport.RequireNoErr(t, err)

	if hasAuth || got != nil {
		t.Fatal("expected no authentication without flags")
	}
}
