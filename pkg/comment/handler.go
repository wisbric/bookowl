package comment

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/bookowl/pkg/tenant"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// CommentRoutes returns a function that registers comment routes inside a document /{id} route group.
func (h *Handler) CommentRoutes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/comments", h.List)
		r.Get("/comments/count", h.Count)
		r.Post("/comments", h.Create)
		r.Route("/comments/{commentId}", func(r chi.Router) {
			r.Put("/", h.Update)
			r.Delete("/", h.Delete)
			r.Post("/replies", h.Reply)
			r.Post("/resolve", h.Resolve)
			r.Post("/unresolve", h.Unresolve)
		})
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	docID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req CreateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)

	resp, err := h.svc.Create(r.Context(), store, docID, req, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusCreated, resp)
}

func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	docID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}
	commentID, err := httpserver.URLParamUUID(r, "commentId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req CreateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)

	resp, err := h.svc.Reply(r.Context(), store, docID, commentID, req, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusCreated, resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	commentID, err := httpserver.URLParamUUID(r, "commentId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req UpdateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)
	isAdmin := isAdminRole(r)

	resp, err := h.svc.Update(r.Context(), store, commentID, req, userID, isAdmin)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	commentID, err := httpserver.URLParamUUID(r, "commentId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)
	isAdmin := isAdminRole(r)

	if err := h.svc.Delete(r.Context(), store, commentID, userID, isAdmin); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	commentID, err := httpserver.URLParamUUID(r, "commentId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := userIDFromContext(r)

	if err := h.svc.Resolve(r.Context(), store, commentID, userID); err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (h *Handler) Unresolve(w http.ResponseWriter, r *http.Request) {
	commentID, err := httpserver.URLParamUUID(r, "commentId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))

	if err := h.svc.Unresolve(r.Context(), store, commentID); err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, map[string]string{"status": "unresolved"})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	docID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	includeResolved := r.URL.Query().Get("include_resolved") == "true"

	store := NewStore(tenant.ConnFromContext(r.Context()))
	comments, err := h.svc.List(r.Context(), store, docID, includeResolved)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, comments)
}

func (h *Handler) Count(w http.ResponseWriter, r *http.Request) {
	docID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	count, err := h.svc.Count(r.Context(), store, docID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, CountResponse{Count: count})
}

func userIDFromContext(r *http.Request) uuid.UUID {
	pgUUID := tenant.UserIDFromContext(r.Context())
	if !pgUUID.Valid {
		return uuid.Nil
	}
	return uuid.UUID(pgUUID.Bytes)
}

func isAdminRole(r *http.Request) bool {
	identity := auth.FromContext(r.Context())
	if identity == nil {
		return false
	}
	return identity.Role == "admin"
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
httpserver.RespondError(w, http.StatusNotFound, "error", err.Error())
	case errors.Is(err, ErrReplyToReply):
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
	case errors.Is(err, ErrNotTopLevel):
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
	case errors.Is(err, ErrEmptyBody):
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
	case errors.Is(err, ErrEditWindowExpired):
httpserver.RespondError(w, http.StatusForbidden, "error", err.Error())
	case errors.Is(err, ErrNotAuthor):
httpserver.RespondError(w, http.StatusForbidden, "error", err.Error())
	default:
		slog.Error("comment handler error", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
	}
}
