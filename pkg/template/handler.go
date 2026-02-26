package template

import (
	"errors"
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
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

// SaveAsTemplateRoute returns a handler for POST /api/v1/documents/{id}/save-as-template.
func (h *Handler) SaveAsTemplateRoute() http.HandlerFunc {
	return h.SaveAsTemplate
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	store := NewStore(tenant.ConnFromContext(r.Context()))

	var spaceID *uuid.UUID
	if s := r.URL.Query().Get("space_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpserver.RespondError(w, http.StatusBadRequest, "error", "invalid space_id")
			return
		}
		spaceID = &id
	}

	docType := r.URL.Query().Get("doc_type")

	templates, err := h.svc.List(r.Context(), store, spaceID, docType)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, templates)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := tenant.UserIDFromContext(r.Context())

	resp, err := h.svc.Create(r.Context(), store, req, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusCreated, resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
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
	id, err := httpserver.URLParamUUID(r, "id")
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

func (h *Handler) SaveAsTemplate(w http.ResponseWriter, r *http.Request) {
	docID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req SaveAsTemplateRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := tenant.UserIDFromContext(r.Context())

	resp, err := h.svc.SaveDocumentAsTemplate(r.Context(), store, docID, req, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusCreated, resp)
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.RespondError(w, http.StatusNotFound, "error", err.Error())
	case errors.Is(err, ErrSystemReadOnly):
		httpserver.RespondError(w, http.StatusForbidden, "error", err.Error())
	default:
		slog.Error("template handler error", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
	}
}
