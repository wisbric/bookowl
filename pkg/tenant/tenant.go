package tenant

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type Tenant struct {
	ID        uuid.UUID       `json:"id"`
	Slug      string          `json:"slug"`
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type tenantKey struct{}
type connKey struct{}
type userIDKey struct{}

func FromContext(ctx context.Context) (Tenant, bool) {
	t, ok := ctx.Value(tenantKey{}).(Tenant)
	return t, ok
}

func ConnFromContext(ctx context.Context) dbtenant.DBTX {
	conn, _ := ctx.Value(connKey{}).(dbtenant.DBTX)
	return conn
}

func UserIDFromContext(ctx context.Context) pgtype.UUID {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func ContextWithTenant(ctx context.Context, t Tenant) context.Context {
	return context.WithValue(ctx, tenantKey{}, t)
}

func ContextWithConn(ctx context.Context, conn dbtenant.DBTX) context.Context {
	return context.WithValue(ctx, connKey{}, conn)
}

func ContextWithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}
