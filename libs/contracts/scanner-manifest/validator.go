package scannermanifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"

	_ "embed"
)

//go:embed schema/scanner-manifest.schema.json
var manifestSchema []byte

var (
	schemaOnce    sync.Once
	compiled      *jsonschema.Schema
	compiledError error
)

func compiledSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("scanner-manifest.schema.json", bytes.NewReader(manifestSchema)); err != nil {
			compiledError = fmt.Errorf("add manifest schema: %w", err)

			return
		}

		schema, err := compiler.Compile("scanner-manifest.schema.json")
		if err != nil {
			compiledError = fmt.Errorf("compile manifest schema: %w", err)

			return
		}

		compiled = schema
	})

	if compiledError != nil {
		return nil, compiledError
	}

	return compiled, nil
}

// ValidateManifestJSON validates a manifest JSON blob against the shared schema.
// It also verifies that configSchema (when present) is a valid JSON Schema.
func ValidateManifestJSON(data []byte) error {
	schema, err := compiledSchema()
	if err != nil {
		return err
	}

	var payload any
	if unmarshalErr := json.Unmarshal(data, &payload); unmarshalErr != nil {
		return fmt.Errorf("decode manifest: %w", unmarshalErr)
	}

	if validateErr := schema.Validate(payload); validateErr != nil {
		return fmt.Errorf("manifest schema validation failed: %w", validateErr)
	}

	if configErr := validateConfigSchema(data); configErr != nil {
		return fmt.Errorf("manifest configSchema validation failed: %w", configErr)
	}

	if capErr := validateCapabilitiesRules(data); capErr != nil {
		return fmt.Errorf("manifest capabilities validation failed: %w", capErr)
	}

	return nil
}

func validateConfigSchema(data []byte) error {
	var payload struct {
		ConfigSchema json.RawMessage `json:"configSchema"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}

	if len(payload.ConfigSchema) == 0 {
		return nil
	}

	var boolValue bool
	if err := json.Unmarshal(payload.ConfigSchema, &boolValue); err == nil {
		return nil
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("configSchema.json", bytes.NewReader(payload.ConfigSchema)); err != nil {
		return fmt.Errorf("add configSchema: %w", err)
	}

	if _, err := compiler.Compile("configSchema.json"); err != nil {
		return fmt.Errorf("compile configSchema: %w", err)
	}

	return nil
}

func validateCapabilitiesRules(data []byte) error {
	var payload struct {
		Capabilities struct {
			SupportsScreenshots bool `json:"supportsScreenshots"`
			RequiresBrowser     bool `json:"requiresBrowser"`
		} `json:"capabilities"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}

	if payload.Capabilities.SupportsScreenshots && !payload.Capabilities.RequiresBrowser {
		return errors.New("capabilities.supportsScreenshots requires capabilities.requiresBrowser to be true")
	}

	return nil
}
