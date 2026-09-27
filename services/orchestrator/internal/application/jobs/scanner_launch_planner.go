package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
// pod's environment as PROVENANCE_AUTH_JSON: the canonical Provenance.auth JSON
// the scanner-runner attaches to its synthesized Provenance. Form recipes carry
// literal values; storage_state carries an artifact_key, never raw bytes.
//
// Nothing from the orchestrator's own environment ever reaches the pod through
// a recipe: {from_env} references are refused here as well as at intake.
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

	if err := rejectFromEnvReferences(&auth); err != nil {
		return err
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

	return nil
}

// rejectFromEnvReferences refuses a {from_env} value on any step, whatever its
// type. The platform-api rejects them at intake; this second check means a
// queued or replayed job can never have the scanner type a value from the
// orchestrator's environment into a page.
func rejectFromEnvReferences(auth *provenance.Auth) error {
	if auth.Form == nil {
		return nil
	}

	for i, step := range auth.Form.Steps {
		if step.Value != nil && step.Value.FromEnv != nil {
			return fmt.Errorf(
				"auth: form.steps[%d].value is a from_env reference; only literal values are accepted",
				i,
			)
		}
	}

	return nil
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
