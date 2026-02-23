package tenant

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/bookowl/internal/auth"
	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/internal/httpserver"
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

		identity, ok := auth.IdentityFromContext(ctx)
		if !ok {
			httpserver.RespondError(w, http.StatusUnauthorized, "no identity in context")
			return
		}

		slug := identity.TenantSlug
		if !validSlug.MatchString(slug) {
			httpserver.RespondError(w, http.StatusBadRequest, "invalid tenant slug")
			return
		}

		t, err := m.globalQ.GetTenantBySlug(ctx, slug)
		if err != nil {
			slog.Error("looking up tenant", "slug", slug, "error", err)
			httpserver.RespondError(w, http.StatusUnauthorized, "tenant not found")
			return
		}

		conn, err := m.pool.Acquire(ctx)
		if err != nil {
			slog.Error("acquiring connection", "error", err)
			httpserver.RespondError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		defer conn.Release()

		schema := fmt.Sprintf("tenant_%s", slug)
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
			slog.Error("setting search_path", "schema", schema, "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "tenant setup failed")
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

		// If OIDC user, resolve user ID in tenant schema.
		if identity.ExternalID != "" {
			q := dbtenant.New(conn)
			user, err := q.GetUserByExternalID(ctx, identity.ExternalID)
			if err == nil {
				ctx = ContextWithUserID(ctx, user.ID)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
