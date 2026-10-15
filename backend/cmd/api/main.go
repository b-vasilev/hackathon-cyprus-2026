package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/app"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/applog"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	logger, err := applog.New(os.Stdout, cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error().Err(err).Msg("application stopped")
		return fmt.Errorf("run application: %w", err)
	}
	return nil
}
