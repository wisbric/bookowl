package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	"github.com/wisbric/bookowl/pkg/livecontext"
	"github.com/wisbric/bookowl/pkg/tenant"
)

// Handler serves /api/v1/admin/ endpoints for tenant configuration.
type Handler struct {
	globalQ   *dbglobal.Queries
	client    *livecontext.Client
	secretKey string        // hex-encoded 32-byte key for AES-256-GCM encryption
	redis     *redis.Client // for cache invalidation
}

func NewHandler(globalDB dbglobal.DBTX, client *livecontext.Client, secretKey string, rdb *redis.Client) *Handler {
	return &Handler{
		globalQ:   dbglobal.New(globalDB),
		client:    client,
		secretKey: secretKey,
		redis:     rdb,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(requireAdmin)
	r.Get("/config", h.GetConfig)
	r.Put("/config", h.UpdateConfig)
	r.Post("/config/test-nightowl", h.TestNightOwl)

	// OIDC config endpoints.
	r.Get("/config/oidc", h.GetOIDCConfig)
	r.Put("/config/oidc", h.UpdateOIDCConfig)
	r.Post("/config/oidc/test", h.TestOIDCConfig)
	return r
}

// requireAdmin ensures the caller has role=admin.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := auth.FromContext(r.Context())
		if identity == nil || identity.Role != "admin" {
			httpserver.RespondError(w, http.StatusForbidden, "error", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ConfigResponse is returned by GET /admin/config.
type ConfigResponse struct {
	NightOwlAPIURL string `json:"nightowl_api_url"`
	NightOwlAPIKey string `json:"nightowl_api_key"` // masked for display
}

// ConfigUpdateRequest is the body for PUT /admin/config.
type ConfigUpdateRequest struct {
	NightOwlAPIURL string `json:"nightowl_api_url"`
	NightOwlAPIKey string `json:"nightowl_api_key"`
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	t, ok := tenant.FromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	var cfg livecontext.TenantNightOwlConfig
	if len(t.Config) > 0 {
		if err := json.Unmarshal(t.Config, &cfg); err != nil {
			slog.Error("parsing tenant config", "tenant", t.Slug, "error", err)
		}
	}

	httpserver.Respond(w, http.StatusOK, ConfigResponse{
		NightOwlAPIURL: cfg.APIURL,
		NightOwlAPIKey: maskKey(cfg.APIKey),
	})
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigUpdateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	t, ok := tenant.FromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	// Merge new config into existing config JSONB.
	existing := map[string]any{}
	if len(t.Config) > 0 {
		if err := json.Unmarshal(t.Config, &existing); err != nil {
			slog.Error("parsing existing tenant config", "tenant", t.Slug, "error", err)
		}
	}

	if req.NightOwlAPIURL != "" {
		existing["nightowl_api_url"] = req.NightOwlAPIURL
	}
	if req.NightOwlAPIKey != "" {
		existing["nightowl_api_key"] = req.NightOwlAPIKey
	}

	configJSON, err := json.Marshal(existing)
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to serialize config")
		return
	}

	_, err = h.globalQ.UpdateTenant(r.Context(), dbglobal.UpdateTenantParams{
		ID:     t.ID,
		Name:   t.Name,
		Config: configJSON,
	})
	if err != nil {
		slog.Error("updating tenant config", "tenant", t.Slug, "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to update config")
		return
	}

	httpserver.Respond(w, http.StatusOK, map[string]string{"status": "updated"})
}

// TestNightOwlResponse is returned by POST /admin/config/test-nightowl.
type TestNightOwlResponse struct {
	Status  string `json:"status"`  // "ok" or "error"
	Message string `json:"message"` // detail
}

func (h *Handler) TestNightOwl(w http.ResponseWriter, r *http.Request) {
	t, ok := tenant.FromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	var cfg livecontext.TenantNightOwlConfig
	if len(t.Config) > 0 {
		if err := json.Unmarshal(t.Config, &cfg); err != nil {
			slog.Error("parsing tenant config for test", "tenant", t.Slug, "error", err)
		}
	}

	if cfg.APIURL == "" || cfg.APIKey == "" {
		httpserver.Respond(w, http.StatusOK, TestNightOwlResponse{
			Status:  "error",
			Message: "NightOwl API URL and API key are not configured",
		})
		return
	}

	// Ping NightOwl by calling its health or ping endpoint.
	_, err := h.client.GetActiveAlerts(r.Context(), cfg, "critical", 1)
	if err != nil {
		httpserver.Respond(w, http.StatusOK, TestNightOwlResponse{
			Status:  "error",
			Message: "Failed to reach NightOwl: " + err.Error(),
		})
		return
	}

	httpserver.Respond(w, http.StatusOK, TestNightOwlResponse{
		Status:  "ok",
		Message: "Successfully connected to NightOwl at " + cfg.APIURL,
	})
}

// maskKey returns a masked version of an API key for display.
func maskKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "****" + key[len(key)-4:]
}
