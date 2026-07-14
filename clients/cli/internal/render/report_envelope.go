package render

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/buildinfo"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

type CLIMeta struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
}

type APIMeta struct {
	BaseURL string `json:"base_url"`
}

type JobMeta struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ReportLinks struct {
	Job     string `json:"job"`
	Results string `json:"results"`
}

type IssueFilters struct {
	MaxIssues      int      `json:"max_issues"`
	IssuesReturned int      `json:"issues_returned"`
	IssuesTotal    int      `json:"issues_total"`
	Truncated      bool     `json:"truncated"`
	Sort           string   `json:"sort"`
	Severities     []string `json:"severities,omitempty"`
	Categories     []string `json:"categories,omitempty"`
}

type ReportEnvelope struct {
	Schema  string                 `json:"schema"`
	CLI     CLIMeta                `json:"cli"`
	API     APIMeta                `json:"api"`
	Job     JobMeta                `json:"job"`
	Links   ReportLinks            `json:"links"`
	URLs    []string               `json:"urls,omitempty"`
	Filters IssueFilters           `json:"filters"`
	Report  report.UnifiedReportV2 `json:"report"`
}

func BuildReportEnvelope(
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
) (ReportEnvelope, error) {
	jobLink, err := buildAPILink(apiBaseURL, fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(status.ID)))
	if err != nil {
		return ReportEnvelope{}, err
	}

	resultsLink, err := buildAPILink(apiBaseURL, fmt.Sprintf("/api/v1/jobs/%s/results", url.PathEscape(status.ID)))
	if err != nil {
		return ReportEnvelope{}, err
	}

	job := JobMeta{ID: status.ID, State: status.State, Error: status.Error}
	if !status.CreatedAt.IsZero() {
		job.CreatedAt = status.CreatedAt.UTC().Format(timeFormatRFC3339)
	}

	if !status.UpdatedAt.IsZero() {
		job.UpdatedAt = status.UpdatedAt.UTC().Format(timeFormatRFC3339)
	}

	return ReportEnvelope{
		Schema: "stageflow-cli/report@v1",
		CLI:    currentCLIMeta(),
		API:    APIMeta{BaseURL: apiBaseURL},
		Job:    job,
		Links: ReportLinks{
			Job:     jobLink,
			Results: resultsLink,
		},
		URLs:    collectReportURLs(doc),
		Filters: filters,
		Report:  doc,
	}, nil
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

func currentCLIMeta() CLIMeta {
	v := strings.TrimSpace(buildinfo.Version)
	if v == "" {
		v = "dev"
	}

	return CLIMeta{
		Version: v,
		Commit:  strings.TrimSpace(buildinfo.Commit),
		Date:    strings.TrimSpace(buildinfo.Date),
	}
}

func buildAPILink(baseURL, apiPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", baseURL, err)
	}

	ref, err := url.Parse(apiPath)
	if err != nil {
		return "", fmt.Errorf("invalid API path %q: %w", apiPath, err)
	}

	return base.ResolveReference(ref).String(), nil
}

func collectReportURLs(doc report.UnifiedReportV2) []string {
	seen := make(map[string]struct{}, len(doc.Pages)+1)
	urls := make([]string, 0, len(doc.Pages)+1)

	if doc.Meta.BaseUrl != nil && *doc.Meta.BaseUrl != "" {
		seen[*doc.Meta.BaseUrl] = struct{}{}
		urls = append(urls, *doc.Meta.BaseUrl)
	}

	for _, page := range doc.Pages {
		if page.Url == "" {
			continue
		}

		if _, ok := seen[page.Url]; ok {
			continue
		}

		seen[page.Url] = struct{}{}
		urls = append(urls, page.Url)
	}

	return urls
}
