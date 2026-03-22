package main

import "github.com/mattboback/stageflow/libs/go/config"

// Config captures Platform API runtime configuration.
type Config struct {
	Port                int
	OrchestratorAPIURL  string
	ScannerConfigPath   string
	AllowPrivateTargets bool
	ProjectDBPath       string
	NATS                config.NATSConfig
	MinIO               config.MinIOConfig
}

func loadConfig() *Config {
	return &Config{
		Port:                config.GetEnvInt("PORT", 8080),
		OrchestratorAPIURL:  config.GetEnv("ORCHESTRATOR_API_URL", "http://localhost:8081"),
		ScannerConfigPath:   config.GetEnv("SCANNER_CONFIG_PATH", ""),
		AllowPrivateTargets: config.GetEnvBool("PLATFORM_API_ALLOW_PRIVATE_TARGETS", false),
		ProjectDBPath:       config.GetEnv("PROJECT_DB_PATH", "./projects.db"),
		NATS:                config.LoadNATSConfig(),
		MinIO:               config.LoadMinIOConfig(),
	}
}

// Validate ensures required configuration is present before startup.
func (c *Config) Validate() error {
	return config.ValidateAll(
		config.RequirePositive("PORT", c.Port),
		config.RequireNonEmpty("NATS_URL", c.NATS.URL),
		config.RequireNonEmpty("MINIO_ENDPOINT", c.MinIO.Endpoint),
		config.RequireNonEmpty("MINIO_ACCESS_KEY (or MINIO_ROOT_USER)", c.MinIO.AccessKey),
		config.RequireNonEmpty("MINIO_SECRET_KEY (or MINIO_ROOT_PASSWORD)", c.MinIO.SecretKey),
		config.RequireNonEmpty("ORCHESTRATOR_API_URL", c.OrchestratorAPIURL),
	)
}
