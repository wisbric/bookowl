package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
)

// apiKeyCreateRequest is the JSON body for POST /admin/api-keys.
type apiKeyCreateRequest struct {
	Description string `json:"description" validate:"required"`
	Role        string `json:"role"`
}

// apiKeyResponse is the JSON response for a single API key.
type apiKeyResponse struct {
	ID          uuid.UUID  `json:"id"`
	KeyPrefix   string     `json:"key_prefix"`
	Description string     `json:"description"`
	Role        string     `json:"role"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// apiKeyCreateResponse includes the raw key (shown once).
type apiKeyCreateResponse struct {
	apiKeyResponse
	RawKey string `json:"raw_key"`
}

func (h *Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	id := auth.FromContext(r.Context())
	if id == nil {
		httpserver.RespondError(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	keys, err := h.globalQ.ListAPIKeysByTenant(r.Context(), id.TenantID)
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list api keys")
		return
	}

	items := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		items = append(items, toAPIKeyResponse(k))
	}

	httpserver.Respond(w, http.StatusOK, map[string]any{
		"keys":  items,
		"count": len(items),
	})
}

func (h *Handler) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req apiKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Description == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "description is required")
		return
	}
	if req.Role == "" {
		req.Role = "admin"
	}

	id := auth.FromContext(r.Context())
	if id == nil {
		httpserver.RespondError(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	raw, hash, prefix := generateBookOwlAPIKey()

	key, err := h.globalQ.CreateAPIKey(r.Context(), dbglobal.CreateAPIKeyParams{
		TenantID:    id.TenantID,
		KeyHash:     hash,
		KeyPrefix:   prefix,
		Description: req.Description,
		Role:        req.Role,
	})
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create api key")
		return
	}

	httpserver.Respond(w, http.StatusCreated, apiKeyCreateResponse{
		apiKeyResponse: toAPIKeyResponse(key),
		RawKey:         raw,
	})
}

func (h *Handler) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid api key ID")
		return
	}

	id := auth.FromContext(r.Context())
	if id == nil {
		httpserver.RespondError(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	err = h.globalQ.DeleteAPIKey(r.Context(), dbglobal.DeleteAPIKeyParams{
		ID:       keyID,
		TenantID: id.TenantID,
	})
	if err != nil {
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete api key")
		return
	}

	httpserver.Respond(w, http.StatusNoContent, nil)
}

func toAPIKeyResponse(k dbglobal.ApiKey) apiKeyResponse {
	resp := apiKeyResponse{
		ID:          k.ID,
		KeyPrefix:   k.KeyPrefix,
		Description: k.Description,
		Role:        k.Role,
		CreatedAt:   k.CreatedAt,
	}
	if k.LastUsedAt.Valid {
		t := k.LastUsedAt.Time
		resp.LastUsed = &t
	}
	return resp
}

// generateBookOwlAPIKey creates a random API key with prefix "bw_".
func generateBookOwlAPIKey() (raw, hash, prefix string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	raw = fmt.Sprintf("bw_%x", b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	prefix = raw[:10]
	return
}
