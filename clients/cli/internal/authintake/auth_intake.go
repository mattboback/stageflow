// Auth intake helpers for `stageflow scan --auth-state` and
// `stageflow scan --auth-recipe`.
//
// These helpers read an auth-state JSON or a YAML/JSON form recipe from disk,
// validate it locally before sending to the platform-api, and shape it into the
// JobAuthInput payload the API expects. The platform-api repeats the schema
// validation server-side; the CLI's local validation is purely so we can fail
// fast with a developer-friendly message instead of a 400 from the API.
//
// Secrets-handling rules (mirroring docs/architecture/system.md#authenticated-scanning):
//
//   - Form recipes: values are either literal strings or {from_env: NAME}
//     references. The CLI never resolves those references; the orchestrator
//     forwards only the named env vars into the scanner-runner pod.
//   - Storage state: the captured Playwright JSON is base64-encoded and shipped
//     once over the wire, then uploaded to MinIO by the platform-api under the
//     job's prefix before job.created is published. It is never persisted to
//     Postgres; the orchestrator only keeps a defensive fallback for legacy
//     producers that still send inline bytes.
package authintake

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

// maxAuthStateBytes caps the storage-state payload at 1 MiB. Real Playwright
// storage-state files for typical authenticated apps are 5-50 KiB; anything
// larger is almost certainly a mistake (e.g. a wrong file path).
const maxAuthStateBytes = 1 << 20 // 1 MiB

// envVarNamePattern mirrors the JSON schema definition for {from_env: NAME}.
// Names must start with an uppercase letter and contain only A-Z, 0-9, _.
var envVarNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// LoadStateFile reads a Playwright storage-state JSON file, validates it
// parses as JSON and stays under the size cap, and returns it as a base64-encoded
// payload ready to ship in JobAuthInput.
func LoadStateFile(path string) (*apiclient.JobAuthInput, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, errors.New("--auth-state path must not be empty")
	}

	f, err := os.Open(trimmed)
	if err != nil {
		return nil, fmt.Errorf("open --auth-state %q: %w", trimmed, err)
	}
	defer func() { _ = f.Close() }()

	// Read up to maxAuthStateBytes+1 so we can detect oversize files cleanly.
	limited := io.LimitReader(f, int64(maxAuthStateBytes)+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read --auth-state %q: %w", trimmed, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf(
			"--auth-state file %q is empty; "+
				"capture a session first with `stageflow auth capture`",
			trimmed,
		)
	}

	if len(data) > maxAuthStateBytes {
		return nil, fmt.Errorf(
			"--auth-state file %q exceeds the %d byte limit; "+
				"this is almost always a bug (recapture with `stageflow auth capture`)",
			trimmed,
			maxAuthStateBytes,
		)
	}

	// Validate the file shape locally so we don't ship garbage. Playwright
	// writes objects with `cookies` and `origins` arrays; we only require valid
	// JSON object form so future Playwright versions remain compatible.
	var probe map[string]any
	if unmarshalErr := json.Unmarshal(data, &probe); unmarshalErr != nil {
		return nil, fmt.Errorf("--auth-state %q is not valid JSON: %w", trimmed, unmarshalErr)
	}

	enc := base64.StdEncoding.EncodeToString(data)

	return &apiclient.JobAuthInput{
		Mode: "storage_state",
		StorageState: &apiclient.JobAuthStorageStateInput{
			ContentBase64: enc,
		},
	}, nil
}

// LoadRecipeFile reads a YAML or JSON form-auth recipe, validates it
// locally against the Provenance.auth.form shape, and returns it as a
// JobAuthInput payload.
//
// The recipe must already match the schema in
// libs/contracts/provenance/schema/provenance.schema.json (ProvenanceAuthForm).
// The platform-api will re-validate; this CLI-side check exists only so the
// developer gets a fast, clear error before paying a network round-trip.
func LoadRecipeFile(path string) (*apiclient.JobAuthInput, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, errors.New("--auth-recipe path must not be empty")
	}

	data, err := os.ReadFile(trimmed)
	if err != nil {
		return nil, fmt.Errorf("read --auth-recipe %q: %w", trimmed, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("--auth-recipe %q is empty", trimmed)
	}

	parsed, err := decodeRecipeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse --auth-recipe %q: %w", trimmed, err)
	}

	recipe, err := buildFormRecipe(parsed)
	if err != nil {
		return nil, fmt.Errorf("invalid --auth-recipe %q: %w", trimmed, err)
	}

	return &apiclient.JobAuthInput{
		Mode: "form",
		Form: recipe,
	}, nil
}

// decodeRecipeBytes accepts YAML or JSON, returns a normalized
// map[string]any with string keys throughout. yaml.v3 already emits
// map[string]any so the conversion is a no-op for compliant input.
func decodeRecipeBytes(data []byte) (map[string]any, error) {
	var node map[string]any
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	if node == nil {
		return nil, errors.New("recipe is empty or null")
	}

	normalized, err := normalizeYAMLMap(node)
	if err != nil {
		return nil, err
	}

	return normalized, nil
}

