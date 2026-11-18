package api

import (
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/models"
)

func validateProductConfig(
	profile, identityScope string,
	verification *models.VerificationConfig,
	modules []string,
) *httputil.ErrorDetail {
	if profile != "" && profile != "production" && profile != "staging" {
		detail := httputil.NewValidationError("profile", "Unknown scan profile", "Use production or staging.")
		return &detail
	}

	if identityScope != "" && identityScope != "path" && identityScope != "origin" {
		detail := httputil.NewValidationError("identity_scope", "Unknown identity scope", "Use path or origin.")
		return &detail
	}

	if verification == nil {
		return nil
	}

	return validateVerificationConfig(verification, modules)
}

func validateVerificationConfig(verification *models.VerificationConfig, modules []string) *httputil.ErrorDetail {
	if len(modules) != 1 || modules[0] != verification.Scanner || verification.SourceJobID == "" ||
		verification.PageID == "" ||
		verification.RuleID == "" {
		detail := httputil.NewValidationError(
			"verification",
			"Verification requires an original finding and its single responsible scanner",
			"Include source_job_id, page_id, rule_id and scanner; select that scanner only.",
		)

		return &detail
	}

	for _, value := range []string{verification.SourceJobID, verification.PageID, verification.RuleID, verification.Scanner, verification.SubjectID} {
		if len(value) > 256 || strings.ContainsAny(value, "\r\n\x00") {
			detail := httputil.NewValidationError(
				"verification",
				"Invalid verification identity",
				"Use identifiers of at most 256 characters.",
			)

			return &detail
		}
	}

	return nil
}
