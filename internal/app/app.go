package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"encoding/json"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	coretelemetry "github.com/wisbric/core/pkg/telemetry"
	"github.com/wisbric/core/pkg/version"

	"github.com/wisbric/bookowl/internal/admin"
	"github.com/wisbric/bookowl/internal/authadapter"
	"github.com/wisbric/bookowl/internal/config"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/internal/integration"
	"github.com/wisbric/bookowl/internal/platform"
	"github.com/wisbric/bookowl/pkg/collection"
	"github.com/wisbric/bookowl/pkg/comment"
	"github.com/wisbric/bookowl/pkg/document"
	"github.com/wisbric/bookowl/pkg/image"
	"github.com/wisbric/bookowl/pkg/livecontext"
	"github.com/wisbric/bookowl/pkg/notification"
	"github.com/wisbric/bookowl/pkg/space"
	"github.com/wisbric/bookowl/pkg/storage"
	"github.com/wisbric/bookowl/pkg/template"
	"github.com/wisbric/bookowl/pkg/tenant"
	"github.com/wisbric/bookowl/pkg/user"
)

type App struct {
	cfg        config.Config
	plat       *platform.Platform
	router     *chi.Mux
	metricsReg *prometheus.Registry
	backend    storage.Backend
	sessionMgr *auth.SessionManager
	authStore  auth.Storage
	startedAt  time.Time
	stopCh     chan struct{}
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	plat, err := platform.New(ctx, cfg.DatabaseURL, cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("initializing platform: %w", err)
	}

	backend, err := initStorageBackend(ctx, cfg)
	if err != nil {
		plat.Close()
		return nil, fmt.Errorf("initializing storage backend: %w", err)
	}

	// Session manager (core's unified cookie-based sessions).
	secretKey := cfg.SecretKey
	if secretKey == "" && cfg.DevMode {
		secretKey = auth.GenerateDevSecret()
		slog.Info("using auto-generated dev secret for session management")
	}
	var sessionMgr *auth.SessionManager
	if secretKey != "" {
		sessionTTL, err := time.ParseDuration(cfg.SessionTTL)
		if err != nil {
			sessionTTL = 12 * time.Hour
		}
		sessionMgr, err = auth.NewSessionManager(secretKey, sessionTTL)
		if err != nil {
			plat.Close()
			return nil, fmt.Errorf("creating session manager: %w", err)
		}
		slog.Info("session management enabled", "ttl", sessionTTL)
	} else {
		slog.Info("session management disabled (no BOOKOWL_SECRET_KEY set)")
	}

	// Auth storage adapter.
	authStore := authadapter.New(plat.DB)

	// Metrics registry (no BookOwl-specific metrics yet — shared only).
	metricsReg := coretelemetry.NewMetricsRegistry()

	app := &App{
		cfg:        cfg,
		plat:       plat,
		metricsReg: metricsReg,
		backend:    backend,
		sessionMgr: sessionMgr,
		authStore:  authStore,
		startedAt:  time.Now(),
		stopCh:     make(chan struct{}),
	}
	app.setupRouter()
	return app, nil
}

