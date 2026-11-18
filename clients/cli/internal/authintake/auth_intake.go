// Package authintake reads CLI authentication files and resolves recipe values
// from the caller's environment. The API validates authentication semantics.
package authintake

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

const maxAuthFileBytes = 1 << 20 // 1 MiB, matching the API storage-state limit.

// LoadStateFile encodes the captured bytes; the API validates the storage state.
func LoadStateFile(path string) (*apiclient.JobAuthInput, error) {
	data, err := readAuthFile(path, "--auth-state")
	if err != nil {
		return nil, err
	}

	return &apiclient.JobAuthInput{
		Mode: "storage_state",
		StorageState: &apiclient.JobAuthStorageStateInput{
			ContentBase64: base64.StdEncoding.EncodeToString(data),
		},
	}, nil
}

// LoadRecipeFile accepts YAML or JSON and replaces environment references with
// literal strings before submission. It never modifies the source file.
func LoadRecipeFile(path string) (*apiclient.JobAuthInput, error) {
	data, err := readAuthFile(path, "--auth-recipe")
	if err != nil {
		return nil, err
	}

	var parsed map[string]any
	if decodeErr := yaml.Unmarshal(data, &parsed); decodeErr != nil {
		// YAML errors can quote source lines containing credentials.
		return nil, errors.New("cannot decode --auth-recipe YAML/JSON")
	}

	// Check before the JSON round trip, which turns a YAML timestamp into a string.
	if steps, ok := parsed["steps"].([]any); ok {
		for i, raw := range steps {
			if step, isMap := raw.(map[string]any); isMap {
				if scalarErr := requireStringScalar(step); scalarErr != nil {
					return nil, fmt.Errorf("--auth-recipe steps[%d]: %w", i, scalarErr)
				}
			}
		}
	}

	wire, err := json.Marshal(parsed)
	if err != nil {
		return nil, errors.New("cannot encode --auth-recipe as JSON")
	}

	var recipe struct {
		Mode string `json:"mode"`
		apiclient.JobAuthFormRecipe
	}
	if decodeErr := json.Unmarshal(wire, &recipe); decodeErr != nil {
		return nil, errors.New("cannot decode --auth-recipe fields")
	}

	for i, step := range recipe.Steps {
		if resolveErr := resolveStepValue(step); resolveErr != nil {
			return nil, fmt.Errorf("--auth-recipe steps[%d]: %w", i, resolveErr)
		}
	}

	return &apiclient.JobAuthInput{Mode: recipe.Mode, Form: &recipe.JobAuthFormRecipe}, nil
}

func resolveStepValue(step map[string]any) error {
	reference, ok := step["value"].(map[string]any)
	if !ok {
		return nil
	}

	rawName, hasReference := reference["from_env"]
	if !hasReference {
		return nil
	}

	name, ok := rawName.(string)
	if !ok || name == "" || len(reference) != 1 {
		return errors.New("value must contain only a nonempty from_env name")
	}

	value, present := os.LookupEnv(name)
	if !present {
		return fmt.Errorf("from_env variable %q is unset", name)
	}

	step["value"] = value

	return nil
}

// requireStringScalar rejects the scalars YAML decodes as numbers, booleans,
// timestamps or null. Unquoted, 00123 would be sent as 83 and a date would be
// rewritten, so the scanner would type something the caller did not write.
func requireStringScalar(step map[string]any) error {
	value, hasValue := step["value"]
	if !hasValue {
		return nil
	}

	switch value.(type) {
	case string, map[string]any, []any:
		return nil
	default:
		return errors.New("value must be a string; quote it in YAML")
	}
}

func readAuthFile(path, flag string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("%s path must not be empty", flag)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s file: %w", flag, err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxAuthFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s file: %w", flag, err)
	}

	if len(data) > maxAuthFileBytes {
		return nil, fmt.Errorf("%s file exceeds the %d byte limit", flag, maxAuthFileBytes)
	}

	return data, nil
}
