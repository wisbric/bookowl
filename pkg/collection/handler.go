package collection

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wisbric/core/pkg/httpserver"

	"github.com/wisbric/bookowl/pkg/tenant"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns a chi.Router for collection endpoints.
// Mounted under /spaces/{id}/collections, so the space ID is in URL param "id".
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Route("/{collectionId}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	spaceID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req CreateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := tenant.UserIDFromContext(r.Context())

	resp, err := h.svc.Create(r.Context(), store, spaceID, req, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusCreated, resp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	spaceID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	resp, err := h.svc.List(r.Context(), store, spaceID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "collectionId")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	resp, err := h.svc.Get(r.Context(), store, id)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "collectionId")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req UpdateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	resp, err := h.svc.Update(r.Context(), store, id, req)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "collectionId")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	if err := h.svc.Delete(r.Context(), store, id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.RespondError(w, http.StatusNotFound, "error", err.Error())
	case errors.Is(err, ErrSlugConflict):
		httpserver.RespondError(w, http.StatusConflict, "error", err.Error())
	default:
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
	}
}
