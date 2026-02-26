package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/wisbric/bookowl/internal/app"
	"github.com/wisbric/bookowl/internal/config"
	"github.com/wisbric/bookowl/internal/migrate"
	"github.com/wisbric/bookowl/internal/seed"
	"github.com/wisbric/core/pkg/telemetry"
	"github.com/wisbric/core/pkg/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bookowl: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	slog.SetDefault(telemetry.NewLogger(cfg.LogFormat, cfg.LogLevel))

	ctx := context.Background()

	// Tracing (noop if OTEL_ENDPOINT is empty).
	shutdownTracer, err := telemetry.InitTracer(ctx, cfg.OTLPEndpoint, "bookowl", version.Version)
	if err != nil {
		return fmt.Errorf("initializing tracer: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(shutdownCtx); err != nil {
			slog.Error("shutting down tracer", "error", err)
		}
	}()

	switch cfg.Mode {
	case "api":
		return runAPI(ctx, cfg)
	case "seed", "seed-demo":
		return seed.Run(ctx, cfg)
	case "migrate-nightowl-runbooks":
		return migrate.RunRunbookMigration(ctx, cfg)
	default:
		return fmt.Errorf("unknown mode: %s", cfg.Mode)
	}
}

func runAPI(ctx context.Context, cfg config.Config) error {
	application, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer application.Close()
	return application.Run()
}
