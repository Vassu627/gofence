package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	APIKey      string
	CORSOrigins []string
	RatePerMin  int
	TrustProxy  bool
}

func Load() Config {
	rate := envInt("RATE_LIMIT_PER_MIN", 120)
	if rate < 1 {
		rate = 120
	}
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		APIKey:      strings.TrimSpace(os.Getenv("API_KEY")),
		CORSOrigins: splitCSV(getenv("CORS_ORIGINS", "*")),
		RatePerMin:  rate,
		TrustProxy:  strings.EqualFold(strings.TrimSpace(os.Getenv("TRUST_PROXY")), "true"),
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