func initStorageBackend(ctx context.Context, cfg config.Config) (storage.Backend, error) {
	switch cfg.StorageBackend {
	case "s3":
		return storage.NewS3Backend(ctx, storage.S3Config{
			Endpoint:        cfg.StorageS3Endpoint,
			Bucket:          cfg.StorageS3Bucket,
			Region:          cfg.StorageS3Region,
			AccessKeyID:     cfg.StorageS3AccessKeyID,
			SecretAccessKey: cfg.StorageS3SecretAccessKey,
			PublicURL:       cfg.StorageS3PublicURL,
			UsePresign:      cfg.StorageS3UsePresign,
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

	// Prometheus metrics (no auth).
	r.Handle("/metrics", promhttp.HandlerFor(a.metricsReg, promhttp.HandlerOpts{}))

	// Health endpoints (no auth).
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpserver.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.plat.DB.Ping(r.Context()); err != nil {
			httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "database unavailable")
			return
		}
		httpserver.Respond(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	// Public status endpoint (no auth required — used by about page).
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(a.startedAt)
		httpserver.Respond(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"version":    version.Version,
			"commit_sha": version.Commit,
			"uptime":     uptime.Truncate(time.Second).String(),
		})
	})

	// Auth config endpoint (no auth required — frontend uses this to decide OIDC vs dev mode).
	r.Get("/api/v1/auth/config", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"oidc_enabled": a.cfg.OIDCIssuerURL != "",
		}
		if a.cfg.OIDCIssuerURL != "" {
			resp["oidc_authority"] = a.cfg.OIDCIssuerURL
			resp["oidc_client_id"] = a.cfg.OIDCClientID
		}
		httpserver.Respond(w, http.StatusOK, resp)
	})

	// Client config endpoint (public, no auth required — frontend fetches runtime config).
	r.Get("/api/v1/config/client", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{}
		if a.cfg.CollabWsURL != "" {
			resp["collab_ws_url"] = a.cfg.CollabWsURL
		}
		httpserver.Respond(w, http.StatusOK, resp)
	})

	// Auth routes (public, no auth required).
	if a.sessionMgr != nil {
		rateLimiter := auth.NewRateLimiter(a.plat.Redis, 10, 15*time.Minute)
		localAdminH := auth.NewLocalAdminHandler(a.sessionMgr, a.authStore, slog.Default(), rateLimiter)
		r.Post("/auth/local", localAdminH.HandleLocalLogin)
		r.Post("/auth/change-password", localAdminH.HandleChangePassword)
		r.Get("/auth/config", localAdminH.HandleAuthConfig)

		loginH := auth.NewLoginHandler(a.sessionMgr, a.authStore, slog.Default(), a.cfg.OIDCIssuerURL != "")
		r.Post("/auth/login", loginH.HandleLogin)
		r.Get("/auth/me", loginH.HandleMe)
		r.Post("/auth/logout", loginH.HandleLogout)
	}

	// Auth + tenant middleware.
	authMW := auth.Middleware(a.sessionMgr, nil, nil, a.authStore, slog.Default())
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

	// Template handler.
	templateSvc := template.NewService()
	templateHandler := template.NewHandler(templateSvc)

	// Comment + notification handlers.
	commentSvc := comment.NewService()
	commentHandler := comment.NewHandler(commentSvc)
	notificationSvc := notification.NewService()
	notificationHandler := notification.NewHandler(notificationSvc)

	// Integration handler (NightOwl → BookOwl).
	integrationHandler := integration.NewHandler(documentSvc, a.cfg.PublicURL)

	// Live Context handler (BookOwl → NightOwl proxy with cache).
	liveContextClient := livecontext.NewClient()
	liveContextCache := livecontext.NewCache(a.plat.Redis)
	liveContextHandler := livecontext.NewHandler(liveContextClient, liveContextCache)

	// Image handler.
	imageHandler := image.NewHandler(a.backend)

	// Profile handler.
	profileHandler := user.NewProfileHandler(a.plat.DB)

	// Admin handler.
	adminHandler := admin.NewHandler(a.plat.DB, liveContextClient, a.cfg.SecretKey, a.plat.Redis)

	// API v1 routes (require auth + tenant).
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMW)
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

		// Health check (DB, Redis, NightOwl).
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			checks := map[string]string{}

			if err := a.plat.DB.Ping(r.Context()); err != nil {
				checks["database"] = "unhealthy"
			} else {
				checks["database"] = "healthy"
			}

			if err := a.plat.Redis.Ping(r.Context()).Err(); err != nil {
				checks["redis"] = "unhealthy"
			} else {
				checks["redis"] = "healthy"
			}

			t, ok := tenant.FromContext(r.Context())
			if ok {
				var cfg livecontext.TenantNightOwlConfig
				if len(t.Config) > 0 {
					_ = json.Unmarshal(t.Config, &cfg)
				}
				if cfg.APIURL != "" && cfg.APIKey != "" {
					_, err := liveContextClient.GetActiveAlerts(r.Context(), cfg, "critical", 1)
					if err != nil {
						checks["nightowl"] = "unhealthy"
					} else {
						checks["nightowl"] = "healthy"
					}
				} else {
					checks["nightowl"] = "not configured"
				}
			} else {
				checks["nightowl"] = "unknown"
			}

			status := "ok"
			for _, v := range checks {
				if v == "unhealthy" {
					status = "degraded"
					break
				}
			}

			httpserver.Respond(w, http.StatusOK, map[string]any{
				"status": status,
				"checks": checks,
			})
		})

		// Collab token — returns a short-lived JWT for WebSocket auth.
		r.Get("/collab/token", func(w http.ResponseWriter, r *http.Request) {
			if a.sessionMgr == nil {
				httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "session management not configured")
				return
			}

			// Try to extract the raw JWT from the wisbric_session cookie.
			if cookie, err := r.Cookie(auth.CookieName); err == nil {
				if _, err := a.sessionMgr.ValidateToken(cookie.Value); err == nil {
					httpserver.Respond(w, http.StatusOK, map[string]string{"token": cookie.Value})
					return
				}
			}

			// Fallback: mint a short-lived token from the authenticated identity.
			id := auth.FromContext(r.Context())
			if id == nil {
				httpserver.RespondError(w, http.StatusUnauthorized, "error", "authentication required")
				return
			}
			userIDStr := ""
			if id.UserID != nil {
				userIDStr = id.UserID.String()
			}
			token, err := a.sessionMgr.MintShortLived(auth.SessionClaims{
				Subject:    id.Subject,
				Email:      id.Email,
				Role:       id.Role,
				TenantSlug: id.TenantSlug,
				TenantID:   id.TenantID.String(),
				UserID:     userIDStr,
				Method:     id.Method,
			}, 5*time.Minute)
			if err != nil {
				httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to mint collab token")
				return
			}
			httpserver.Respond(w, http.StatusOK, map[string]string{"token": token})
		})

		// Spaces (with nested collections).
		r.Mount("/spaces", spaceHandler.Routes(collectionHandler.Routes()))

		// Documents (includes save-as-template and comment sub-routes).
		r.Mount("/documents", documentHandler.Routes(
			func(r chi.Router) {
				r.Post("/save-as-template", templateHandler.SaveAsTemplate)
			},
			commentHandler.CommentRoutes(),
		))

		// Templates.
		r.Mount("/templates", templateHandler.Routes())

		// Search.
		r.Get("/search", searchHandler.Search)

		// Integration endpoints (NightOwl calls BookOwl; requires admin role).
		r.Mount("/integration", integrationHandler.Routes())

		// Live Context proxy (frontend calls BookOwl → NightOwl).
		r.Mount("/live-context", liveContextHandler.Routes())

		// Images.
		r.Mount("/images", imageHandler.Routes())

		// Profile.
		r.Mount("/profile", profileHandler.Routes())

		// Notifications.
		r.Mount("/notifications", notificationHandler.Routes())

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
