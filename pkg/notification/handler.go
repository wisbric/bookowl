package notification

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/bookowl/pkg/tenant"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/unread-count", h.UnreadCount)
	r.Get("/", h.List)
	r.Post("/mark-read", h.MarkRead)
	return r
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)

	count, err := h.svc.UnreadCount(r.Context(), store, userID)
	if err != nil {
		slog.Error("notification unread count error", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}
	httpserver.Respond(w, http.StatusOK, CountResponse{Count: count})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg , _ := httpserver.ParseOffsetParams(r)
	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)

	notifications, err := h.svc.List(r.Context(), store, userID, int32(pg.PageSize), int32(pg.Offset))
	if err != nil {
		slog.Error("notification list error", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}
	httpserver.Respond(w, http.StatusOK, notifications)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkReadRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)

	if req.All {
		if err := h.svc.MarkAllRead(r.Context(), store, userID); err != nil {
			slog.Error("notification mark all read error", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
			return
		}
	} else if len(req.IDs) > 0 {
		if err := h.svc.MarkRead(r.Context(), store, userID, req.IDs); err != nil {
			slog.Error("notification mark read error", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
			return
		}
	}

	httpserver.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func userIDFromContext(r *http.Request) uuid.UUID {
	pgUUID := tenant.UserIDFromContext(r.Context())
	if !pgUUID.Valid {
		return uuid.Nil
	}
	return uuid.UUID(pgUUID.Bytes)
}
