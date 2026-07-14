package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func (c *Client) FetchJobStatus(ctx context.Context, jobID string) (JobStatus, error) {
	var status JobStatus
	apiPath := fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(jobID))
	if err := c.GetJSON(ctx, apiPath, &status); err != nil {
		return JobStatus{}, err
	}

	return status, nil
}

func (c *Client) FetchReport(ctx context.Context, jobID string) (report.UnifiedReportV2, error) {
	apiPath := fmt.Sprintf("/api/v1/jobs/%s/results", url.PathEscape(jobID))

	var raw json.RawMessage
	if err := c.GetJSON(ctx, apiPath, &raw); err != nil {
		return report.UnifiedReportV2{}, err
	}

	var doc report.UnifiedReportV2
	if err := json.Unmarshal(SanitizeReportJSON(raw), &doc); err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("failed to decode report: %w", err)
	}

	return doc, nil
}

// SanitizeReportJSON tolerates score grades emitted by older API versions.
func SanitizeReportJSON(raw []byte) []byte {
	return scoreGradeReplacer.ReplaceAll(raw, []byte(`"scoreGrade":"A+"`))
}

var scoreGradeReplacer = regexp.MustCompile(`"scoreGrade"\s*:\s*"(?:Excellent)"`)
