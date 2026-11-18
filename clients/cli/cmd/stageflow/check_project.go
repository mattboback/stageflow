package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

var checkReportVersion = regexp.MustCompile(`^2\.\d+\.\d+$`)

type checkScanner struct {
	ID      string         `json:"id"`
	Enabled *bool          `json:"enabled"`
	Config  map[string]any `json:"config,omitempty"`
}
type checkConfiguration struct {
	URLs           []string       `json:"urls"`
	Scanners       []checkScanner `json:"scanners"`
	Browser        string         `json:"browser"`
	HighlightStyle string         `json:"highlightStyle"`
	Profile        string         `json:"profile,omitempty"`
}
type checkProject struct {
	configuration checkConfiguration
	Schema        string `json:"schema"`
	Project       struct {
		ID            string          `json:"id"`
		Configuration json.RawMessage `json:"configuration"`
	} `json:"project"`
	Baseline *struct {
		ProjectID   string                 `json:"projectId"`
		JobID       string                 `json:"jobId"`
		Fingerprint string                 `json:"configFingerprint"`
		Report      report.UnifiedReportV2 `json:"report"`
	} `json:"baseline"`
}

func loadCheckProject(path string) (checkProject, apiclient.SubmitJobRequest, error) {
	var project checkProject

	req := apiclient.SubmitJobRequest{Screenshot: true}

	file, err := os.Open(path)
	if err != nil {
		return project, req, err
	}
	defer file.Close()
	// Match the browser's import bound without unbounded file allocation.
	data, err := io.ReadAll(io.LimitReader(file, (96<<20)+1))
	if err != nil {
		return project, req, err
	}

	if len(data) > 96<<20 {
		return project, req, errors.New("project export exceeds 96 MiB")
	}

	if err = json.Unmarshal(data, &project); err != nil {
		return project, req, fmt.Errorf("invalid project export: %w", err)
	}

	if project.Schema != "stageflow/local-project@v2" && project.Schema != "stageflow/local-project@v1" {
		return project, req, errors.New("expected a browser project export (stageflow/local-project@v2 or @v1)")
	}

	if project.Project.ID == "" {
		return project, req, errors.New("project ID is missing")
	}

	var configuration checkConfiguration

	decoder := json.NewDecoder(bytes.NewReader(project.Project.Configuration))
	decoder.DisallowUnknownFields()

	if err = decoder.Decode(&configuration); err != nil {
		return project, req, fmt.Errorf("unsupported project configuration: %w", err)
	}

	project.configuration = configuration

	req, err = configuration.request()
	if err != nil {
		return project, req, err
	}

	if project.Baseline != nil {
		if err = project.validateBaseline(configuration); err != nil {
			return project, req, fmt.Errorf("baseline: %w", err)
		}
	}

	return project, req, nil
}

func (configuration checkConfiguration) request() (apiclient.SubmitJobRequest, error) {
	req := apiclient.SubmitJobRequest{Screenshot: true, Profile: defaultProfile(configuration.Profile)}
	if configuration.Browser != "chromium" || configuration.HighlightStyle != "solid" {
		return req, errors.New("project check supports chromium with solid highlighting only")
	}

	if configuration.Profile != "" && configuration.Profile != "production" && configuration.Profile != "staging" {
		return req, errors.New("project profile must be production or staging")
	}

	if len(configuration.URLs) == 0 || configuration.Scanners == nil {
		return req, errors.New("project must contain URLs and a scanner selection")
	}

	if err := validateCheckURLs(configuration.URLs); err != nil {
		return req, err
	}

	req.URLs = configuration.URLs

	return configuration.scannerRequest(req)
}

func (configuration checkConfiguration) scannerRequest(
	req apiclient.SubmitJobRequest,
) (apiclient.SubmitJobRequest, error) {
	req.ScannerConfigs = map[string]map[string]any{}

	seen := map[string]bool{}
	for _, scanner := range configuration.Scanners {
		if !scannercatalog.HasID(scanner.ID) || scanner.Enabled == nil || seen[scanner.ID] {
			return req, errors.New("project contains an unknown, duplicate or invalid scanner selection")
		}

		seen[scanner.ID] = true
		if hasExecutionOnlyKey(scanner.Config) {
			return req, errors.New(
				"project scanner configuration contains execution-only authentication or secret fields",
			)
		}

		if !*scanner.Enabled {
			continue
		}

		req.Modules = append(req.Modules, scanner.ID)
		if scanner.Config != nil {
			req.ScannerConfigs[scanner.ID] = scanner.Config
		}
	}

	if len(configuration.Scanners) > 0 && len(req.Modules) == 0 {
		return req, errors.New("project has no enabled scanners")
	}

	return req, nil
}

