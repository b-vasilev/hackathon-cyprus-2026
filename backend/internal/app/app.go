package app

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/api"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/config"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/research"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
)

// Run opens storage, applies migrations, and serves until cancellation or an HTTP failure.
func Run(ctx context.Context, cfg config.Config, logger zerolog.Logger) error {
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	database, err := db.Open(startup, cfg.DatabasePath)
	cancel()
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			logger.Error().Err(closeErr).Msg("close database")
		}
	}()
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		return fmt.Errorf("migrate storage: %w", migrationErr)
	}
	files := &content.Store{Dir: cfg.ContentDir}
	if migrationErr := content.MigrateLegacy(ctx, database, files); migrationErr != nil {
		return fmt.Errorf("migrate passport content: %w", migrationErr)
	}
	policy, err := llmgateway.NewNetworkPolicy(cfg.LLMPrivateOrigins)
	if err != nil {
		return fmt.Errorf("configure LLM network access: %w", err)
	}
	var connections *llmgateway.Store
	if cfg.ConnectionMasterKey != "" {
		connections, err = llmgateway.NewStore(database, cfg.ConnectionMasterKey)
		if err != nil {
			return fmt.Errorf("initialize LLM connections: %w", err)
		}
	}
	client := &llm.Client{Policy: policy, Limiter: llmgateway.NewLimiter(8, 16, 2, 8)}
	collector := &source.Collector{WorkDir: cfg.FactoryWorkDir, RepomixBin: cfg.RepomixBin, GitHubToken: cfg.GitHubToken, MaxBytes: 128 << 20, MaxFiles: 20000, CacheScope: generation.PromptVersion}
	factory := generation.New(database, connections, client, collector, logger, files)
	factory.Research = &research.Runner{Policy: policy, Limiter: client.Limiter, WorkDir: cfg.FactoryWorkDir, RepomixBin: cfg.RepomixBin}
	var githubStore *githubconnection.Store
	if cfg.ConnectionMasterKey != "" {
		githubStore, err = githubconnection.NewStore(database, cfg.ConnectionMasterKey)
		if err != nil {
			return fmt.Errorf("initialize GitHub connection: %w", err)
		}
		collector.Token = githubStore.Token
	}
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); factory.Work(workerCtx) }()
	defer func() { stopWorker(); <-workerDone }()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.New(database, logger, cfg.CORSOrigins, &api.Factory{Access: &access.Store{DB: database}, Service: factory, Network: policy, FrontendDir: cfg.FrontendDir, GitHub: githubStore, GitHubOAuth: githubconnection.OAuthConfig{ClientID: cfg.GitHubClientID, ClientSecret: cfg.GitHubClientSecret, RedirectURL: cfg.GitHubRedirectURL}}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          stdlog.New(&logger, "", 0),
	}
	// Also close active connections if Serve exits unexpectedly.
	defer server.Close()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	logger.Info().Str("address", listener.Addr().String()).Msg("http server started")
	select {
	case err := <-serveErr:
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}
	logger.Info().Msg("http server stopping")
	shutdown, stop := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer stop()
	if err := server.Shutdown(shutdown); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown HTTP: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}
	return nil
}
