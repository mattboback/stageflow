// Package provenance is a small Go-side mirror of the parts of the Provenance
// contract the orchestrator and platform-api need to reason about: the auth
// block and its from_env references.
//
// The canonical schema lives at libs/contracts/provenance/schema/provenance.schema.json
// and the scanner-runner consumes it through its TypeScript types and
// secrets-resolver.ts. The collectFromEnvReferences algorithm here is a direct
// port of the TypeScript implementation in
// services/scanner-runner/src/core/secrets-resolver.ts; both implementations
// must produce identical output for the same Provenance and are kept in sync by
// shared fixtures (see fixtures-driven tests under tests/contracts/).
package provenance

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
)

// AuthMode is the discriminator on Provenance.auth.
const (
	AuthModeForm         = "form"
	AuthModeStorageState = "storage_state"
)

// envVarNamePattern mirrors PreScanActionValue.from_env in the JSON schema.
var envVarNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// FromEnvReference is the {from_env: NAME} reference shape used inside step
// values.
type FromEnvReference struct {
	FromEnv string `json:"from_env"`
}

// Auth is the optional authentication block on a Provenance document. Exactly
// one of Form or StorageState is non-nil; Mode is the discriminator.
type Auth struct {
	Mode         string             `json:"mode"`
	Form         *FormRecipe        `json:"-"`
	StorageState *StorageStateBlock `json:"-"`
}

// FormRecipe is the declarative login replay carried inside Provenance.auth
// when Mode == AuthModeForm.
type FormRecipe struct {
	LoginURL string         `json:"login_url"`
	Steps    []FormStep     `json:"steps"`
	Success  map[string]any `json:"success"`
}

// FormStep is one step in a login replay. It mirrors PreScanAction in the
// schema. Type-specific fields are kept generic so the scanner-runner remains
// the single source of truth for action semantics; the orchestrator only needs
// Type and Value to derive the env-var allow-list.
type FormStep struct {
	Type     string         `json:"type"`
	Selector string         `json:"selector,omitempty"`
	Value    *FormStepValue `json:"value,omitempty"`
	Timeout  *int           `json:"timeout,omitempty"`
	Ms       *int           `json:"ms,omitempty"`
	Key      string         `json:"key,omitempty"`
	// Direction and Pixels are scroll-only; carried as raw to round-trip
	// without changing the schema.
	Direction string `json:"direction,omitempty"`
	Pixels    *int   `json:"pixels,omitempty"`
}

// FormStepValue is either a literal string or a {from_env: NAME} reference.
// Exactly one of Literal or FromEnv is set when the step has a value.
type FormStepValue struct {
	Literal *string
	FromEnv *string
}

// StorageStateBlock is the storage_state branch of Provenance.auth. The
// platform-api uploads a captured Playwright JSON to MinIO and writes its key
// here before publishing job.created; the orchestrator retains a legacy
// defensive uploader for older producers. ContentBase64 is only set on the
// CLI-to-platform wire, never on persisted Provenance.
type StorageStateBlock struct {
	ArtifactKey   string `json:"artifact_key,omitempty"`
	ContentBase64 string `json:"content_b64,omitempty"`
}

// Compact returns a wire-safe representation suitable for storage in Postgres
// or for serialization into PROVENANCE_AUTH_JSON. For storage_state, the
// returned map omits any inline content_b64 so the bytes never leak past the
// orchestrator's upload step.
func (a *Auth) Compact() map[string]any {
	if a == nil {
		return nil
	}

	switch a.Mode {
	case AuthModeStorageState:
		out := map[string]any{"mode": AuthModeStorageState}
		if a.StorageState != nil {
			out["artifact_key"] = a.StorageState.ArtifactKey
		}

		return out
	case AuthModeForm:
		out := map[string]any{"mode": AuthModeForm}
		if a.Form == nil {
			return out
		}

		out["login_url"] = a.Form.LoginURL
		out["steps"] = stepsToWire(a.Form.Steps)
		out["success"] = a.Form.Success

		return out
	default:
		return map[string]any{"mode": a.Mode}
	}
}

