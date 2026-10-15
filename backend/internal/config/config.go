package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains process settings. Secrets and provider settings belong to their future modules.
type Config struct {
	HTTPAddr        string
	DatabasePath    string
	LogLevel        string
	LogFormat       string
	ShutdownTimeout time.Duration
	CORSOrigins     []string
}

// Load reads environment variables and fails on invalid settings.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	get := func(key, fallback string) string {
		if value, ok := lookup(key); ok {
			return strings.TrimSpace(value)
		}
		return fallback
	}
	cfg := Config{
		HTTPAddr:     get("HTTP_ADDR", "127.0.0.1:8080"),
		DatabasePath: get("DATABASE_PATH", "./data/onboarding.db"),
		LogLevel:     get("LOG_LEVEL", "info"),
		LogFormat:    get("LOG_FORMAT", "console"),
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	timeout, err := time.ParseDuration(get("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil || timeout <= 0 {
		return Config{}, errors.New("SHUTDOWN_TIMEOUT must be a positive duration")
	}
	cfg.ShutdownTimeout = timeout
	origins, err := parseOrigins(get("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"))
	if err != nil {
		return Config{}, err
	}
	cfg.CORSOrigins = origins
	return cfg, nil
}

func validate(cfg Config) error {
	_, port, err := net.SplitHostPort(cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR must be host:port: %w", err)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return errors.New("HTTP_ADDR must contain a port from 1 to 65535")
	}
	if cfg.DatabasePath == "" || cfg.DatabasePath == ":memory:" || strings.HasPrefix(cfg.DatabasePath, "file:") {
		return errors.New("DATABASE_PATH must be a file path")
	}
	switch cfg.LogLevel {
	case "trace", "debug", "info", "warn", "error":
	default:
		return errors.New("LOG_LEVEL must be trace, debug, info, warn or error")
	}
	if cfg.LogFormat != "console" && cfg.LogFormat != "json" {
		return errors.New("LOG_FORMAT must be console or json")
	}
	return nil
}

func parseOrigins(value string) ([]string, error) {
	var origins []string
	for entry := range strings.SplitSeq(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		origin, err := url.Parse(entry)
		if err != nil || origin.Host == "" || strings.Contains(origin.Host, "*") || origin.User != nil || origin.Path != "" ||
			origin.RawQuery != "" || origin.Fragment != "" || origin.ForceQuery ||
			(origin.Scheme != "http" && origin.Scheme != "https") {
			return nil, errors.New("CORS_ORIGINS contains an invalid origin")
		}
		origins = append(origins, entry)
	}
	return origins, nil
}
