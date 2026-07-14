package authintake

import (
	"errors"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

// LoadFromFlags resolves --auth-state or --auth-recipe into a JobAuthInput.
func LoadFromFlags(authStatePath, authRecipePath string) (*apiclient.JobAuthInput, bool, error) {
	if authStatePath != "" && authRecipePath != "" {
		return nil, false, errors.New("--auth-state and --auth-recipe are mutually exclusive")
	}

	if authStatePath != "" {
		input, err := LoadStateFile(authStatePath)
		return input, err == nil, err
	}
	if authRecipePath != "" {
		input, err := LoadRecipeFile(authRecipePath)
		return input, err == nil, err
	}

	return nil, false, nil
}
