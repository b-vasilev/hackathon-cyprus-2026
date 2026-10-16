package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config contains process settings. Secrets and provider settings belong to their future modules.
type Config struct {
	HTTPAddr            string
	PublicURL           string
	DatabasePath        string
	ContentDir          string
	LogLevel            string
	LogFormat           string
	ShutdownTimeout     time.Duration
	CORSOrigins         []string
	ConnectionMasterKey string
	LLMPrivateOrigins   string
	FactoryWorkDir      string
	GenerationWorkers   int
	RepomixBin          string
	GitHubToken         string
	GitHubClientID      string
	GitHubClientSecret  string
	GitHubRedirectURL   string
	FrontendDir         string
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
		HTTPAddr:            get("HTTP_ADDR", "127.0.0.1:8080"),
		PublicURL:           strings.TrimRight(get("PUBLIC_URL", "http://127.0.0.1:5173"), "/"),
		DatabasePath:        get("DATABASE_PATH", "./data/onboarding.db"),
		LogLevel:            get("LOG_LEVEL", "info"),
		LogFormat:           get("LOG_FORMAT", "console"),
		ConnectionMasterKey: get("CONNECTION_MASTER_KEY", ""),
		LLMPrivateOrigins:   get("LLM_ALLOWED_PRIVATE_ORIGINS", ""),
		RepomixBin:          get("REPOMIX_BIN", "repomix"),
		GitHubToken:         get("GITHUB_TOKEN", ""),
		GitHubClientID:      get("GITHUB_OAUTH_CLIENT_ID", ""),
		GitHubClientSecret:  get("GITHUB_OAUTH_CLIENT_SECRET", ""),
		GitHubRedirectURL:   get("GITHUB_OAUTH_REDIRECT_URL", ""),
		FrontendDir:         get("FRONTEND_DIR", ""),
	}
	cfg.ContentDir = get("PASSPORT_CONTENT_DIR", filepath.Join(filepath.Dir(cfg.DatabasePath), "content"))
	if !validPublicOrigin(cfg.PublicURL) {
		return Config{}, errors.New("PUBLIC_URL must be an HTTPS origin, or HTTP on localhost, without a path, credentials, query or fragment")
	}
	if cfg.ContentDir == "" {
		return Config{}, errors.New("PASSPORT_CONTENT_DIR must be a directory path")
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	workDir, err := filepath.Abs(filepath.Join(filepath.Dir(cfg.DatabasePath), "wayfinder"))
	if err != nil {
		return Config{}, fmt.Errorf("resolve factory work directory: %w", err)
	}
	cfg.FactoryWorkDir = get("FACTORY_WORK_DIR", workDir)
	workers, err := strconv.Atoi(get("GENERATION_WORKERS", "3"))
	if err != nil || workers < 1 {
		return Config{}, errors.New("GENERATION_WORKERS must be a positive integer")
	}
	cfg.GenerationWorkers = workers
	timeout, err := time.ParseDuration(get("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil || timeout <= 0 {
		return Config{}, errors.New("SHUTDOWN_TIMEOUT must be a positive duration")
	}
	cfg.ShutdownTimeout = timeout
	origins, err := parseOrigins(get("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,https://wayfinder-t1a.eastus2.cloudapp.azure.com"))
	if err != nil {
		return Config{}, err
	}
	cfg.CORSOrigins = origins
	if cfg.ConnectionMasterKey != "" && len(cfg.ConnectionMasterKey) < 32 {
		return Config{}, errors.New("CONNECTION_MASTER_KEY must contain at least 32 characters")
	}
	if err := validateGitHubOAuth(cfg); err != nil {
		return Config{}, err
	}
	if !filepath.IsAbs(cfg.FactoryWorkDir) {
		return Config{}, errors.New("FACTORY_WORK_DIR must be an absolute path")
	}
	if binary, err := exec.LookPath(cfg.RepomixBin); err == nil {
		cfg.RepomixBin = binary
	} else if cfg.RepomixBin == "repomix" {
		for _, candidate := range []string{"../frontend/node_modules/.bin/repomix", "frontend/node_modules/.bin/repomix"} {
			if _, err := os.Stat(candidate); err == nil {
				absolute, err := filepath.Abs(candidate)
				if err == nil {
					cfg.RepomixBin = absolute
					break
				}
			}
		}
	}
	return cfg, nil
}

func validPublicOrigin(raw string) bool {
	public, err := url.Parse(raw)
	if err != nil || public.Host == "" || public.User != nil || public.Path != "" || public.RawQuery != "" || public.Fragment != "" {
		return false
	}
	local := public.Hostname() == "localhost" || public.Hostname() == "127.0.0.1" || public.Hostname() == "::1"
	return public.Scheme == "https" || public.Scheme == "http" && local
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

func validateGitHubOAuth(cfg Config) error {
	if cfg.GitHubClientID == "" && cfg.GitHubClientSecret == "" && cfg.GitHubRedirectURL == "" {
		return nil
	}
	message := "GitHub OAuth requires a master key, client ID, client secret and HTTPS callback URL ending in /api/git-connection/oauth/callback (HTTP is allowed on localhost)"
	if cfg.GitHubClientID == "" || cfg.GitHubClientSecret == "" || cfg.ConnectionMasterKey == "" {
		return errors.New(message)
	}
	callback, err := url.Parse(cfg.GitHubRedirectURL)
	if err != nil {
		return errors.New(message)
	}
	localHTTP := callback.Scheme == "http" && (callback.Hostname() == "localhost" || callback.Hostname() == "127.0.0.1")
	if callback.Hostname() == "" || callback.User != nil || callback.RawQuery != "" || callback.Fragment != "" || callback.Path != "/api/git-connection/oauth/callback" || (callback.Scheme != "https" && !localHTTP) {
		return errors.New(message)
	}
	return nil
}
