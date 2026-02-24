package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/wisbric/bookowl/internal/admin"
	"github.com/wisbric/bookowl/internal/auth"
	"github.com/wisbric/bookowl/internal/config"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/internal/httpserver"
	"github.com/wisbric/bookowl/internal/integration"
	"github.com/wisbric/bookowl/internal/platform"
	"github.com/wisbric/bookowl/internal/version"
	"github.com/wisbric/bookowl/pkg/collection"
	"github.com/wisbric/bookowl/pkg/document"
	"github.com/wisbric/bookowl/pkg/image"
	"github.com/wisbric/bookowl/pkg/livecontext"
	"github.com/wisbric/bookowl/pkg/space"
	"github.com/wisbric/bookowl/pkg/storage"
	"github.com/wisbric/bookowl/pkg/tenant"
)

type App struct {
	cfg     config.Config
	plat    *platform.Platform
	router  *chi.Mux
	backend storage.Backend
	stopCh  chan struct{}
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	plat, err := platform.New(ctx, cfg.DBURL, cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("initializing platform: %w", err)
	}

	backend, err := initStorageBackend(ctx, cfg)
	if err != nil {
		plat.Close()
		return nil, fmt.Errorf("initializing storage backend: %w", err)
	}

	app := &App{
		cfg:     cfg,
		plat:    plat,
		backend: backend,
		stopCh:  make(chan struct{}),
	}
	app.setupRouter()
	return app, nil
}

func initStorageBackend(ctx context.Context, cfg config.Config) (storage.Backend, error) {
	switch cfg.StorageBackend {
	case "s3":
		return storage.NewS3Backend(ctx, storage.S3Config{
			Endpoint:       cfg.StorageS3Endpoint,
			Bucket:         cfg.StorageS3Bucket,
			Region:         cfg.StorageS3Region,
			AccessKeyID:    cfg.StorageS3AccessKeyID,
			SecretAccessKey: cfg.StorageS3SecretAccessKey,
			PublicURL:      cfg.StorageS3PublicURL,
			UsePresign:     cfg.StorageS3UsePresign,
		})
	default:
		return storage.NewLocalBackend(cfg.StorageLocalPath), nil
	}
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

	// Image handler.
	imageHandler := image.NewHandler(a.backend)

	// Admin handler.
	adminHandler := admin.NewHandler(a.plat.DB, liveContextClient)

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

		// Images.
		r.Mount("/images", imageHandler.Routes())

		// Admin.
		r.Mount("/admin", adminHandler.Routes())
	})

	a.router = r
}

func (a *App) Run() error {
	addr := fmt.Sprintf("%s:%d", a.cfg.Host, a.cfg.Port)
	slog.Info("starting BookOwl API", "addr", addr, "mode", a.cfg.Mode, "version", version.Version, "storage", a.backend.Name())

	// Start weekly orphan cleanup.
	go a.runOrphanCleanup()

	return http.ListenAndServe(addr, a.router)
}

func (a *App) Close() {
	close(a.stopCh)
	a.plat.Close()
}

// runOrphanCleanup runs the image orphan cleanup job on a weekly ticker.
func (a *App) runOrphanCleanup() {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			conn, err := a.plat.DB.Acquire(ctx)
			if err != nil {
				slog.Error("orphan cleanup: acquiring connection", "error", err)
				cancel()
				continue
			}
			q := dbtenant.New(conn)
			deleted, err := image.CleanupOrphans(ctx, q, a.backend, 24*time.Hour)
			conn.Release()
			cancel()
			if err != nil {
				slog.Error("orphan cleanup failed", "error", err)
			} else if deleted > 0 {
				slog.Info("orphan cleanup completed", "deleted", deleted)
			}
		case <-a.stopCh:
			return
		}
	}
}
