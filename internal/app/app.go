package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/wisbric/bookowl/internal/auth"
	"github.com/wisbric/bookowl/internal/config"
	"github.com/wisbric/bookowl/internal/httpserver"
	"github.com/wisbric/bookowl/internal/integration"
	"github.com/wisbric/bookowl/internal/platform"
	"github.com/wisbric/bookowl/internal/version"
	"github.com/wisbric/bookowl/pkg/collection"
	"github.com/wisbric/bookowl/pkg/document"
	"github.com/wisbric/bookowl/pkg/livecontext"
	"github.com/wisbric/bookowl/pkg/space"
	"github.com/wisbric/bookowl/pkg/tenant"
)

type App struct {
	cfg    config.Config
	plat   *platform.Platform
	router *chi.Mux
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	plat, err := platform.New(ctx, cfg.DBURL, cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("initializing platform: %w", err)
	}

	app := &App{
		cfg:  cfg,
		plat: plat,
	}
	app.setupRouter()
	return app, nil
}

func (a *App) setupRouter() {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// Health endpoints (no auth).
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpserver.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.plat.DB.Ping(r.Context()); err != nil {
			httpserver.RespondError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		httpserver.Respond(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	// Auth + tenant middleware.
	authMW := auth.NewMiddleware(a.plat.DB, a.cfg.DevMode)
	tenantMW := tenant.NewResolveMiddleware(a.plat.DB, a.plat.DB)

	// Domain services.
	spaceSvc := space.NewService()
	collectionSvc := collection.NewService()
	documentSvc := document.NewService()

	// Domain handlers.
	spaceHandler := space.NewHandler(spaceSvc)
	collectionHandler := collection.NewHandler(collectionSvc)
	documentHandler := document.NewHandler(documentSvc)
	searchHandler := document.NewSearchHandler(documentSvc)

	// Integration handler (NightOwl → BookOwl).
	integrationHandler := integration.NewHandler(documentSvc)

	// Live Context handler (BookOwl → NightOwl proxy with cache).
	liveContextClient := livecontext.NewClient()
	liveContextCache := livecontext.NewCache(a.plat.Redis)
	liveContextHandler := livecontext.NewHandler(liveContextClient, liveContextCache)

	// API v1 routes (require auth + tenant).
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(tenantMW.Resolve)

		// System.
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			httpserver.Respond(w, http.StatusOK, map[string]string{"pong": "ok"})
		})
		r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
			httpserver.Respond(w, http.StatusOK, map[string]string{
				"version": version.Version,
				"commit":  version.Commit,
			})
		})

		// Spaces (with nested collections).
		r.Mount("/spaces", spaceHandler.Routes(collectionHandler.Routes()))

		// Documents.
		r.Mount("/documents", documentHandler.Routes())

		// Search.
		r.Get("/search", searchHandler.Search)

		// Integration endpoints (NightOwl calls BookOwl; requires admin role).
		r.Mount("/integration", integrationHandler.Routes())

		// Live Context proxy (frontend calls BookOwl → NightOwl).
		r.Mount("/live-context", liveContextHandler.Routes())
	})

	a.router = r
}

func (a *App) Run() error {
	addr := fmt.Sprintf("%s:%d", a.cfg.Host, a.cfg.Port)
	slog.Info("starting BookOwl API", "addr", addr, "mode", a.cfg.Mode, "version", version.Version)
	return http.ListenAndServe(addr, a.router)
}

func (a *App) Close() {
	a.plat.Close()
}
