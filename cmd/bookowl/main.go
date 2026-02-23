package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/wisbric/bookowl/internal/app"
	"github.com/wisbric/bookowl/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bookowl: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()

	switch cfg.Mode {
	case "api":
		return runAPI(ctx, cfg)
	case "seed", "seed-demo":
		slog.Info("seed mode not yet implemented", "mode", cfg.Mode)
		return nil
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
