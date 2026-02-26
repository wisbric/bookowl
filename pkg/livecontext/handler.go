package livecontext

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wisbric/core/pkg/httpserver"

	"github.com/wisbric/bookowl/pkg/tenant"
)

// Handler proxies live context requests through BookOwl with Redis caching.
type Handler struct {
	client *Client
	cache  *Cache
}

func NewHandler(client *Client, cache *Cache) *Handler {
	return &Handler{client: client, cache: cache}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/oncall/{rosterId}", h.OnCall)
	r.Get("/service/{serviceName}", h.ServiceStatus)
	r.Get("/alerts", h.ActiveAlerts)
	r.Get("/incident/{incidentId}", h.IncidentStatus)
	return r
}

func (h *Handler) OnCall(w http.ResponseWriter, r *http.Request) {
	rosterID := chi.URLParam(r, "rosterId")
	if rosterID == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "error", "roster ID is required")
		return
	}

	t, cfg := resolveConfig(r)
	if cfg == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "NightOwl not configured for this tenant")
		return
	}

	key := OnCallKey(t.Slug, rosterID)
	h.proxyWithCache(w, r, key, func() (json.RawMessage, error) {
		return h.client.GetOnCall(r.Context(), *cfg, rosterID)
	})
}

func (h *Handler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	if serviceName == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "error", "service name is required")
		return
	}

	t, cfg := resolveConfig(r)
	if cfg == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "NightOwl not configured for this tenant")
		return
	}

	key := ServiceAlertsKey(t.Slug, serviceName)
	h.proxyWithCache(w, r, key, func() (json.RawMessage, error) {
		return h.client.GetServiceAlerts(r.Context(), *cfg, serviceName)
	})
}

func (h *Handler) ActiveAlerts(w http.ResponseWriter, r *http.Request) {
	severity := r.URL.Query().Get("severity")
	if severity == "" {
		severity = "critical,major"
	}
	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	t, cfg := resolveConfig(r)
	if cfg == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "NightOwl not configured for this tenant")
		return
	}

	key := ActiveAlertsKey(t.Slug, severity)
	h.proxyWithCache(w, r, key, func() (json.RawMessage, error) {
		return h.client.GetActiveAlerts(r.Context(), *cfg, severity, limit)
	})
}

func (h *Handler) IncidentStatus(w http.ResponseWriter, r *http.Request) {
	incidentID := chi.URLParam(r, "incidentId")
	if incidentID == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "error", "incident ID is required")
		return
	}

	t, cfg := resolveConfig(r)
	if cfg == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "NightOwl not configured for this tenant")
		return
	}

	key := IncidentKey(t.Slug, incidentID)
	h.proxyWithCache(w, r, key, func() (json.RawMessage, error) {
		return h.client.GetIncident(r.Context(), *cfg, incidentID)
	})
}

func resolveConfig(r *http.Request) (tenant.Tenant, *TenantNightOwlConfig) {
	t, ok := tenant.FromContext(r.Context())
	if !ok {
		return tenant.Tenant{}, nil
	}
	var cfg TenantNightOwlConfig
	if len(t.Config) > 0 {
		if err := json.Unmarshal(t.Config, &cfg); err != nil {
			slog.Error("parsing tenant NightOwl config", "tenant", t.Slug, "error", err)
			return t, nil
		}
	}
	if cfg.APIURL == "" || cfg.APIKey == "" {
		return t, nil
	}
	return t, &cfg
}

func (h *Handler) proxyWithCache(w http.ResponseWriter, r *http.Request, cacheKey string, fetch func() (json.RawMessage, error)) {
	ctx := r.Context()

	// Check cache first.
	cached, err := h.cache.Get(ctx, cacheKey)
	if err != nil {
		slog.Error("reading live context cache", "key", cacheKey, "error", err)
	}

	// Fresh cache hit — return immediately.
	if cached != nil && cached.IsFresh() {
		httpserver.Respond(w, http.StatusOK, Envelope{
			Data:     cached.Data,
			CachedAt: &cached.CachedAt,
			Source:   "cache",
		})
		return
	}

	// Stale or no cache — call NightOwl.
	data, err := fetch()
	if err != nil {
		slog.Warn("NightOwl call failed", "key", cacheKey, "error", err)

		// Graceful degradation: return stale cache if available.
		if cached != nil {
			httpserver.Respond(w, http.StatusOK, Envelope{
				Data:     cached.Data,
				CachedAt: &cached.CachedAt,
				Source:   "stale",
			})
			return
		}

		// No cache at all — unavailable.
		httpserver.Respond(w, http.StatusOK, Envelope{
			Data:   json.RawMessage(`{"status":"unavailable"}`),
			Source: "unavailable",
		})
		return
	}

	// Cache the fresh result.
	if cacheErr := h.cache.Set(ctx, cacheKey, data); cacheErr != nil {
		slog.Error("writing live context cache", "key", cacheKey, "error", cacheErr)
	}

	now := time.Now().UTC()
	httpserver.Respond(w, http.StatusOK, Envelope{
		Data:     data,
		CachedAt: &now,
		Source:   "live",
	})
}
