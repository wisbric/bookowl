package document

import (
	"errors"
	"log/slog"
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

func (h *Handler) Routes(extraDocRoutes ...func(chi.Router)) chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
		r.Get("/history", h.ListVersions)
		r.Get("/history/{versionId}", h.GetVersion)
		r.Post("/restore/{versionId}", h.Restore)
		for _, fn := range extraDocRoutes {
			fn(r)
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
	pg , _ := httpserver.ParseOffsetParams(r)

	store := NewStore(tenant.ConnFromContext(r.Context()))
	items, total, err := h.svc.List(r.Context(), store, int32(pg.PageSize), int32(pg.Offset))
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, httpserver.OffsetPage[Response]{
		Items:      items,
		TotalItems: int(total),
		PageSize:   pg.PageSize,
		Page:       pg.Page,
	})
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
	userID := tenant.UserIDFromContext(r.Context())

	resp, err := h.svc.Update(r.Context(), store, id, req, userID)
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

func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}
	pg , _ := httpserver.ParseOffsetParams(r)

	store := NewStore(tenant.ConnFromContext(r.Context()))
	versions, err := h.svc.ListVersions(r.Context(), store, id, int32(pg.PageSize), int32(pg.Offset))
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, versions)
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	versionID, err := httpserver.URLParamUUID(r, "versionId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	resp, err := h.svc.GetVersion(r.Context(), store, versionID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	docID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}
	versionID, err := httpserver.URLParamUUID(r, "versionId")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	store := NewStore(tenant.ConnFromContext(r.Context()))
	userID := tenant.UserIDFromContext(r.Context())

	resp, err := h.svc.Restore(r.Context(), store, docID, versionID, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	httpserver.Respond(w, http.StatusOK, resp)
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
httpserver.RespondError(w, http.StatusNotFound, "error", err.Error())
	case errors.Is(err, ErrSlugConflict):
httpserver.RespondError(w, http.StatusConflict, "error", err.Error())
	default:
		slog.Error("document handler error", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
	}
}
