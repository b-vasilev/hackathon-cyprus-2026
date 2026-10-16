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
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/analysis"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/api"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/config"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learning"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learningpack"
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
	client := &llm.Client{Policy: policy, Limiter: llmgateway.NewLimiter(llmgateway.MaxConcurrentRequests, 60, llmgateway.MaxConcurrentRequests, 60)}
	collector := &source.Collector{WorkDir: cfg.FactoryWorkDir, RepomixBin: cfg.RepomixBin, GitHubToken: cfg.GitHubToken, MaxBytes: 128 << 20, MaxFiles: 20000, CacheScope: generation.PromptVersion}
	factory := generation.New(database, connections, client, collector, logger, files)
	factory.Research = &research.Runner{Policy: policy, Limiter: client.Limiter, WorkDir: cfg.FactoryWorkDir, RepomixBin: cfg.RepomixBin}
	factory.Analysis = &analysis.Runner{Policy: policy, Limiter: client.Limiter}
	var githubStore *githubconnection.Store
	if cfg.ConnectionMasterKey != "" {
		githubStore, err = githubconnection.NewStore(database, cfg.ConnectionMasterKey)
		if err != nil {
			return fmt.Errorf("initialize GitHub connection: %w", err)
		}
		collector.Token = githubStore.Token
	}
	workerCtx, stopWorker := context.WithCancel(ctx)
	packs := &learningpack.Service{DB: database, Passports: factory, Logger: logger, Background: workerCtx, Token: sourceToken(githubStore, cfg.GitHubToken)}
	factory.StartQuiz = func(ctx context.Context, revisionID string) error {
		if _, startErr := packs.Start(ctx, revisionID, connectionowner.UserID(ctx)); startErr != nil {
			return fmt.Errorf("start passport quiz: %w", startErr)
		}
		return nil
	}
	if recoverErr := packs.Recover(ctx); recoverErr != nil {
		stopWorker()
		return fmt.Errorf("start learning packs: %w", recoverErr)
	}
	// Deferred before stopWorker, so it runs after cancellation and before the database closes.
	defer packs.Wait()
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); factory.Work(workerCtx, cfg.GenerationWorkers) }()
	defer func() { stopWorker(); <-workerDone }()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.New(database, logger, cfg.CORSOrigins, &api.Factory{Access: &access.Store{DB: database}, PublicURL: cfg.PublicURL, Learning: &learning.Store{DB: database}, Service: factory, Packs: packs, Network: policy, FrontendDir: cfg.FrontendDir, GitHub: githubStore, GitHubOAuth: githubconnection.OAuthConfig{ClientID: cfg.GitHubClientID, ClientSecret: cfg.GitHubClientSecret, RedirectURL: cfg.GitHubRedirectURL}}),
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

// sourceToken keeps server credentials out of signed-in users' tasks.
func sourceToken(store *githubconnection.Store, fallback string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		if store != nil {
			token, err := store.Token(ctx)
			if err != nil {
				return "", fmt.Errorf("read GitHub connection token: %w", err)
			}
			if token != "" {
				return token, nil
			}
		}
		if connectionowner.UserID(ctx) != "" {
			return "", nil
		}
		return fallback, nil
	}
}
