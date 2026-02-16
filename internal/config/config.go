package config

import (
	"os"
	"time"
)

type Config struct {
	GatewayBaseURL string
	GatewayAPIKey  string

	ListenAddr    string
	DatabaseURL   string
	GatewayAPIKey string
	MaxEventSkew  time.Duration
}

func Load() Config {
	return Config{
		ListenAddr:     getenv("CPMS_LISTEN_ADDR", ":8081"),
		GatewayBaseURL: getenv("GATEWAY_BASE_URL", "http://localhost:8080"),
		GatewayAPIKey:  getenv("GATEWAY_API_KEY", ""),
		DatabaseURL:    getenv("CPMS_DATABASE_URL", "postgres://cpms:cpms@localhost:5432/cpms?sslmode=disable"),
		GatewayAPIKey:  getenv("CPMS_GATEWAY_API_KEY", ""),
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
