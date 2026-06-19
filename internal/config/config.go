// Package config loads runtime configuration from environment variables with
// sensible defaults, so the service runs out of the box for local development.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the server runtime configuration.
type Config struct {
	Port        string        // HTTP port to listen on
	AllowOrigin string        // CORS allowed origin
	DataPath    string        // JSON file the dashboard state is persisted to (when no DB)
	DatabaseURL string        // PostgreSQL DSN; when set, used instead of the JSON file
	SessionTTL  time.Duration // bearer-token session lifetime

	// ---- Google Sheets ingest ----
	GoogleCreds string // path to service-account JSON; empty disables sync
	SheetID     string // spreadsheet ID to read all tabs from
	SyncSec     int    // auto-sync interval in seconds (0 = off)

	// ---- executive focus ----
	FocusYear  int // year the dashboard summary focuses on
	TargetAkad int // akad target for the focus year (0 = heuristic)
}

// defaultSheetID is the "Keuangan" akad/KPR spreadsheet shared with the team.
const defaultSheetID = "10au7z7FR6SpWt1VJ5TTB7WJSbuIBauECCj9zf9xWlYw"

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:        getenv("FINANCE_PORT", "8084"),
		AllowOrigin: getenv("FINANCE_ALLOW_ORIGIN", "*"),
		DataPath:    getenv("FINANCE_DATA_PATH", "data/finance-data.json"),
		DatabaseURL: getenv("FINANCE_DATABASE_URL", ""),
		SessionTTL:  12 * time.Hour,
		GoogleCreds: getenv("FINANCE_GOOGLE_CREDENTIALS", ""),
		SheetID:     getenv("FINANCE_GSHEET_ID", defaultSheetID),
		SyncSec:     getint("FINANCE_SYNC_INTERVAL_SEC", 0),
		FocusYear:   getint("FINANCE_FOCUS_YEAR", 2026),
		TargetAkad:  getint("FINANCE_TARGET_AKAD", 0),
	}
}

func getint(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
