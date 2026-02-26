package space

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

func (h *Handler) Routes(collectionRoutes chi.Router) chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
		r.Get("/tree", h.Tree)
		r.Get("/members", h.ListMembers)
		r.Post("/members", h.AddMember)
		r.Delete("/members/{userId}", h.RemoveMember)
		if collectionRoutes != nil {
			r.Mount("/collections", collectionRoutes)
		}
	})
	return r
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	store := NewStore(tenant.ConnFromContext(r.Context()))

	resp, err := h.svc.List(r.Context(), store)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
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

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	resp, err := h.svc.ListMembers(r.Context(), store, id)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	var req AddMemberRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	if err := h.svc.AddMember(r.Context(), store, id, req); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	spaceID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}
	userID, err := httpserver.URLParamUUID(r, "userId")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	if err := h.svc.RemoveMember(r.Context(), store, spaceID, userID); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Tree(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	tree, err := h.svc.GetTree(r.Context(), store, id)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, tree)
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
