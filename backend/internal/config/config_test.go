package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		env     map[string]string
		invalid bool
	}{
		{name: "defaults"},
		{name: "overrides", env: map[string]string{"HTTP_ADDR": ":9000", "LOG_FORMAT": "json", "CORS_ORIGINS": "https://learn.example"}},
		{name: "cors disabled", env: map[string]string{"CORS_ORIGINS": ""}},
		{name: "bad address", env: map[string]string{"HTTP_ADDR": "localhost"}, invalid: true},
		{name: "bad port", env: map[string]string{"HTTP_ADDR": ":0"}, invalid: true},
		{name: "empty database", env: map[string]string{"DATABASE_PATH": ""}, invalid: true},
		{name: "database URI", env: map[string]string{"DATABASE_PATH": "file:test.db"}, invalid: true},
		{name: "bad level", env: map[string]string{"LOG_LEVEL": "verbose"}, invalid: true},
		{name: "bad format", env: map[string]string{"LOG_FORMAT": "yaml"}, invalid: true},
		{name: "bad timeout", env: map[string]string{"SHUTDOWN_TIMEOUT": "0s"}, invalid: true},
		{name: "bad duration", env: map[string]string{"SHUTDOWN_TIMEOUT": "soon"}, invalid: true},
		{name: "wildcard host", env: map[string]string{"CORS_ORIGINS": "https://*.example.com"}, invalid: true},
		{name: "wildcard origin", env: map[string]string{"CORS_ORIGINS": "*"}, invalid: true},
		{name: "origin path", env: map[string]string{"CORS_ORIGINS": "https://example.com/path"}, invalid: true},
		{name: "origin credentials", env: map[string]string{"CORS_ORIGINS": "https://user:pass@example.com"}, invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := load(func(key string) (string, bool) { value, ok := tt.env[key]; return value, ok })
			if (err != nil) != tt.invalid {
				t.Fatalf("load error = %v, invalid = %v", err, tt.invalid)
			}
			if !tt.invalid && cfg.ShutdownTimeout != 10*time.Second {
				t.Errorf("unexpected shutdown timeout: %v", cfg.ShutdownTimeout)
			}
			if tt.name == "overrides" && (cfg.HTTPAddr != ":9000" || cfg.LogFormat != "json" || len(cfg.CORSOrigins) != 1) {
				t.Fatalf("overrides not applied: %+v", cfg)
			}
			if tt.name == "defaults" && (cfg.HTTPAddr != "127.0.0.1:8080" || cfg.DatabasePath != "./data/onboarding.db") {
				t.Fatalf("unexpected defaults: %+v", cfg)
			}
		})
	}
}
