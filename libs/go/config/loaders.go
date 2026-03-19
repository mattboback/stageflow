package config

import "time"

// NATSConfig captures JetStream connection defaults shared across services.
type NATSConfig struct {
	URL            string
	MaxReconnects  int
	ReconnectWait  time.Duration
	ConnectTimeout time.Duration
}

// LoadNATSConfig centralizes NATS env parsing for consistent defaults.
func LoadNATSConfig() NATSConfig {
	return NATSConfig{
		URL:            GetEnv("NATS_URL", "nats://localhost:4222"),
		MaxReconnects:  GetEnvInt("NATS_MAX_RECONNECTS", 10),
		ReconnectWait:  GetEnvDuration("NATS_RECONNECT_WAIT", 2*time.Second),
		ConnectTimeout: GetEnvDuration("NATS_CONNECT_TIMEOUT", 10*time.Second),
	}
}

// MinIOConfig captures MinIO connection and public URL settings shared across services.
type MinIOConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	PublicUseSSL   bool
	Region         string
	UseProxyURLs   bool
}

// LoadMinIOConfig centralizes MinIO env parsing for consistent defaults.
func LoadMinIOConfig() MinIOConfig {
	publicEndpoint := GetEnv("MINIO_PUBLIC_ENDPOINT", "")
	if publicEndpoint == "" {
		publicEndpoint = GetEnv("STAGEFLOW_PUBLIC_DOMAIN", "example.com")
	}

	accessKey := GetEnv("MINIO_ACCESS_KEY", "")
	if accessKey == "" {
		accessKey = GetEnv("MINIO_ROOT_USER", "")
	}

	secretKey := GetEnv("MINIO_SECRET_KEY", "")
	if secretKey == "" {
		secretKey = GetEnv("MINIO_ROOT_PASSWORD", "")
	}

	return MinIOConfig{
		Endpoint:       GetEnv("MINIO_ENDPOINT", "localhost:9000"),
		PublicEndpoint: publicEndpoint,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		UseSSL:         GetEnvBool("MINIO_USE_SSL", false),
		PublicUseSSL:   GetEnvBool("MINIO_PUBLIC_USE_SSL", true),
		Region:         GetEnv("MINIO_REGION", "us-east-1"),
		UseProxyURLs:   GetEnvBool("MINIO_USE_PROXY_URLS", false),
	}
}
