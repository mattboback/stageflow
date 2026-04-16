package main

import (
	"errors"
	"strings"

	"github.com/mattboback/stageflow/libs/go/config"
)

// Config captures orchestrator runtime configuration.
type Config struct {
	NATS                          config.NATSConfig
	MinIO                         config.MinIOConfig
	PodmanSocket                  string
	DatabaseURL                   string
	ExtractionImage               string
	ScannerImage                  string
	ScannerImageOverride          string
	APIPort                       string
	APIToken                      string
	PodNetwork                    string
	PodNetnsMode                  string   // Pod network namespace mode (bridge|host). Host mode is local-only.
	PodHostMappings               []string // Custom host:ip mappings for pods (e.g., "mysite.com:169.254.1.2")
	NatsHost                      string
	MinioHost                     string
	PageLoadTimeout               int
	ScrollTimeout                 int
	JobEventsRetentionDays        int
	JobEventsPruneIntervalMinutes int
	JobEventsPruneBatchSize       int
}

func loadConfig() *Config {
	natsCfg := config.LoadNATSConfig()
	minioCfg := config.LoadMinIOConfig()

	scannerImageEnv := strings.TrimSpace(config.GetEnv("SCANNER_IMAGE", ""))

	scannerImage := scannerImageEnv
	if scannerImage == "" {
		scannerImage = "localhost/stageflow/scanner-runner:latest"
	}

	// Parse host mappings from comma-separated env var (e.g., "host1:ip1,host2:ip2")
	var podHostMappings []string

	hostMappingsEnv := strings.TrimSpace(config.GetEnv("POD_HOST_MAPPINGS", ""))
	if hostMappingsEnv != "" {
		for _, mapping := range strings.Split(hostMappingsEnv, ",") {
			mapping = strings.TrimSpace(mapping)
			if mapping != "" {
				podHostMappings = append(podHostMappings, mapping)
			}
		}
	}

	databaseURL := strings.TrimSpace(config.GetEnv("DATABASE_URL", ""))

	return &Config{
		NATS:                          natsCfg,
		MinIO:                         minioCfg,
		PodmanSocket:                  config.GetEnv("PODMAN_SOCKET", "/run/podman/podman.sock"),
		DatabaseURL:                   databaseURL,
		ExtractionImage:               config.GetEnv("EXTRACTION_IMAGE", "localhost/stageflow/extractor:latest"),
		ScannerImage:                  scannerImage,
		ScannerImageOverride:          scannerImageEnv,
		APIPort:                       config.GetEnv("API_PORT", "8081"),
		APIToken:                      strings.TrimSpace(config.GetEnv("ORCHESTRATOR_API_TOKEN", "")),
		PodNetwork:                    config.GetEnv("POD_NETWORK", ""),
		PodNetnsMode:                  config.GetEnv("POD_NETNS_MODE", "bridge"),
		PodHostMappings:               podHostMappings,
		NatsHost:                      config.GetEnv("NATS_HOST", "nats"),
		MinioHost:                     config.GetEnv("MINIO_HOST", "minio"),
		PageLoadTimeout:               config.GetEnvInt("PAGE_LOAD_TIMEOUT", 15000),
		ScrollTimeout:                 config.GetEnvInt("A11Y_SCROLL_TIMEOUT", 300),
		JobEventsRetentionDays:        config.GetEnvInt("JOB_EVENTS_RETENTION_DAYS", 30),
		JobEventsPruneIntervalMinutes: config.GetEnvInt("JOB_EVENTS_PRUNE_INTERVAL_MINUTES", 60),
		JobEventsPruneBatchSize:       config.GetEnvInt("JOB_EVENTS_PRUNE_BATCH_SIZE", 1000),
	}
}

// Validate ensures required configuration is present before startup.
func (c *Config) Validate() error {
	errs := []error{
		config.RequireNonEmpty("NATS_URL", c.NATS.URL),
		config.RequireNonEmpty("MINIO_ENDPOINT", c.MinIO.Endpoint),
		config.RequireNonEmpty("MINIO_ACCESS_KEY (or MINIO_ROOT_USER)", c.MinIO.AccessKey),
		config.RequireNonEmpty("MINIO_SECRET_KEY (or MINIO_ROOT_PASSWORD)", c.MinIO.SecretKey),
		config.RequireNonEmpty("PODMAN_SOCKET", c.PodmanSocket),
		config.RequireNonEmpty("DATABASE_URL", c.DatabaseURL),
		config.RequireNonEmpty("EXTRACTION_IMAGE", c.ExtractionImage),
		config.RequireNonEmpty("API_PORT", c.APIPort),
		config.RequireNonEmpty("ORCHESTRATOR_API_TOKEN", c.APIToken),
		config.RequireNonEmpty("NATS_HOST", c.NatsHost),
		config.RequireNonEmpty("MINIO_HOST", c.MinioHost),
	}

	podNetnsMode := strings.TrimSpace(c.PodNetnsMode)
	podNetwork := strings.TrimSpace(c.PodNetwork)

	switch podNetnsMode {
	case "", "bridge", "host":
	default:
		errs = append(errs, errors.New("POD_NETNS_MODE must be one of: bridge, host"))
	}

	if podNetnsMode != "" && podNetnsMode != "bridge" && podNetwork != "" {
		errs = append(errs, errors.New("POD_NETNS_MODE must be bridge when POD_NETWORK is set"))
	}

	if c.PageLoadTimeout < 0 {
		errs = append(errs, errors.New("PAGE_LOAD_TIMEOUT must be >= 0"))
	}

	if c.ScrollTimeout < 0 {
		errs = append(errs, errors.New("A11Y_SCROLL_TIMEOUT must be >= 0"))
	}

	if c.JobEventsRetentionDays < 0 {
		errs = append(errs, errors.New("JOB_EVENTS_RETENTION_DAYS must be >= 0"))
	}

	if c.JobEventsPruneIntervalMinutes < 0 {
		errs = append(errs, errors.New("JOB_EVENTS_PRUNE_INTERVAL_MINUTES must be >= 0"))
	}

	if c.JobEventsPruneBatchSize < 0 {
		errs = append(errs, errors.New("JOB_EVENTS_PRUNE_BATCH_SIZE must be >= 0"))
	}

	if c.JobEventsRetentionDays > 0 && c.JobEventsPruneIntervalMinutes == 0 {
		errs = append(errs, errors.New("JOB_EVENTS_PRUNE_INTERVAL_MINUTES must be > 0 when retention is enabled"))
	}

	if c.JobEventsRetentionDays > 0 && c.JobEventsPruneBatchSize == 0 {
		errs = append(errs, errors.New("JOB_EVENTS_PRUNE_BATCH_SIZE must be > 0 when retention is enabled"))
	}

	return config.ValidateAll(errs...)
}
