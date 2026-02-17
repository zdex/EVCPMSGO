package config

import (
	"os"
	"time"
)

type Config struct {
	ListenAddr  string
	DatabaseURL string

	// Gateway (CPMS -> Gateway commands)
	GatewayBaseURL string
	GatewayAPIKey  string

	// Ingestion hardening
	MaxEventSkew time.Duration
}

func Load() Config {
	return Config{
		ListenAddr:     getenv("CPMS_LISTEN_ADDR", ":8081"),
		DatabaseURL:    getenv("CPMS_DATABASE_URL", "postgres://cpms:cpms@localhost:5432/cpms?sslmode=disable"),
		GatewayBaseURL: getenv("GATEWAY_BASE_URL", "http://localhost:8080"),
		GatewayAPIKey:  getenv("GATEWAY_API_KEY", ""),
		MaxEventSkew:   parseDuration(getenv("CPMS_MAX_EVENT_SKEW", "0s")),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
