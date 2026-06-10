package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
)

type AINavigatorRequest struct {
	Objective string `json:"objective"`
}

type AINavigatorResponse struct {
	Success    bool             `json:"success"`
	Provenance []CompressedProv `json:"provenance,omitempty"`
	FullProv   []any            `json:"full_provenance,omitempty"`
}

type CompressedProv struct {
	URL    string `json:"url"`
	Action string `json:"action,omitempty"`
}

func newAICmd(root *rootOptions) *cobra.Command {
	var (
		expandProvenance bool
		timeout          time.Duration
		allowPrivate     bool
	)

	cmd := &cobra.Command{
		Use:   "ai <url> <objective>",
		Short: "Run the experimental AI navigator",
		Long: "Run the experimental AI navigator scanner against one URL.\n\n" +
			"This command requires the target StageFlow API to be configured with an AI provider such as OpenRouter.",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			urlStrs, err := urlcheck.NormalizeTargets([]string{args[0]})
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			urlStr := urlStrs[0]
			objective := args[1]

			validateErr := urlcheck.ValidateLocalTargets(root.apiURL, []string{urlStr})
			if validateErr != nil {
				return exitCodeError{Code: 2, Err: validateErr}
			}

			effectiveAllowPrivate := allowPrivate

			if !cobraFlagChanged(cmd, "allow-private-targets") && urlcheck.ContainsPrivateTargets([]string{urlStr}) {
				effectiveAllowPrivate = true

				fmt.Fprintln(
					cmd.ErrOrStderr(),
					"Detected private/loopback targets; setting allow_private_targets=true.",
				)
			}

			client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

			req := apiclient.SubmitJobRequest{
				URLs:                []string{urlStr},
				Modules:             []string{"ai-navigator"},
				Screenshot:          false,
				AllowPrivateTargets: effectiveAllowPrivate,
				ScannerConfigs: map[string]map[string]any{
					"ai-navigator": {
						"goal": map[string]any{
							"objective": objective,
						},
						"vision": map[string]any{
							"provider": "openrouter",
							"model":    "openrouter/healer-alpha",
						},
					},
				},
			}

			status, reportDoc, err := runScanJob(
				cmd.Context(),
				client,
				req,
				timeout,
				cmd.ErrOrStderr(),
				false,
			)
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			success := determineSuccess(reportDoc)

			fullProv, compressedProv := fetchProvenance(cmd.Context(), status)

			res := AINavigatorResponse{
				Success: success,
			}

			if expandProvenance {
				res.FullProv = fullProv
			} else {
				res.Provenance = compressedProv
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(out))

			return nil
		},
	}

	cmd.Flags().BoolVar(&expandProvenance, "expand-provenance", false, "Show full provenance JSON")
	cmd.Flags().BoolVar(&allowPrivate, "allow-private-targets", false, "Allow private/loopback targets")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Max wait time")

	return cmd
}

func determineSuccess(reportDoc report.UnifiedReportV2) bool {
	success := true

	for _, issue := range reportDoc.Issues {
		if issue.Scanner == "ai-navigator" {
			if issue.Category != nil && *issue.Category == "flow" {
				success = false
				break
			}
		}
	}

	return success
}

//nolint:gocognit,gocyclo
func fetchProvenance(ctx context.Context, status apiclient.JobStatus) ([]any, []CompressedProv) {
	var fullProv []any

	var compressedProv []CompressedProv

	if status.Artifacts == nil {
		return fullProv, compressedProv
	}

	var artifacts models.ArtifactLocation

	if err := json.Unmarshal(status.Artifacts, &artifacts); err != nil || artifacts.ScannerArtifacts == nil {
		return fullProv, compressedProv
	}

	aiArt, ok := artifacts.ScannerArtifacts["ai-navigator"]
	if !ok || aiArt.ResultsJSON == "" {
		return fullProv, compressedProv
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, aiArt.ResultsJSON, nil)
	if err != nil {
		return fullProv, compressedProv
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fullProv, compressedProv
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fullProv, compressedProv
	}

	var results []map[string]any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&results); decodeErr != nil {
		return fullProv, compressedProv
	}

	for _, res := range results {
		rawRaw, hasRaw := res["rawResults"]
		if !hasRaw {
			continue
		}

		fullProv = append(fullProv, rawRaw)

		raw, isMap := rawRaw.(map[string]any)
		if !isMap {
			continue
		}

		stepsRaw, hasSteps := raw["steps"].([]any)
		if !hasSteps {
			continue
		}

		for _, stepRaw := range stepsRaw {
			step, isStepMap := stepRaw.(map[string]any)
			if !isStepMap {
				continue
			}

			actionStr := ""

			if action, hasAction := step["action"].(map[string]any); hasAction {
				actionStr = fmt.Sprintf("%v", action["type"])
			}

			compressedProv = append(compressedProv, CompressedProv{
				URL:    fmt.Sprintf("%v", step["url"]),
				Action: actionStr,
			})
		}
	}

	return fullProv, compressedProv
}