func stepsToWire(steps []FormStep) []map[string]any {
	out := make([]map[string]any, len(steps))

	for i, step := range steps {
		entry := map[string]any{"type": step.Type}

		if step.Selector != "" {
			entry["selector"] = step.Selector
		}

		if step.Value != nil {
			switch {
			case step.Value.FromEnv != nil:
				entry["value"] = map[string]any{"from_env": *step.Value.FromEnv}
			case step.Value.Literal != nil:
				entry["value"] = *step.Value.Literal
			}
		}

		if step.Timeout != nil {
			entry["timeout"] = *step.Timeout
		}

		if step.Ms != nil {
			entry["ms"] = *step.Ms
		}

		if step.Key != "" {
			entry["key"] = step.Key
		}

		if step.Direction != "" {
			entry["direction"] = step.Direction
		}

		if step.Pixels != nil {
			entry["pixels"] = *step.Pixels
		}

		out[i] = entry
	}

	return out
}

// CollectFromEnvReferences walks the auth block and returns a deduplicated,
// sorted list of env-var names referenced via {from_env: NAME}. The result is
// the allow-list the orchestrator forwards into the scanner-runner pod.
//
// The algorithm is a direct port of collectFromEnvReferences from
// services/scanner-runner/src/core/secrets-resolver.ts:
//
//   - storage_state mode produces an empty list (no env refs).
//   - form mode walks every step's value field; only fill and select have a
//     value field, but FormStepValue is empty for other types so they are
//     ignored uniformly.
//   - Names are deduplicated and sorted ascii-ascending.
func CollectFromEnvReferences(auth *Auth) []string {
	if auth == nil || auth.Mode != AuthModeForm || auth.Form == nil {
		return nil
	}

	seen := make(map[string]struct{})

	for _, step := range auth.Form.Steps {
		if step.Value == nil || step.Value.FromEnv == nil {
			continue
		}

		// fill and select are the only step types with a value in the schema,
		// but defensive: any step that carries a from_env ref counts.
		if step.Type != "" && step.Type != "fill" && step.Type != "select" {
			continue
		}

		seen[*step.Value.FromEnv] = struct{}{}
	}

	if len(seen) == 0 {
		return nil
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// ValidateAuth performs the structural checks the orchestrator needs before
// using the block to drive a pod launch. It is intentionally narrower than the
// canonical JSON schema (the platform-api applies that one), checking only the
// invariants the orchestrator relies on.
func ValidateAuth(auth *Auth) error {
	if auth == nil {
		return nil
	}

	switch auth.Mode {
	case AuthModeStorageState:
		return validateStorageStateAuth(auth.StorageState)
	case AuthModeForm:
		return validateFormAuth(auth.Form)
	case "":
		return errors.New("provenance.auth.mode is required")
	default:
		return fmt.Errorf("provenance.auth.mode %q is not supported (expected %q or %q)",
			auth.Mode, AuthModeForm, AuthModeStorageState)
	}
}

func validateStorageStateAuth(storageState *StorageStateBlock) error {
	if storageState == nil {
		return errors.New("provenance.auth.storage_state is required when mode=storage_state")
	}

	if storageState.ArtifactKey == "" && storageState.ContentBase64 == "" {
		return errors.New(
			"provenance.auth.storage_state must carry either an artifact_key or an inline content_b64 blob",
		)
	}

	return nil
}

func validateFormAuth(form *FormRecipe) error {
	if form == nil {
		return errors.New("provenance.auth.form is required when mode=form")
	}

	if form.LoginURL == "" {
		return errors.New("provenance.auth.form.login_url is required")
	}

	if len(form.Steps) == 0 {
		return errors.New("provenance.auth.form.steps must contain at least one step")
	}

	if form.Success == nil {
		return errors.New("provenance.auth.form.success is required")
	}

	if _, hasType := form.Success["type"].(string); !hasType {
		return errors.New("provenance.auth.form.success.type is required")
	}

	for i, step := range form.Steps {
		if err := validateFormAuthStep(i, step); err != nil {
			return err
		}
	}

	return nil
}

func validateFormAuthStep(index int, step FormStep) error {
	if step.Type == "" {
		return fmt.Errorf("provenance.auth.form.steps[%d].type is required", index)
	}

	if step.Value == nil || step.Value.FromEnv == nil {
		return nil
	}

	if envVarNamePattern.MatchString(*step.Value.FromEnv) {
		return nil
	}

	return fmt.Errorf(
		"provenance.auth.form.steps[%d].value.from_env %q must match %s",
		index,
		*step.Value.FromEnv,
		envVarNamePattern.String(),
	)
}

// MarshalJSON encodes Auth so it round-trips with the JSON schema shape: a flat
// object with `mode` plus the form/storage_state branch fields. The TS shape
// embeds the branch directly in the auth object, which we mirror by emitting
// the form fields at the top level when Mode == form.
func (a Auth) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Compact())
}

