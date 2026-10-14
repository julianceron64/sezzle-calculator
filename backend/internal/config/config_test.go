package config

import (
	"log/slog"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// API-10
func TestLoad_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{"defaults", nil, Config{Port: 8080, LogLevel: slog.LevelInfo}},
		{"empty values use defaults", map[string]string{"PORT": "", "STATIC_DIR": "", "LOG_LEVEL": ""}, Config{Port: 8080, LogLevel: slog.LevelInfo}},
		{"port override", map[string]string{"PORT": "9090"}, Config{Port: 9090, LogLevel: slog.LevelInfo}},
		{"port min", map[string]string{"PORT": "1"}, Config{Port: 1, LogLevel: slog.LevelInfo}},
		{"port max", map[string]string{"PORT": "65535"}, Config{Port: 65535, LogLevel: slog.LevelInfo}},
		{"static dir", map[string]string{"STATIC_DIR": "/srv/www"}, Config{Port: 8080, StaticDir: "/srv/www", LogLevel: slog.LevelInfo}},
		{"debug", map[string]string{"LOG_LEVEL": "debug"}, Config{Port: 8080, LogLevel: slog.LevelDebug}},
		{"info", map[string]string{"LOG_LEVEL": "info"}, Config{Port: 8080, LogLevel: slog.LevelInfo}},
		{"warn", map[string]string{"LOG_LEVEL": "warn"}, Config{Port: 8080, LogLevel: slog.LevelWarn}},
		{"error", map[string]string{"LOG_LEVEL": "error"}, Config{Port: 8080, LogLevel: slog.LevelError}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Load(env(tc.env))
			if err != nil {
				t.Fatalf("Load error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Load = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// API-10
func TestLoad_Invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"port zero", map[string]string{"PORT": "0"}},
		{"port negative", map[string]string{"PORT": "-1"}},
		{"port too large", map[string]string{"PORT": "65536"}},
		{"port not a number", map[string]string{"PORT": "abc"}},
		{"port with spaces", map[string]string{"PORT": " 80 "}},
		{"unknown log level", map[string]string{"LOG_LEVEL": "verbose"}},
		{"log level wrong case", map[string]string{"LOG_LEVEL": "INFO"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(env(tc.env)); err == nil {
				t.Fatal("Load succeeded, want error")
			}
		})
	}
}
