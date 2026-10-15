package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdkauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/auth"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "CLIProxyAPI configuration file")
	healthcheck := flag.Bool("healthcheck", false, "Check the local subscription service")
	flag.Parse()
	if *healthcheck {
		if err := checkHealth(os.Getenv("LLM_SUBSCRIPTION_KEY")); err != nil {
			log.Fatal(err)
		}
		return
	}

	cfg, err := sdkconfig.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if key := os.Getenv("LLM_SUBSCRIPTION_KEY"); key != "" {
		cfg.APIKeys = []string{key}
	}
	if len(cfg.APIKeys) == 0 {
		log.Fatal("LLM_SUBSCRIPTION_KEY or config api-keys is required")
	}
	if authDir := os.Getenv("LLM_SUBSCRIPTION_AUTH_DIR"); authDir != "" {
		cfg.AuthDir = authDir
	}
	if cfg.AuthDir == "" {
		log.Fatal("LLM_SUBSCRIPTION_AUTH_DIR or config auth-dir is required")
	}
	bridge := newCredentialBridge(cfg.AuthDir)
	created, err := bridge.reconcile()
	if err != nil {
		log.Fatalf("prepare Codex credentials: %v", err)
	}
	if created > 0 {
		log.Printf("created %d Codex proxy credential files", created)
	}
	sdkauth.RegisterTokenStore(bridge)
	service, err := cliproxy.NewBuilder().
		WithConfig(cfg).
		WithConfigPath(*configPath).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				created, err := bridge.reconcile()
				if err != nil {
					log.Printf("scan Codex credentials: %v", err)
				} else if created > 0 {
					log.Printf("created %d Codex proxy credential files", created)
				}
			}
		}
	}()
	if err := service.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