// UnmarshalJSON decodes the schema's flat shape ({mode, login_url, steps,
// success} or {mode, artifact_key, content_b64}) into the typed Auth struct.
func (a *Auth) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	modeRaw, ok := raw["mode"]
	if !ok {
		return errors.New("provenance.auth: missing mode")
	}

	if err := json.Unmarshal(modeRaw, &a.Mode); err != nil {
		return fmt.Errorf("provenance.auth: invalid mode: %w", err)
	}

	switch a.Mode {
	case AuthModeStorageState:
		return a.decodeStorageStateAuth(raw)
	case AuthModeForm:
		return a.decodeFormAuth(raw)
	default:
		return fmt.Errorf("provenance.auth.mode %q is not supported", a.Mode)
	}
}

func (a *Auth) decodeStorageStateAuth(raw map[string]json.RawMessage) error {
	ss := &StorageStateBlock{}
	if v, has := raw["artifact_key"]; has {
		if err := json.Unmarshal(v, &ss.ArtifactKey); err != nil {
			return fmt.Errorf("provenance.auth.artifact_key: %w", err)
		}
	}

	if v, has := raw["content_b64"]; has {
		if err := json.Unmarshal(v, &ss.ContentBase64); err != nil {
			return fmt.Errorf("provenance.auth.content_b64: %w", err)
		}
	}

	a.StorageState = ss

	return nil
}

func (a *Auth) decodeFormAuth(raw map[string]json.RawMessage) error {
	form := &FormRecipe{}

	if v, has := raw["login_url"]; has {
		if err := json.Unmarshal(v, &form.LoginURL); err != nil {
			return fmt.Errorf("provenance.auth.login_url: %w", err)
		}
	}

	if v, has := raw["steps"]; has {
		if err := json.Unmarshal(v, &form.Steps); err != nil {
			return fmt.Errorf("provenance.auth.steps: %w", err)
		}
	}

	if v, has := raw["success"]; has {
		if err := json.Unmarshal(v, &form.Success); err != nil {
			return fmt.Errorf("provenance.auth.success: %w", err)
		}
	}

	a.Form = form

	return nil
}

// MarshalJSON for FormStepValue emits either a literal string or a
// {from_env: NAME} object, matching the schema's PreScanActionValue oneOf.
func (v FormStepValue) MarshalJSON() ([]byte, error) {
	switch {
	case v.FromEnv != nil:
		return json.Marshal(map[string]string{"from_env": *v.FromEnv})
	case v.Literal != nil:
		return json.Marshal(*v.Literal)
	default:
		return []byte("null"), nil
	}
}

// UnmarshalJSON for FormStepValue accepts either a string or a
// {from_env: NAME} object.
func (v *FormStepValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		v.Literal = &s

		return nil
	}

	var ref FromEnvReference
	if err := json.Unmarshal(data, &ref); err != nil {
		return err
	}

	if ref.FromEnv == "" {
		return errors.New("provenance.auth step value: from_env must not be empty")
	}

	v.FromEnv = &ref.FromEnv

	return nil
}
