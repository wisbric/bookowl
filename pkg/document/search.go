package document

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/bookowl/pkg/tenant"
)

// SearchHandler handles GET /api/v1/search.
type SearchHandler struct {
	svc *Service
}

func NewSearchHandler(svc *Service) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
httpserver.RespondError(w, http.StatusBadRequest, "error", "query parameter 'q' is required")
		return
	}

	var spaceID *uuid.UUID
	if raw := r.URL.Query().Get("space"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", "invalid space UUID")
			return
		}
		spaceID = &id
	}

	docType := r.URL.Query().Get("type")
	pg , _ := httpserver.ParseOffsetParams(r)

	store := NewStore(tenant.ConnFromContext(r.Context()))
	results, err := h.svc.Search(r.Context(), store, query, spaceID, docType, int32(pg.PageSize), int32(pg.Offset))
	if err != nil {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "search failed")
		return
	}
	httpserver.Respond(w, http.StatusOK, results)
}