func hasExecutionOnlyKey(value any) bool {
	switch nested := value.(type) {
	case map[string]any:
		for key, child := range nested {
			if executionOnlyKey(key) {
				return true
			}

			if hasExecutionOnlyKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range nested {
			if hasExecutionOnlyKey(child) {
				return true
			}
		}
	}

	return false
}

func (project checkProject) validateBaseline(configuration checkConfiguration) error {
	baseline := project.Baseline

	doc := baseline.Report
	if baseline.ProjectID != project.Project.ID || baseline.JobID == "" || baseline.JobID != doc.Meta.JobId ||
		!checkReportVersion.MatchString(doc.Version) ||
		doc.Summary.TotalIssues != len(doc.Issues) {
		return errors.New("invalid or incomplete baseline report")
	}

	context := doc.Meta.ScanContext
	if context == nil || context.Purpose != report.ScanContextPurposeFull || context.Authenticated ||
		context.InputType != report.ScanContextInputTypeUrls {
		return errors.New("requires a full public URL scan with recorded execution settings")
	}

	if !completeCheckCoverage(doc) {
		return errors.New("scanner coverage is incomplete")
	}

	if baseline.Fingerprint == "" {
		return errors.New("missing saved configuration fingerprint")
	}

	return configuration.validateReport(doc)
}

func (configuration checkConfiguration) validateReport(doc report.UnifiedReportV2) error {
	context := doc.Meta.ScanContext
	if context == nil || context.Purpose != report.ScanContextPurposeFull || context.Authenticated ||
		context.InputType != report.ScanContextInputTypeUrls {
		return errors.New("requires a full public URL scan with recorded execution settings")
	}

	if context.Viewport == nil || context.Viewport.Width != 1280 || context.Viewport.Height != 720 {
		return errors.New("report viewport cannot be replayed by this project")
	}

	profile := "production"
	if doc.Meta.Profile != nil {
		profile = string(*doc.Meta.Profile)
	}

	if profile != defaultProfile(configuration.Profile) {
		return errors.New("report profile does not match the project")
	}

	pages := make([]string, 0, len(doc.Pages))

	ids := make([]string, 0, len(doc.Pages))
	for _, page := range doc.Pages {
		if !context.VerificationEligiblePages[page.Id] {
			return errors.New("report page settings cannot be replayed by this project")
		}

		pages = append(pages, page.Url)
		ids = append(ids, page.Id)
	}

	expectedIDs := slices.Clone(context.ExpectedPages)
	slices.Sort(expectedIDs)
	slices.Sort(ids)

	if !slices.Equal(expectedIDs, ids) ||
		!slices.Equal(normalizedCheckURLs(configuration.URLs), normalizedCheckURLs(pages)) {
		return errors.New("report pages do not match the saved project configuration")
	}

	return configuration.validateReportScanners(doc)
}

func (configuration checkConfiguration) validateReportScanners(doc report.UnifiedReportV2) error {
	requested, err := configuration.request()
	if err != nil {
		return err
	}

	expected := requested.Modules
	if len(expected) == 0 {
		expected = scannercatalog.IDs()
	}

	actual := make([]string, 0, len(doc.Scanners))
	for _, scanner := range doc.Scanners {
		actual = append(actual, scanner.Id)
	}

	slices.Sort(expected)
	slices.Sort(actual)

	if !slices.Equal(expected, actual) {
		return errors.New("report scanners do not match the project")
	}

	for _, id := range expected {
		config := requested.ScannerConfigs[id]
		if config == nil {
			config = map[string]any{}
		}

		context := doc.Meta.ScanContext
		if context.ScannerConfigFingerprints[id] == "" ||
			!reflect.DeepEqual(config, context.ScannerConfigurations[id]) {
			return errors.New("report scanner settings do not match the project")
		}
	}

	return nil
}

func normalizedCheckURLs(values []string) []string {
	result := make([]string, 0, len(values))
	for _, raw := range values {
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil
		}

		parsed.Fragment = ""

		parsed.Host = strings.ToLower(parsed.Host)
		if parsed.Path == "" {
			parsed.Path = "/"
		}

		if (parsed.Scheme == "https" && parsed.Port() == "443") || (parsed.Scheme == "http" && parsed.Port() == "80") {
			parsed.Host = parsed.Hostname()
		}

		result = append(result, parsed.String())
	}

	slices.Sort(result)

	return slices.Compact(result)
}

func validateCheckURLs(urls []string) error {
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
			parsed.User != nil ||
			strings.TrimSpace(raw) != raw {
			return errors.New(
				"project URLs must be explicit HTTP(S) URLs without credentials or surrounding whitespace",
			)
		}
	}

	if urlcheck.ContainsPrivateTargets(urls) {
		return errors.New("project check supports public pages only")
	}

	return nil
}

func executionOnlyKey(key string) bool {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}

		return -1
	}, key)
	switch normalized {
	case "apikey",
		"auth",
		"authorization",
		"inputvalues",
		"password",
		"secret",
		"storagestate",
		"token",
		"username",
		"fromenv":
		return true
	}

	return false
}

// Manual rule outcomes remain reviewable in a full scan; only evaluated comparisons
// may call a finding fixed. Intake checks page/scanner completion independently.
func completeCheckCoverage(doc report.UnifiedReportV2) bool {
	context := doc.Meta.ScanContext
	if context == nil || len(context.ExpectedPages) == 0 || len(doc.Scanners) == 0 ||
		len(diff.IncompleteScanners(doc.Scanners)) > 0 {
		return false
	}

	for _, scanner := range doc.Scanners {
		completed := map[string]bool{}

		for _, page := range scanner.Pages {
			if page.Status == report.ScannerPageCoverageStatusSuccess {
				completed[page.PageId] = true
			}
		}

		for _, id := range context.ExpectedPages {
			if !completed[id] {
				return false
			}
		}
	}

	return true
}
