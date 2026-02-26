package tenant

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"

	"github.com/wisbric/bookowl/internal/authadapter"
	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

var validSlug = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

type ResolveMiddleware struct {
	pool    *pgxpool.Pool
	globalQ *dbglobal.Queries
}

func NewResolveMiddleware(pool *pgxpool.Pool, globalDB dbglobal.DBTX) *ResolveMiddleware {
	return &ResolveMiddleware{
		pool:    pool,
		globalQ: dbglobal.New(globalDB),
	}
}

func (m *ResolveMiddleware) Resolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		identity := auth.FromContext(ctx)
		if identity == nil {
			httpserver.RespondError(w, http.StatusUnauthorized, "error", "no identity in context")
			return
		}

		slug := identity.TenantSlug
		if !validSlug.MatchString(slug) {
			httpserver.RespondError(w, http.StatusBadRequest, "error", "invalid tenant slug")
			return
		}

		t, err := m.globalQ.GetTenantBySlug(ctx, slug)
		if err != nil {
			slog.Error("looking up tenant", "slug", slug, "error", err)
			httpserver.RespondError(w, http.StatusUnauthorized, "error", "tenant not found")
			return
		}

		conn, err := m.pool.Acquire(ctx)
		if err != nil {
			slog.Error("acquiring connection", "error", err)
			httpserver.RespondError(w, http.StatusServiceUnavailable, "error", "database unavailable")
			return
		}
		defer conn.Release()

		schema := fmt.Sprintf("tenant_%s", slug)
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
			slog.Error("setting search_path", "schema", schema, "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant setup failed")
			return
		}

		tenant := Tenant{
			ID:        t.ID,
			Slug:      t.Slug,
			Name:      t.Name,
			Config:    t.Config,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}

		ctx = ContextWithTenant(ctx, tenant)
		ctx = ContextWithConn(ctx, conn)

		// If OIDC user, resolve or auto-provision user in tenant schema.
		if identity.Subject != "" && identity.Method == "oidc" {
			// Determine role from group mapping if OIDC groups are configured.
			provisionRole := "editor"
			if len(identity.Groups) > 0 {
				groupCfg := loadOIDCGroupConfig(tenant.Config)
				if groupCfg != nil {
					mapped := authadapter.MapGroupsToRole(identity.Groups, *groupCfg)
					if mapped == "" {
						httpserver.RespondError(w, http.StatusForbidden, "error", "not in any allowed group")
						return
					}
					provisionRole = mapped
				}
			}

			q := dbtenant.New(conn)
			user, err := q.GetUserByExternalID(ctx, identity.Subject)
			if err != nil {
				// User not found — auto-provision via upsert.
				slog.Info("auto-provisioning OIDC user", "external_id", identity.Subject, "email", identity.Email, "role", provisionRole)
				user, err = q.UpsertUser(ctx, dbtenant.UpsertUserParams{
					ExternalID:  identity.Subject,
					Email:       identity.Email,
					DisplayName: identity.Name,
					Role:        provisionRole,
				})
				if err != nil {
					slog.Error("auto-provisioning user failed", "error", err)
					httpserver.RespondError(w, http.StatusInternalServerError, "error", "user provisioning failed")
					return
				}
			} else if len(identity.Groups) > 0 {
				// Existing user: update role from groups if group config is set.
				groupCfg := loadOIDCGroupConfig(tenant.Config)
				if groupCfg != nil {
					mapped := authadapter.MapGroupsToRole(identity.Groups, *groupCfg)
					if mapped == "" {
						httpserver.RespondError(w, http.StatusForbidden, "error", "not in any allowed group")
						return
					}
					provisionRole = mapped
				}
			}

			ctx = ContextWithUserID(ctx, user.ID)
			// Use group-mapped role if available, otherwise sync from DB.
			if len(identity.Groups) > 0 {
				identity.Role = provisionRole
			} else {
				identity.Role = user.Role
			}
			ctx = auth.NewContext(ctx, identity)
		} else if identity.Subject != "" {
			// Dev mode: resolve existing user if present.
			q := dbtenant.New(conn)
			user, err := q.GetUserByExternalID(ctx, identity.Subject)
			if err == nil {
				ctx = ContextWithUserID(ctx, user.ID)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// oidcSection is used to extract the OIDC config from tenant config JSONB.
type oidcSection struct {
	AllowedGroups []string `json:"allowed_groups"`
	AdminGroups   []string `json:"admin_groups"`
	EditorGroups  []string `json:"editor_groups"`
	Enabled       bool     `json:"enabled"`
}

// loadOIDCGroupConfig extracts the group mapping config from a tenant's config JSONB.
// Returns nil if OIDC is not configured or has no group settings.
func loadOIDCGroupConfig(configJSON json.RawMessage) *authadapter.OIDCGroupConfig {
	var full map[string]json.RawMessage
	if len(configJSON) == 0 {
		return nil
	}
	if err := json.Unmarshal(configJSON, &full); err != nil {
		return nil
	}
	raw, ok := full["oidc"]
	if !ok {
		return nil
	}
	var section oidcSection
	if err := json.Unmarshal(raw, &section); err != nil {
		return nil
	}
	if !section.Enabled {
		return nil
	}
	// Only return config if at least one group field is set.
	if len(section.AdminGroups) == 0 && len(section.EditorGroups) == 0 && len(section.AllowedGroups) == 0 {
		return nil
	}
	return &authadapter.OIDCGroupConfig{
		AllowedGroups: section.AllowedGroups,
		AdminGroups:   section.AdminGroups,
		EditorGroups:  section.EditorGroups,
	}
}
