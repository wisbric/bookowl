package authadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/authadapter"

	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

// Adapter implements auth.Storage for BookOwl. It embeds the shared
// BaseAdapter for cross-tenant scan methods and adds sqlc-specific operations.
type Adapter struct {
	authadapter.BaseAdapter
	pool *pgxpool.Pool
}

// New creates a new auth storage adapter.
func New(pool *pgxpool.Pool) *Adapter {
	a := &Adapter{pool: pool}
	a.BaseAdapter = authadapter.BaseAdapter{Pool: pool, TQ: a}
	return a
}

// --- sqlc-based methods (service-specific) ---

func (a *Adapter) GetAPIKeyByHash(ctx context.Context, hash string) (*auth.APIKeyResult, error) {
	q := dbglobal.New(a.pool)
	key, err := q.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	var expiresAt *time.Time
	if key.ExpiresAt.Valid {
		expiresAt = &key.ExpiresAt.Time
	}
	return &auth.APIKeyResult{
		APIKeyID:  key.ID,
		TenantID:  key.TenantID,
		KeyPrefix: key.KeyPrefix,
		Role:      key.Role,
		ExpiresAt: expiresAt,
	}, nil
}

func (a *Adapter) UpdateAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error {
	q := dbglobal.New(a.pool)
	return q.UpdateAPIKeyLastUsed(ctx, keyID)
}

func (a *Adapter) GetTenant(ctx context.Context, tenantID uuid.UUID) (*auth.TenantResult, error) {
	q := dbglobal.New(a.pool)
	t, err := q.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &auth.TenantResult{ID: t.ID, Slug: t.Slug}, nil
}

func (a *Adapter) GetTenantBySlug(ctx context.Context, slug string) (*auth.TenantResult, error) {
	q := dbglobal.New(a.pool)
	t, err := q.GetTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &auth.TenantResult{ID: t.ID, Slug: t.Slug}, nil
}

func (a *Adapter) ListTenants(ctx context.Context) ([]auth.TenantResult, error) {
	q := dbglobal.New(a.pool)
	tenants, err := q.ListTenants(ctx)
	if err != nil {
		return nil, err
	}
	var res []auth.TenantResult
	for _, t := range tenants {
		res = append(res, auth.TenantResult{ID: t.ID, Slug: t.Slug})
	}
	return res, nil
}

// FindLocalAdmin overrides BaseAdapter.FindLocalAdmin because BookOwl's
// local_admins table uses tenant_slug (text) instead of tenant_id (uuid).
func (a *Adapter) FindLocalAdmin(ctx context.Context, username, tenantSlug string) (*auth.LocalAdminRow, string, error) {
	if tenantSlug != "" {
		var admin auth.LocalAdminRow
		err := a.pool.QueryRow(ctx,
			"SELECT la.id, t.id, la.username, la.password_hash, la.must_change FROM public.local_admins la JOIN public.tenants t ON t.slug = la.tenant_slug WHERE la.tenant_slug = $1 AND la.username = $2",
			tenantSlug, username,
		).Scan(&admin.ID, &admin.TenantID, &admin.Username, &admin.PasswordHash, &admin.MustChange)
		if err != nil {
			return nil, "", fmt.Errorf("local admin not found for tenant %s: %w", tenantSlug, err)
		}
		return &admin, tenantSlug, nil
	}

	// Search across all tenants.
	rows, err := a.pool.Query(ctx,
		"SELECT la.id, t.id, la.username, la.password_hash, la.must_change, la.tenant_slug FROM public.local_admins la JOIN public.tenants t ON t.slug = la.tenant_slug WHERE la.username = $1",
		username,
	)
	if err != nil {
		return nil, "", fmt.Errorf("finding local admin: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var admin auth.LocalAdminRow
		var slug string
		if err := rows.Scan(&admin.ID, &admin.TenantID, &admin.Username, &admin.PasswordHash, &admin.MustChange, &slug); err != nil {
			return nil, "", fmt.Errorf("scanning local admin: %w", err)
		}
		return &admin, slug, nil
	}

	return nil, "", fmt.Errorf("local admin not found")
}

func (a *Adapter) FindOrCreateOIDCUser(ctx context.Context, tenantSlug, subject, email, displayName, role string) (*auth.UserRow, string, error) {
	t, err := a.GetTenantBySlug(ctx, tenantSlug)
	if err != nil {
		return nil, "", fmt.Errorf("looking up tenant %s: %w", tenantSlug, err)
	}

	conn, err := a.pool.Acquire(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	schema := fmt.Sprintf("tenant_%s", t.Slug)
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		return nil, "", fmt.Errorf("setting search_path: %w", err)
	}

	tq := dbtenant.New(conn)
	user, err := tq.GetUserByExternalID(ctx, subject)
	if err != nil {
		user, err = tq.UpsertUser(ctx, dbtenant.UpsertUserParams{
			ExternalID:  subject,
			Email:       email,
			DisplayName: displayName,
			Role:        role,
		})
		if err != nil {
			return nil, "", fmt.Errorf("creating user: %w", err)
		}
	}

	return &auth.UserRow{
		ID:          user.ID,
		ExternalID:  &user.ExternalID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		IsActive:    user.IsActive,
	}, t.ID.String(), nil
}
