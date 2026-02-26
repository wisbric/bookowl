package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/bookowl/internal/db"
	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/bookowl/pkg/tenant"
)

type ProfileHandler struct {
	globalQ *dbglobal.Queries
}

func NewProfileHandler(globalDB dbglobal.DBTX) *ProfileHandler {
	return &ProfileHandler{
		globalQ: dbglobal.New(globalDB),
	}
}

func (h *ProfileHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetProfile)
	r.Put("/", h.UpdateProfile)
	r.Get("/tokens", h.ListTokens)
	r.Post("/tokens", h.CreateToken)
	r.Delete("/tokens/{id}", h.RevokeToken)
	return r
}

// ProfileResponse is returned by GET /api/v1/profile.
type ProfileResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Timezone    string `json:"timezone"`
	Role        string `json:"role"`
	AuthMethod  string `json:"auth_method"`
	CreatedAt   string `json:"created_at"`
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	identity := auth.FromContext(r.Context())
	if identity == nil {
httpserver.RespondError(w, http.StatusUnauthorized, "error", "not authenticated")
		return
	}

	conn := tenant.ConnFromContext(r.Context())
	if conn == nil {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "no database connection")
		return
	}
	q := dbtenant.New(conn)

	user, err := q.GetUserByExternalID(r.Context(), identity.Subject)
	if err != nil {
		// For local admin or dev mode, return identity-based profile.
		httpserver.Respond(w, http.StatusOK, ProfileResponse{
			ID:          identity.Subject,
			Email:       identity.Email,
			DisplayName: identity.Name,
			Timezone:    "UTC",
			Role:        identity.Role,
			AuthMethod:  identity.Method,
		})
		return
	}

	httpserver.Respond(w, http.StatusOK, ProfileResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Timezone:    user.Timezone,
		Role:        user.Role,
		AuthMethod:  user.AuthMethod,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}

// UpdateProfileRequest is the body for PUT /api/v1/profile.
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Timezone    string `json:"timezone"`
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	userID := tenant.UserIDFromContext(r.Context())
	if !userID.Valid {
httpserver.RespondError(w, http.StatusForbidden, "error", "user not resolved")
		return
	}

	conn := tenant.ConnFromContext(r.Context())
	q := dbtenant.New(conn)

	u, err := q.UpdateUserProfile(r.Context(), dbtenant.UpdateUserProfileParams{
		ID:          uuid.UUID(userID.Bytes),
		DisplayName: req.DisplayName,
		Timezone:    req.Timezone,
	})
	if err != nil {
		slog.Error("updating profile", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to update profile")
		return
	}

	httpserver.Respond(w, http.StatusOK, ProfileResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Timezone:    u.Timezone,
		Role:        u.Role,
		AuthMethod:  u.AuthMethod,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
	})
}

// PersonalTokenResponse is returned by GET /api/v1/profile/tokens.
type PersonalTokenResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at"`
	ExpiresAt  *string `json:"expires_at"`
}

// PersonalTokenCreatedResponse is returned by POST /api/v1/profile/tokens.
type PersonalTokenCreatedResponse struct {
	PersonalTokenResponse
	Plaintext string `json:"plaintext"`
}

// CreateTokenRequest is the body for POST /api/v1/profile/tokens.
type CreateTokenRequest struct {
	Name      string  `json:"name"`
	ExpiresIn *string `json:"expires_in,omitempty"` // "30d", "90d", "1y", or empty for never
}

