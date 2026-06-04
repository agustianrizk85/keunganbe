// Package config loads runtime configuration from environment variables with
// sensible defaults, so the service runs out of the box for local development.
package config

import (
	"os"
	"time"
)

// Config holds the server runtime configuration.
type Config struct {
	Port        string        // HTTP port to listen on
	AllowOrigin string        // CORS allowed origin
	DataPath    string        // JSON file the master data is persisted to (when no DB)
	DatabaseURL string        // PostgreSQL DSN; when set, used instead of the JSON file
	SessionTTL  time.Duration // bearer-token session lifetime
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:        getenv("FINANCE_PORT", "8084"),
		AllowOrigin: getenv("FINANCE_ALLOW_ORIGIN", "*"),
		DataPath:    getenv("FINANCE_DATA_PATH", "data/finance-data.json"),
		DatabaseURL: getenv("FINANCE_DATABASE_URL", ""),
		SessionTTL:  12 * time.Hour,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
