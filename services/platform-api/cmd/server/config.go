package main

import "github.com/mattboback/stageflow/libs/go/config"

// Config captures Platform API runtime configuration.
type Config struct {
	DataDir             string
	Port                int
	DatabaseURL         string
	AllowPrivateTargets bool
	APIToken            string
	TombstoneDBPath     string
	NATS                config.NATSConfig
}

func loadConfig() *Config {
	return &Config{
		DataDir:             config.GetEnv("STAGEFLOW_DATA_DIR", "/data"),
		Port:                config.GetEnvInt("PORT", 8080),
		DatabaseURL:         config.GetEnv("DATABASE_URL", ""),
		AllowPrivateTargets: config.GetEnvBool("PLATFORM_API_ALLOW_PRIVATE_TARGETS", false),
		APIToken:            config.GetEnv("PLATFORM_API_TOKEN", ""),
		TombstoneDBPath:     config.GetEnv("PROJECT_DB_PATH", "./projects.db"),
		NATS:                config.LoadNATSConfig(),
	}
}

// Validate ensures required configuration is present before startup.
func (c *Config) Validate() error {
	return config.ValidateAll(
		config.RequirePositive("PORT", c.Port),
		config.RequireNonEmpty("NATS_URL", c.NATS.URL),
		config.RequireNonEmpty("DATABASE_URL", c.DatabaseURL),
		config.RequireNonEmpty("STAGEFLOW_DATA_DIR", c.DataDir),
	)
}
