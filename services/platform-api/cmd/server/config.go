package main

import (
	"errors"

	"github.com/mattboback/stageflow/libs/go/config"
)

// Config captures Platform API runtime configuration.
type Config struct {
	DataDir             string
	Port                int
	DatabaseURL         string
	AllowPrivateTargets bool
	APIToken            string
	MaxPages            int
	ZIPUploadEnabled    bool
}

func loadConfig() *Config {
	return &Config{
		DataDir:             config.GetEnv("STAGEFLOW_DATA_DIR", "/data"),
		Port:                config.GetEnvInt("PORT", 8080),
		DatabaseURL:         config.GetEnv("DATABASE_URL", ""),
		AllowPrivateTargets: config.GetEnvBool("PLATFORM_API_ALLOW_PRIVATE_TARGETS", false),
		APIToken:            config.GetEnv("PLATFORM_API_TOKEN", ""),
		MaxPages:            config.GetEnvInt("PLATFORM_API_MAX_PAGES", 100),
		ZIPUploadEnabled:    config.GetEnvBool("PLATFORM_API_ZIP_UPLOAD_ENABLED", true),
	}
}

// Validate ensures required configuration is present before startup.
func (c *Config) Validate() error {
	if c.MaxPages > 100 {
		return errors.New("PLATFORM_API_MAX_PAGES must be at most 100")
	}

	return config.ValidateAll(
		config.RequirePositive("PORT", c.Port),
		config.RequirePositive("PLATFORM_API_MAX_PAGES", c.MaxPages),
		config.RequireNonEmpty("DATABASE_URL", c.DatabaseURL),
		config.RequireNonEmpty("STAGEFLOW_DATA_DIR", c.DataDir),
	)
}
