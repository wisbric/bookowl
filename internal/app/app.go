package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"encoding/json"

	"github.com/wisbric/bookowl/internal/admin"
	"github.com/wisbric/bookowl/internal/auth"
	"github.com/wisbric/bookowl/internal/authhandler"
	"github.com/wisbric/bookowl/internal/config"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/internal/httpserver"
	"github.com/wisbric/bookowl/internal/integration"
	"github.com/wisbric/bookowl/internal/platform"
	"github.com/wisbric/bookowl/internal/session"
	"github.com/wisbric/bookowl/internal/version"
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
	cfg          config.Config
	plat         *platform.Platform
	router       *chi.Mux
	backend      storage.Backend
	oidcVerifier *oidc.IDTokenVerifier
	sess         *session.Manager
	stopCh       chan struct{}
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

	var oidcVerifier *oidc.IDTokenVerifier
	if cfg.OIDCIssuer != "" {
		provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
		if err != nil {
			plat.Close()
			return nil, fmt.Errorf("initializing OIDC provider: %w", err)
		}
		oidcVerifier = provider.Verifier(&oidc.Config{
			ClientID: cfg.OIDCClientID,
		})
		slog.Info("OIDC authentication enabled", "issuer", cfg.OIDCIssuer, "client_id", cfg.OIDCClientID)
	} else {
		slog.Info("OIDC authentication disabled (no BOOKOWL_OIDC_ISSUER set)")
	}

	// Session manager for local admin and OIDC sessions.
	var sess *session.Manager
	secretKey := cfg.SecretKey
	if secretKey == "" && cfg.DevMode {
		secretKey = "bookowl-dev-secret-key-do-not-use-in-production"
		slog.Info("using dev secret key for session management")
	}
	if secretKey != "" {
		sessionTTL, err := time.ParseDuration(cfg.SessionTTL)
		if err != nil {
			sessionTTL = 12 * time.Hour
		}
		sess = session.NewManager(secretKey, sessionTTL)
		slog.Info("session management enabled", "ttl", sessionTTL)
	} else {
		slog.Info("session management disabled (no BOOKOWL_SECRET_KEY set)")
	}

	app := &App{
		cfg:          cfg,
		plat:         plat,
		backend:      backend,
		oidcVerifier: oidcVerifier,
		sess:         sess,
		stopCh:       make(chan struct{}),
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

	// Auth config endpoint (no auth required — frontend uses this to decide OIDC vs dev mode).
	r.Get("/api/v1/auth/config", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"oidc_enabled": a.cfg.OIDCIssuer != "",
		}
		if a.cfg.OIDCIssuer != "" {
			resp["oidc_authority"] = a.cfg.OIDCIssuer
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

	// Auth handler (public, no auth required).
	authH := authhandler.NewHandler(a.plat.DB, a.sess, a.plat.Redis)
	r.Mount("/auth", authH.Routes())

	// Auth + tenant middleware.
	authMW := auth.NewMiddleware(a.plat.DB, a.cfg.DevMode, a.oidcVerifier, a.sess)
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
			if a.sess == nil {
				httpserver.RespondError(w, http.StatusServiceUnavailable, "session management not configured")
				return
			}

			// Try to extract the raw JWT from the bw_session cookie.
			if cookie, err := r.Cookie(session.CookieName); err == nil {
				if _, err := a.sess.Validate(cookie.Value); err == nil {
					httpserver.Respond(w, http.StatusOK, map[string]string{"token": cookie.Value})
					return
				}
			}

			// Fallback: mint a short-lived token from the authenticated identity.
			id, ok := auth.IdentityFromContext(r.Context())
			if !ok {
				httpserver.RespondError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			token, err := a.sess.MintShortLived(id.TenantSlug, id.UserID, id.Role, id.Method, 5*time.Minute)
			if err != nil {
				httpserver.RespondError(w, http.StatusInternalServerError, "failed to mint collab token")
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
