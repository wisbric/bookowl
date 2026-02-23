package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	"github.com/wisbric/bookowl/internal/httpserver"
)

type Identity struct {
	TenantSlug string
	ExternalID string     // OIDC subject claim
	Email      string     // from JWT or API key description
	Name       string     // display name
	APIKeyID   *uuid.UUID // non-nil for API key auth
	Role       string     // admin, editor, viewer
	Method     string     // "apikey", "oidc", "dev"
}

type ctxKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

func contextWithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

type Middleware struct {
	globalQ *dbglobal.Queries
	devMode bool
}

func NewMiddleware(globalDB dbglobal.DBTX, devMode bool) *Middleware {
	return &Middleware{
		globalQ: dbglobal.New(globalDB),
		devMode: devMode,
	}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try API key first.
		if key := r.Header.Get("X-API-Key"); key != "" {
			id, ok := m.authenticateAPIKey(r.Context(), key)
			if !ok {
				httpserver.RespondError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			next.ServeHTTP(w, r.WithContext(contextWithIdentity(r.Context(), id)))
			return
		}

		// Try Bearer token.
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			// TODO: Implement OIDC JWT validation.
			// For now fall through to dev mode if enabled.
			slog.Warn("OIDC JWT validation not implemented, falling through")
		}

		// Dev mode fallback.
		if m.devMode {
			if slug := r.Header.Get("X-Tenant-Slug"); slug != "" {
				next.ServeHTTP(w, r.WithContext(contextWithIdentity(r.Context(), Identity{
					TenantSlug: slug,
					ExternalID: "dev-user",
					Email:      "dev@bookowl.local",
					Name:       "Dev User",
					Role:       "admin",
					Method:     "dev",
				})))
				return
			}
		}

		httpserver.RespondError(w, http.StatusUnauthorized, "authentication required")
	})
}

func (m *Middleware) authenticateAPIKey(ctx context.Context, rawKey string) (Identity, bool) {
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	row, err := m.globalQ.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return Identity{}, false
	}

	keyID := row.ID
	return Identity{
		TenantSlug: row.TenantSlug,
		APIKeyID:   &keyID,
		Role:       row.Role,
		Method:     "apikey",
	}, true
}
