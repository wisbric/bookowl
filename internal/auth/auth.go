package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	"github.com/wisbric/bookowl/internal/httpserver"
	"github.com/wisbric/bookowl/internal/session"
)

type Identity struct {
	TenantSlug string
	ExternalID string     // OIDC subject claim
	Email      string     // from JWT or API key description
	Name       string     // display name
	APIKeyID   *uuid.UUID // non-nil for API key auth
	Role       string     // admin, editor, viewer
	Method     string     // "apikey", "oidc", "dev", "local"
	Groups     []string   // OIDC groups claim (for group→role mapping)
	UserID     string     // set from session JWT
}

type ctxKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

func ContextWithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

type Middleware struct {
	globalQ      *dbglobal.Queries
	devMode      bool
	oidcVerifier *oidc.IDTokenVerifier
	sess         *session.Manager
}

func NewMiddleware(globalDB dbglobal.DBTX, devMode bool, oidcVerifier *oidc.IDTokenVerifier, sess *session.Manager) *Middleware {
	return &Middleware{
		globalQ:      dbglobal.New(globalDB),
		devMode:      devMode,
		oidcVerifier: oidcVerifier,
		sess:         sess,
	}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check bw_session cookie (local admin or OIDC session).
		if m.sess != nil {
			if cookie, err := r.Cookie(session.CookieName); err == nil {
				claims, err := m.sess.Validate(cookie.Value)
				if err == nil {
					// Silent refresh: issue new cookie when <2h remaining.
					if m.sess.ShouldRefresh(claims) {
						_ = m.sess.Issue(w, claims.Tenant, claims.Subject, claims.Role, claims.AuthMethod)
					}
					id := Identity{
						TenantSlug: claims.Tenant,
						ExternalID: claims.Subject,
						Role:       claims.Role,
						Method:     claims.AuthMethod,
						UserID:     claims.Subject,
					}
					next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), id)))
					return
				}
				// Invalid cookie — clear it and fall through.
				session.Clear(w)
			}
		}

		// 2. Check X-API-Key header.
		if key := r.Header.Get("X-API-Key"); key != "" {
			id, ok := m.authenticateAPIKey(r.Context(), key)
			if !ok {
				httpserver.RespondError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), id)))
			return
		}

		// 3. Try Bearer token (OIDC).
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			rawToken := strings.TrimPrefix(authHeader, "Bearer ")

			if m.oidcVerifier != nil {
				id, err := m.authenticateOIDC(r.Context(), r, rawToken)
				if err != nil {
					slog.Warn("OIDC authentication failed", "error", err)
					httpserver.RespondError(w, http.StatusUnauthorized, "invalid token")
					return
				}
				next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), id)))
				return
			}

			slog.Warn("Bearer token provided but OIDC not configured, falling through to dev mode")
		}

		// 4. Dev mode fallback.
		if m.devMode {
			if slug := r.Header.Get("X-Tenant-Slug"); slug != "" {
				next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), Identity{
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

type oidcClaims struct {
	Sub         string `json:"sub"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	TenantSlug  string `json:"tenant_slug"`
}

func (m *Middleware) authenticateOIDC(ctx context.Context, r *http.Request, rawToken string) (Identity, error) {
	idToken, err := m.oidcVerifier.Verify(ctx, rawToken)
	if err != nil {
		return Identity{}, err
	}

	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, err
	}

	// Extract groups claim for group→role mapping.
	var rawClaims map[string]interface{}
	_ = idToken.Claims(&rawClaims)
	groups := ExtractGroups(rawClaims)

	// Resolve tenant slug: prefer JWT claim, fall back to header.
	tenantSlug := claims.TenantSlug
	if tenantSlug == "" {
		tenantSlug = r.Header.Get("X-Tenant-Slug")
	}

	name := claims.Name
	if name == "" {
		name = claims.Email
	}

	return Identity{
		TenantSlug: tenantSlug,
		ExternalID: claims.Sub,
		Email:      claims.Email,
		Name:       name,
		Role:       "editor", // default; tenant middleware syncs from DB
		Method:     "oidc",
		Groups:     groups,
	}, nil
}

// ExtractGroups extracts the groups claim from JWT claims.
func ExtractGroups(claims map[string]interface{}) []string {
	raw, ok := claims["groups"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		groups := make([]string, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		return groups
	case []string:
		return v
	}
	return nil
}

// OIDCGroupConfig holds the group mapping config from tenant OIDC settings.
type OIDCGroupConfig struct {
	AllowedGroups []string
	AdminGroups   []string
	EditorGroups  []string
}

// MapGroupsToRole maps JWT group membership to a BookOwl role.
// Returns empty string if user should be denied access.
func MapGroupsToRole(groups []string, cfg OIDCGroupConfig) string {
	for _, g := range groups {
		for _, adminGroup := range cfg.AdminGroups {
			if g == adminGroup {
				return "admin"
			}
		}
	}
	for _, g := range groups {
		for _, editorGroup := range cfg.EditorGroups {
			if g == editorGroup {
				return "editor"
			}
		}
	}
	if len(cfg.AllowedGroups) > 0 {
		for _, g := range groups {
			for _, allowed := range cfg.AllowedGroups {
				if g == allowed {
					return "viewer"
				}
			}
		}
		return "" // not in any allowed group → deny
	}
	return "viewer" // no group restrictions → everyone is viewer
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
