package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/provenance"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/libs/go/storage"
)

const (
	PodNetnsModeBridge       = "bridge"
	PodNetnsModeHost         = "host"
	HostNetnsNATSURL         = "nats://127.0.0.1:4222"
	HostNetnsMinioEndpoint   = "127.0.0.1:9000"
	defaultScannerMemoryMB   = 2048
	defaultPageLoadTimeoutMs = 15_000
	defaultScrollTimeoutMs   = 300
	defaultScannerUser       = "0"
)

type ScannerLaunchPlannerConfig struct {
	ScannerRegistry     *scanners.Registry
	DefaultScannerImage string
	NatsHost            string
	MinioHost           string
	MinioAccessKey      string
	MinioSecretKey      string
	MinioUseSSL         bool
	PageLoadTimeout     int
	ScrollTimeout       int
	PodNetnsMode        string
	DefaultScannerUser  string
	// HostEnv is the source of truth for env-var values forwarded into the
	// scanner-runner pod. The planner only ever reads names that appear in the
	// auth recipe's from_env references; arbitrary passthrough is forbidden.
	// Tests inject a stub map; production wires os.Getenv at startup.
	HostEnv func(name string) string
}

type ScannerLaunchPlanner struct {
	config ScannerLaunchPlannerConfig
}

type VolumeRequirement struct {
	Name        string
	Destination string
	ReadOnly    bool
}

type ResourceLimits struct {
	MemoryLimitMB int64
	MemorySwapMB  int64
}

type ScannerLaunchPlan struct {
	Name           string
	Image          string
	User           string
	Env            map[string]string
	Labels         map[string]string
	Volumes        []VolumeRequirement
	ResourceLimits ResourceLimits
}

func NewScannerLaunchPlanner(config ScannerLaunchPlannerConfig) *ScannerLaunchPlanner {
	if config.NatsHost == "" {
		config.NatsHost = "nats"
	}

	if config.MinioHost == "" {
		config.MinioHost = "minio"
	}

	if config.PageLoadTimeout == 0 {
		config.PageLoadTimeout = defaultPageLoadTimeoutMs
	}

	if config.ScrollTimeout == 0 {
		config.ScrollTimeout = defaultScrollTimeoutMs
	}

	if config.PodNetnsMode == "" {
		config.PodNetnsMode = PodNetnsModeBridge
	}

	if config.DefaultScannerUser == "" {
		config.DefaultScannerUser = defaultScannerUser
	}

	return &ScannerLaunchPlanner{config: config}
}

func (p *ScannerLaunchPlanner) Plan(
	ctx context.Context,
	job *models.Job,
	scannerType string,
) (*ScannerLaunchPlan, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}

	if scannerType == "" {
		return nil, errors.New("scanner type is required")
	}

	natsURL, minioEndpoint := p.serviceEndpoints()
	resultsDir, provenancePath := scannerPaths(job, scannerType)
	env := p.baseEnv(ctx, job, scannerType, natsURL, minioEndpoint, resultsDir, provenancePath)

	if err := p.applyURLInputs(env, job); err != nil {
		return nil, err
	}

	if err := p.applyScannerConfig(env, job, scannerType); err != nil {
		return nil, err
	}

	if err := p.applyAuth(env, job); err != nil {
		return nil, err
	}

	return &ScannerLaunchPlan{
		Name:  fmt.Sprintf("scanner-%s-%s", scannerType, job.ID),
		Image: p.scannerImage(scannerType),
		User:  p.config.DefaultScannerUser,
		Env:   env,
		Labels: map[string]string{
			"managed_by":   "orchestrator",
			"job_id":       job.ID,
			"component":    "scanner",
			"scanner_type": scannerType,
		},
		Volumes: []VolumeRequirement{
			{
				Name:        "workspace-" + job.ID,
				Destination: "/workspace",
				ReadOnly:    true,
			},
			{
				Name:        "results-" + job.ID,
				Destination: "/results",
			},
		},
		ResourceLimits: p.resourceLimits(scannerType),
	}, nil
}

func (p *ScannerLaunchPlanner) serviceEndpoints() (string, string) {
	natsURL := "nats://" + p.config.NatsHost + ":4222"
	minioEndpoint := p.config.MinioHost + ":9000"

	if p.config.PodNetnsMode == PodNetnsModeHost {
		return HostNetnsNATSURL, HostNetnsMinioEndpoint
	}

	return natsURL, minioEndpoint
}

func scannerPaths(job *models.Job, scannerType string) (string, string) {
	resultsDir := "/results/" + scannerType
	provenancePath := "/workspace/provenance.json"

	if job.InputType == models.JobInputTypeURLs {
		provenancePath = resultsDir + "/provenance.json"
	}

	return resultsDir, provenancePath
}