func (h *ProfileHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	t, ok := tenant.FromContext(r.Context())
	if !ok {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	userID := tenant.UserIDFromContext(r.Context())
	if !userID.Valid {
		httpserver.Respond(w, http.StatusOK, []PersonalTokenResponse{})
		return
	}

	tokens, err := h.globalQ.ListPersonalTokens(r.Context(), dbglobal.ListPersonalTokensParams{
		TenantID: t.ID,
		UserID:   userID,
	})
	if err != nil {
		slog.Error("listing personal tokens", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to list tokens")
		return
	}

	resp := make([]PersonalTokenResponse, 0, len(tokens))
	for _, tok := range tokens {
		item := PersonalTokenResponse{
			ID:        tok.ID.String(),
			Name:      tok.Description,
			Prefix:    tok.KeyPrefix,
			CreatedAt: tok.CreatedAt.Format(time.RFC3339),
		}
		if tok.LastUsedAt.Valid {
			s := tok.LastUsedAt.Time.Format(time.RFC3339)
			item.LastUsedAt = &s
		}
		if tok.ExpiresAt.Valid {
			s := tok.ExpiresAt.Time.Format(time.RFC3339)
			item.ExpiresAt = &s
		}
		resp = append(resp, item)
	}

	httpserver.Respond(w, http.StatusOK, resp)
}

func (h *ProfileHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	var req CreateTokenRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}

	if req.Name == "" {
httpserver.RespondError(w, http.StatusBadRequest, "error", "token name is required")
		return
	}

	t, ok := tenant.FromContext(r.Context())
	if !ok {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	identity := auth.FromContext(r.Context())
	userID := tenant.UserIDFromContext(r.Context())
	if !userID.Valid {
httpserver.RespondError(w, http.StatusForbidden, "error", "user not resolved")
		return
	}

	// Generate random token.
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to generate token")
		return
	}
	plaintext := "bwp_" + hex.EncodeToString(rawBytes)
	prefix := plaintext[:8]

	hash := sha256.Sum256([]byte(plaintext))
	keyHash := hex.EncodeToString(hash[:])

	// Parse expiration.
	var expiresAt pgtype.Timestamptz
	if req.ExpiresIn != nil && *req.ExpiresIn != "" {
		dur, err := parseExpiration(*req.ExpiresIn)
		if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		expiresAt = pgtype.Timestamptz{Time: time.Now().Add(dur), Valid: true}
	}

	tok, err := h.globalQ.CreatePersonalToken(r.Context(), dbglobal.CreatePersonalTokenParams{
		TenantID:    t.ID,
		KeyHash:     keyHash,
		KeyPrefix:   prefix,
		Description: req.Name,
		Role:        identity.Role,
		UserID:      userID,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		slog.Error("creating personal token", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to create token")
		return
	}

	item := PersonalTokenResponse{
		ID:        tok.ID.String(),
		Name:      tok.Description,
		Prefix:    tok.KeyPrefix,
		CreatedAt: tok.CreatedAt.Format(time.RFC3339),
	}
	if tok.ExpiresAt.Valid {
		s := tok.ExpiresAt.Time.Format(time.RFC3339)
		item.ExpiresAt = &s
	}

	httpserver.Respond(w, http.StatusCreated, PersonalTokenCreatedResponse{
		PersonalTokenResponse: item,
		Plaintext:             plaintext,
	})
}

func (h *ProfileHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	tokenID, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	t, ok := tenant.FromContext(r.Context())
	if !ok {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	userID := tenant.UserIDFromContext(r.Context())
	if !userID.Valid {
httpserver.RespondError(w, http.StatusForbidden, "error", "user not resolved")
		return
	}

	err = h.globalQ.DeletePersonalToken(r.Context(), dbglobal.DeletePersonalTokenParams{
		ID:       tokenID,
		TenantID: t.ID,
		UserID:   userID,
	})
	if err != nil {
		if db.IsNotFound(err) {
httpserver.RespondError(w, http.StatusNotFound, "error", "token not found")
			return
		}
		slog.Error("revoking personal token", "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to revoke token")
		return
	}

	httpserver.Respond(w, http.StatusNoContent, nil)
}

func parseExpiration(s string) (time.Duration, error) {
	switch s {
	case "30d":
		return 30 * 24 * time.Hour, nil
	case "90d":
		return 90 * 24 * time.Hour, nil
	case "1y":
		return 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid expiration: %s (use 30d, 90d, or 1y)", s)
	}
}
