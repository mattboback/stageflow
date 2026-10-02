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
		MaxReconnects:  MustGetEnvInt("NATS_MAX_RECONNECTS", 10),
		ReconnectWait:  MustGetEnvDuration("NATS_RECONNECT_WAIT", 2*time.Second),
		ConnectTimeout: MustGetEnvDuration("NATS_CONNECT_TIMEOUT", 10*time.Second),
	}
}