func (p *ScannerLaunchPlanner) baseEnv(
	ctx context.Context,
	job *models.Job,
	scannerType, natsURL, minioEndpoint, resultsDir, provenancePath string,
) map[string]string {
	env := map[string]string{
		"JOB_ID":                job.ID,
		"SCANNER_TYPE":          scannerType,
		"NATS_URL":              natsURL,
		"MINIO_ENDPOINT":        minioEndpoint,
		"MINIO_ACCESS_KEY":      p.config.MinioAccessKey,
		"MINIO_SECRET_KEY":      p.config.MinioSecretKey,
		"MINIO_USE_SSL":         strconv.FormatBool(p.config.MinioUseSSL),
		"MINIO_ARTIFACT_BUCKET": storage.BucketArtifacts,
		"PROVENANCE_PATH":       provenancePath,
		"RESULTS_DIR":           resultsDir,
		"PAGE_LOAD_TIMEOUT":     strconv.Itoa(p.config.PageLoadTimeout),
		"A11Y_SCROLL_TIMEOUT":   strconv.Itoa(p.config.ScrollTimeout),
		"A11Y_SHOT_ENABLED":     strconv.FormatBool(job.Config.Screenshot),
		"A11Y_HIGHLIGHT_STYLE":  highlightStyle(job.Config.HighlightStyle),
		"BROWSER_ENGINE":        browserEngine(job.Config.Browser),
	}

	if job.Config.AllowPrivateTargets {
		env["ALLOW_PRIVATE_TARGETS"] = strconv.FormatBool(true)
	}

	if requestID := logging.RequestID(ctx); requestID != "" {
		env["REQUEST_ID"] = requestID
	}

	if runID := logging.RunID(ctx); runID != "" {
		env["RUN_ID"] = runID
	}

	return env
}

func (p *ScannerLaunchPlanner) applyURLInputs(env map[string]string, job *models.Job) error {
	if job.InputType != models.JobInputTypeURLs || len(job.URLs) == 0 {
		return nil
	}

	urlsJSON, err := json.Marshal(job.URLs)
	if err != nil {
		return fmt.Errorf("marshal scan urls: %w", err)
	}

	env["SCAN_URLS"] = string(urlsJSON)

	return nil
}

func (p *ScannerLaunchPlanner) applyScannerConfig(
	env map[string]string,
	job *models.Job,
	scannerType string,
) error {
	if len(job.Config.ScannerConfigs) == 0 {
		return nil
	}

	scannerConfig, ok := job.Config.ScannerConfigs[scannerType]
	if !ok || len(scannerConfig) == 0 {
		return nil
	}

	configJSON, err := json.Marshal(scannerConfig)
	if err != nil {
		return fmt.Errorf("marshal scanner config for %s: %w", scannerType, err)
	}

	env["SCANNER_OPTIONS"] = string(configJSON)

	return nil
}

// applyAuth wires the optional Provenance.auth block into the scanner-runner
// pod's environment.
//
// Two outputs:
//
//  1. PROVENANCE_AUTH_JSON: the canonical Provenance.auth JSON the scanner-runner
//     attaches to its synthesized Provenance. Carries from_env references for
//     form mode and an artifact_key for storage_state mode; never raw bytes,
//     never resolved credentials.
//  2. Forwarded env vars: exactly the names referenced by Provenance.auth.form.steps
//     via {from_env: NAME}. The planner reads them from the orchestrator host
//     (HostEnv) and injects them into the pod env. Anything else from the host
//     environment stays out of the pod.
//
// If a referenced env var is unset on the orchestrator host we fail fast with
// a structured error before the pod starts; producing a clean axe report
// against the login page is the worst possible outcome and the recipe is the
// place to surface this.
func (p *ScannerLaunchPlanner) applyAuth(env map[string]string, job *models.Job) error {
	if len(job.Config.Auth) == 0 {
		return nil
	}

	var auth provenance.Auth
	if err := json.Unmarshal(job.Config.Auth, &auth); err != nil {
		return fmt.Errorf("auth: invalid JobConfig.Auth payload: %w", err)
	}

	if err := provenance.ValidateAuth(&auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if auth.Mode == provenance.AuthModeStorageState && auth.StorageState != nil &&
		auth.StorageState.ContentBase64 != "" {
		// The platform-api normally uploads storage-state bytes before
		// job.created is published. The orchestrator also normalizes inline
		// legacy producer payloads during CreateJob, so reaching scanner launch
		// with inline content is a wiring bug; refuse rather than leak.
		return errors.New(
			"auth.storage_state still has inline content_b64 at scanner-launch time; " +
				"it must be uploaded to MinIO and replaced with an artifact_key first",
		)
	}

	compactJSON, err := json.Marshal(auth.Compact())
	if err != nil {
		return fmt.Errorf("auth: failed to encode PROVENANCE_AUTH_JSON: %w", err)
	}

	env["PROVENANCE_AUTH_JSON"] = string(compactJSON)

	allowList := provenance.CollectFromEnvReferences(&auth)
	if len(allowList) == 0 {
		return nil
	}

	hostEnv := p.config.HostEnv
	if hostEnv == nil {
		hostEnv = os.Getenv
	}

	missing := make([]string, 0)

	for _, name := range allowList {
		// Refuse to overwrite operationally critical env vars even if a
		// recipe accidentally references them. The scanner-runner sets these
		// via baseEnv and they must not be co-opted by an auth recipe.
		if _, reserved := reservedScannerEnvNames[name]; reserved {
			return fmt.Errorf(
				"auth: from_env reference %q collides with a reserved scanner env var; "+
					"choose a different env var name in the recipe",
				name,
			)
		}

		value := hostEnv(name)
		if value == "" {
			missing = append(missing, name)

			continue
		}

		env[name] = value
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"auth: from_env references not set on orchestrator host: %v "+
				"(set them in the orchestrator deployment env, or remove them from the recipe)",
			missing,
		)
	}

	return nil
}

