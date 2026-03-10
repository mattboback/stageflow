package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/mattboback/stageflow/packages/shared-go/logging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	scanners "github.com/mattboback/stageflow/packages/shared-go/scannerregistry"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
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
	ScannerRegistry      *scanners.Registry
	DefaultScannerImage  string
	NatsHost             string
	MinioHost            string
	MinioAccessKey       string
	MinioSecretKey       string
	MinioUseSSL          bool
	PageLoadTimeout      int
	ScrollTimeout        int
	PodNetnsMode         string
	DefaultScannerUser   string
	OpenRouterAPIKey     string
	OpenRouterAppTitle   string
	OpenRouterAppReferer string
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

	p.applyAINavigatorEnv(env, scannerType)

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

func (p *ScannerLaunchPlanner) applyAINavigatorEnv(env map[string]string, scannerType string) {
	if scannerType != "ai-navigator" {
		return
	}

	if p.config.OpenRouterAPIKey != "" {
		env["OPENROUTER_API_KEY"] = p.config.OpenRouterAPIKey
	}

	if p.config.OpenRouterAppTitle != "" {
		env["OPENROUTER_APP_TITLE"] = p.config.OpenRouterAppTitle
	}

	if p.config.OpenRouterAppReferer != "" {
		env["OPENROUTER_APP_REFERER"] = p.config.OpenRouterAppReferer
	}
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
