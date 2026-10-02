package main

import (
	"errors"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/config"
)

// Config captures orchestrator runtime configuration.
type Config struct {
	DataVolume                  string
	DataDir                     string
	NATS                        config.NATSConfig
	PodmanSocket                string
	PodmanRequestTimeout        time.Duration
	PodmanResponseHeaderTimeout time.Duration
	PodmanDialTimeout           time.Duration
	DatabaseURL                 string
	ExtractionImage             string
	ScannerImage                string
	ScannerImageOverride        string
	APIPort                     string
	PodNetwork                  string
	PodNetnsMode                string   // Pod network namespace mode (bridge|host). Host mode is local-only.
	PodHostMappings             []string // Custom host:ip mappings for pods (e.g., "mysite.com:169.254.1.2")
	NatsHost                    string
	PageLoadTimeout             int
}

func loadConfig() *Config {
	natsCfg := config.LoadNATSConfig()

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
		DataVolume:                  config.GetEnv("STAGEFLOW_DATA_VOLUME", "stageflow_data"),
		DataDir:                     config.GetEnv("STAGEFLOW_DATA_DIR", "/data"),
		NATS:                        natsCfg,
		PodmanSocket:                config.GetEnv("PODMAN_SOCKET", "/run/podman/podman.sock"),
		PodmanRequestTimeout:        config.MustGetEnvDuration("PODMAN_REQUEST_TIMEOUT", 120*time.Second),
		PodmanResponseHeaderTimeout: config.MustGetEnvDuration("PODMAN_RESPONSE_HEADER_TIMEOUT", 60*time.Second),
		PodmanDialTimeout:           config.MustGetEnvDuration("PODMAN_DIAL_TIMEOUT", 5*time.Second),
		DatabaseURL:                 databaseURL,
		ExtractionImage:             config.GetEnv("EXTRACTION_IMAGE", "localhost/stageflow/extractor:latest"),
		ScannerImage:                scannerImage,
		ScannerImageOverride:        scannerImageEnv,
		APIPort:                     config.GetEnv("API_PORT", "8081"),
		PodNetwork:                  config.GetEnv("POD_NETWORK", ""),
		PodNetnsMode:                config.GetEnv("POD_NETNS_MODE", "bridge"),
		PodHostMappings:             podHostMappings,
		NatsHost:                    config.GetEnv("NATS_HOST", "nats"),
		PageLoadTimeout:             config.GetEnvInt("PAGE_LOAD_TIMEOUT", 15000),
	}
}

// Validate ensures required configuration is present before startup.
func (c *Config) Validate() error {
	errs := []error{
		config.RequireNonEmpty("NATS_URL", c.NATS.URL),
		config.RequireNonEmpty("PODMAN_SOCKET", c.PodmanSocket),
		config.RequireNonEmpty("DATABASE_URL", c.DatabaseURL),
		config.RequireNonEmpty("EXTRACTION_IMAGE", c.ExtractionImage),
		config.RequireNonEmpty("API_PORT", c.APIPort),
		config.RequireNonEmpty("NATS_HOST", c.NatsHost),
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

	errs = append(errs, c.validatePodmanConfig()...)

	return config.ValidateAll(errs...)
}

// validatePodmanConfig checks the Podman client timeout budget.
func (c *Config) validatePodmanConfig() []error {
	var errs []error

	if c.PodmanRequestTimeout <= 0 {
		errs = append(errs, errors.New("PODMAN_REQUEST_TIMEOUT must be > 0"))
	}

	if c.PodmanResponseHeaderTimeout <= 0 {
		errs = append(errs, errors.New("PODMAN_RESPONSE_HEADER_TIMEOUT must be > 0"))
	}

	if c.PodmanDialTimeout <= 0 {
		errs = append(errs, errors.New("PODMAN_DIAL_TIMEOUT must be > 0"))
	}

	// The request timeout caps the whole exchange, so a larger header timeout can
	// never be reached -- the knob would silently do nothing.
	if c.PodmanResponseHeaderTimeout > c.PodmanRequestTimeout {
		errs = append(errs, errors.New(
			"PODMAN_RESPONSE_HEADER_TIMEOUT must be <= PODMAN_REQUEST_TIMEOUT",
		))
	}

	return errs
}
