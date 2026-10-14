// Package config reads the service configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"strconv"
)

// Config is the validated service configuration.
type Config struct {
	Port      int
	StaticDir string // empty: do not serve the SPA
	LogLevel  slog.Level
}

var levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// Load reads PORT, STATIC_DIR and LOG_LEVEL. Unset or empty values use defaults.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{Port: 8080, StaticDir: getenv("STATIC_DIR"), LogLevel: slog.LevelInfo}

	if v := getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return Config{}, fmt.Errorf("PORT must be an integer in 1-65535, got %q", v)
		}
		cfg.Port = p
	}
	if v := getenv("LOG_LEVEL"); v != "" {
		l, ok := levels[v]
		if !ok {
			return Config{}, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error, got %q", v)
		}
		cfg.LogLevel = l
	}
	return cfg, nil
}