// normalizeYAMLMap converts any nested map[any]any (which yaml.v2 used to
// produce; yaml.v3 keeps string keys but be defensive) into map[string]any.
// It returns an error if a non-string key is encountered.
func normalizeYAMLMap(in map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(in))
	for k, v := range in {
		nv, err := normalizeYAMLValue(v)
		if err != nil {
			return nil, err
		}

		out[k] = nv
	}

	return out, nil
}

func normalizeYAMLValue(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		return normalizeYAMLMap(t)
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string map key %v (type %T)", k, k)
			}

			nv, err := normalizeYAMLValue(vv)
			if err != nil {
				return nil, err
			}

			out[ks] = nv
		}

		return out, nil
	case []any:
		out := make([]any, len(t))

		for i, vv := range t {
			nv, err := normalizeYAMLValue(vv)
			if err != nil {
				return nil, err
			}

			out[i] = nv
		}

		return out, nil
	default:
		return v, nil
	}
}

func buildFormRecipe(parsed map[string]any) (*apiclient.JobAuthFormRecipe, error) {
	mode, ok := parsed["mode"].(string)
	if !ok {
		return nil, errors.New(`recipe.mode must be "form"`)
	}

	if mode != "form" {
		return nil, fmt.Errorf(`recipe.mode must be "form" (got %q)`, mode)
	}

	loginURL, ok := parsed["login_url"].(string)
	if !ok {
		return nil, errors.New("recipe.login_url is required")
	}

	if strings.TrimSpace(loginURL) == "" {
		return nil, errors.New("recipe.login_url is required")
	}

	stepsRaw, ok := parsed["steps"].([]any)
	if !ok {
		return nil, errors.New("recipe.steps must be a non-empty array")
	}

	if len(stepsRaw) == 0 {
		return nil, errors.New("recipe.steps must contain at least one step")
	}

	steps := make([]map[string]any, 0, len(stepsRaw))

	for i, raw := range stepsRaw {
		step, isMap := raw.(map[string]any)
		if !isMap {
			return nil, fmt.Errorf("recipe.steps[%d] must be an object", i)
		}

		if validateErr := validateStep(step); validateErr != nil {
			return nil, fmt.Errorf("recipe.steps[%d]: %w", i, validateErr)
		}

		steps = append(steps, step)
	}

	successRaw, ok := parsed["success"].(map[string]any)
	if !ok {
		return nil, errors.New("recipe.success must be an object describing a wait strategy")
	}

	if _, hasType := successRaw["type"].(string); !hasType {
		return nil, errors.New("recipe.success.type is required (load|domcontentloaded|networkidle|selector|timeout)")
	}

	return &apiclient.JobAuthFormRecipe{
		LoginURL: loginURL,
		Steps:    steps,
		Success:  successRaw,
	}, nil
}

// validateStep enforces the local-shape invariants that map cleanly to
// PreScanAction in the schema. The platform-api applies the canonical
// validation; we just want to catch obvious mistakes early.
func validateStep(step map[string]any) error {
	stepType, ok := step["type"].(string)
	if !ok || stepType == "" {
		return errors.New(`step.type is required (one of: click, fill, select, hover, wait, scroll, keyboard)`)
	}

	switch stepType {
	case "click", "fill", "select", "hover":
		if _, hasSel := step["selector"].(string); !hasSel {
			return fmt.Errorf("step type %q requires a selector", stepType)
		}
	case "wait":
		if _, hasMS := step["ms"]; !hasMS {
			return errors.New(`step type "wait" requires "ms"`)
		}
	case "scroll":
		// selector optional
	case "keyboard":
		if _, hasKey := step["key"].(string); !hasKey {
			return errors.New(`step type "keyboard" requires "key"`)
		}
	default:
		return fmt.Errorf("unknown step.type %q", stepType)
	}

	if stepType == "fill" || stepType == "select" {
		raw, hasValue := step["value"]
		if !hasValue {
			return fmt.Errorf("step type %q requires a value (literal string or {from_env: NAME})", stepType)
		}

		if err := validateActionValue(raw); err != nil {
			return err
		}
	}

	return nil
}

func validateActionValue(raw any) error {
	switch v := raw.(type) {
	case string:
		return nil
	case map[string]any:
		name, ok := v["from_env"].(string)
		if !ok {
			return errors.New(`step.value object must be {from_env: NAME}`)
		}

		if !envVarNamePattern.MatchString(name) {
			return fmt.Errorf(
				`step.value.from_env %q must match ^[A-Z][A-Z0-9_]*$ `+
					`(uppercase letter first, then uppercase/digits/underscores)`,
				name,
			)
		}

		// Reject any extra keys to mirror the schema's additionalProperties:false.
		for k := range v {
			if k != "from_env" {
				return fmt.Errorf(`step.value object has unexpected key %q (only "from_env" is allowed)`, k)
			}
		}

		return nil
	default:
		return fmt.Errorf("step.value must be a string or {from_env: NAME} object (got %T)", raw)
	}
}
