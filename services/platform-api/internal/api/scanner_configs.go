package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

const scannerIDAiNavigator = "ai-navigator"
const aiNavigatorDefaultModelEnv = "AI_NAVIGATOR_DEFAULT_MODEL"

func validateScannerConfigs(modules []string, scannerConfigs map[string]map[string]any) *httputil.ErrorDetail {
	for _, module := range modules {
		if strings.EqualFold(module, scannerIDAiNavigator) {
			return validateAiNavigatorScannerConfig(scannerConfigs)
		}
	}

	return nil
}

func validateAiNavigatorScannerConfig(scannerConfigs map[string]map[string]any) *httputil.ErrorDetail {
	config := scannerConfigs[scannerIDAiNavigator]
	if len(config) == 0 {
		detail := httputil.NewValidationError(
			"scanner_configs.ai-navigator",
			"ai-navigator requires scanner configuration",
			`Include scanner_configs for ai-navigator, e.g. {"ai-navigator":{"goal":{"objective":"Reach checkout"},"vision":{"model":"openai/gpt-4o-mini"}}}`,
		)

		return &detail
	}

	goal, ok := asObject(config["goal"])
	if !ok {
		detail := httputil.NewValidationError(
			"scanner_configs.ai-navigator.goal",
			"ai-navigator requires goal (object)",
			`Provide {"goal":{"objective":"..."}} under scanner_configs.ai-navigator`,
		)

		return &detail
	}

	if objective, hasObjective := goal["objective"].(string); !hasObjective || strings.TrimSpace(objective) == "" {
		detail := httputil.NewValidationError(
			"scanner_configs.ai-navigator.goal.objective",
			"ai-navigator requires goal.objective (string)",
			`Provide a non-empty objective, e.g. {"goal":{"objective":"Reach checkout"}}`,
		)

		return &detail
	}

	vision, ok := asObject(config["vision"])
	if !ok {
		vision = map[string]any{}
		config["vision"] = vision
	}

	model, hasModel := vision["model"].(string)
	if !hasModel || strings.TrimSpace(model) == "" {
		defaultModel := strings.TrimSpace(os.Getenv(aiNavigatorDefaultModelEnv))
		if defaultModel == "" {
			detail := httputil.NewValidationError(
				"scanner_configs.ai-navigator.vision.model",
				"ai-navigator requires vision.model (string)",
				`Provide a non-empty model, e.g. {"vision":{"model":"openai/gpt-4o-mini"}}, or set AI_NAVIGATOR_DEFAULT_MODEL in your server environment.`,
			)

			return &detail
		}

		vision["model"] = defaultModel
	}

	if provider, hasProvider := vision["provider"]; hasProvider {
		if providerStr, isStr := provider.(string); isStr && strings.TrimSpace(providerStr) != "" &&
			!strings.EqualFold(providerStr, "openrouter") {
			detail := httputil.NewValidationError(
				"scanner_configs.ai-navigator.vision.provider",
				fmt.Sprintf("ai-navigator vision.provider must be %q", "openrouter"),
				`Use "openrouter" or omit provider (it defaults to openrouter).`,
			)

			return &detail
		}
	}

	return nil
}

func asObject(raw any) (map[string]any, bool) {
	if raw == nil {
		return nil, false
	}

	obj, ok := raw.(map[string]any)

	return obj, ok
}