// reservedScannerEnvNames are env vars the scanner-runner interprets as
// infrastructure or behavior controls, plus retired provider settings that may
// still contain credentials on an orchestrator host. An auth recipe must not be
// able to set them via {from_env: "NAME"}: doing so would let a recipe pull a
// value off the orchestrator host env and inject it into the scanner pod. The
// list is intentionally broader than the vars baseEnv() sets today, so the
// invariant holds even for controls the scanner reads directly from its
// environment (see scanner-runner src/core/config-loader.ts).
var reservedScannerEnvNames = map[string]struct{}{
	"JOB_ID":                {},
	"SCANNER_TYPE":          {},
	"NATS_URL":              {},
	"MINIO_ENDPOINT":        {},
	"MINIO_ACCESS_KEY":      {},
	"MINIO_SECRET_KEY":      {},
	"MINIO_USE_SSL":         {},
	"MINIO_ARTIFACT_BUCKET": {},
	"PROVENANCE_PATH":       {},
	"PROVENANCE_AUTH_JSON":  {},
	"RESULTS_DIR":           {},
	"PAGE_LOAD_TIMEOUT":     {},
	"A11Y_SCROLL_TIMEOUT":   {},
	"A11Y_SHOT_ENABLED":     {},
	"A11Y_HIGHLIGHT_STYLE":  {},
	"ALLOW_PRIVATE_TARGETS": {},
	"REQUEST_ID":            {},
	"RUN_ID":                {},
	"SCAN_URLS":             {},
	"SCANNER_OPTIONS":       {},

	// Retired AI Navigator provider settings remain denied so lingering host
	// credentials cannot be forwarded through an auth recipe.
	"OPENROUTER_API_KEY":     {},
	"OPENROUTER_APP_TITLE":   {},
	"OPENROUTER_APP_REFERER": {},

	// Headless-browser controls: arbitrary Chromium flags or a CSP bypass would
	// undermine the browser sandbox if an attacker could inject them; the engine
	// is an orchestrator-set, validated job field, not a recipe-supplied value.
	"BROWSER_ARGS":       {},
	"BROWSER_BYPASS_CSP": {},
	"BROWSER_HEADLESS":   {},
	"BROWSER_ENGINE":     {},

	// Result-subject overrides: redirecting these would let a recipe forge or
	// suppress scan-result events on arbitrary NATS subjects.
	"NATS_SUBJECT_PAGE_COMPLETED": {},
	"NATS_SUBJECT_SCAN_COMPLETED": {},
	"NATS_SUBJECT_SCAN_FAILED":    {},

	// Data path and resource limits: control where the pod writes and how much
	// CPU/memory/time it can consume.
	"SCANNER_DATA_DIR": {},
	"SCAN_CONCURRENCY": {},
	"MAX_RETRIES":      {},
	"DEFAULT_TIMEOUT":  {},
	"VIEWPORT_WIDTH":   {},
	"VIEWPORT_HEIGHT":  {},
}

func (p *ScannerLaunchPlanner) scannerImage(scannerType string) string {
	if p.config.ScannerRegistry != nil {
		if def, ok := p.config.ScannerRegistry.Get(scannerType); ok && def.Image != "" && def.Image != scannerType {
			return def.Image
		}
	}

	if p.config.DefaultScannerImage != "" {
		return p.config.DefaultScannerImage
	}

	if p.config.ScannerRegistry != nil {
		if image := p.config.ScannerRegistry.GetImage(scannerType); image != "" {
			return image
		}
	}

	return scannerType
}

func (p *ScannerLaunchPlanner) resourceLimits(scannerType string) ResourceLimits {
	if p.config.ScannerRegistry != nil {
		if def, ok := p.config.ScannerRegistry.Get(scannerType); ok && def.Requirements.MaxMemoryMB > 0 {
			memory := int64(def.Requirements.MaxMemoryMB)

			return ResourceLimits{
				MemoryLimitMB: memory,
				MemorySwapMB:  memory,
			}
		}
	}

	return ResourceLimits{
		MemoryLimitMB: defaultScannerMemoryMB,
		MemorySwapMB:  defaultScannerMemoryMB,
	}
}

func highlightStyle(style string) string {
	if style == "" {
		return "dashed"
	}

	return style
}

// browserEngine normalizes the job's Playwright engine for the scanner env.
// The Platform API already validates this field, but BROWSER_ENGINE is a
// reserved/security-sensitive var, so we defensively re-validate here: anything
// outside the known set (including empty) falls back to chromium.
func browserEngine(engine string) string {
	switch engine {
	case "firefox", "webkit":
		return engine
	default:
		return "chromium"
	}
}
