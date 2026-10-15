package app

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/api"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/config"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
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
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.New(database, logger, cfg.CORSOrigins),
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
