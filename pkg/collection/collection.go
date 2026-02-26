package collection

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

var (
	ErrNotFound     = errors.New("collection not found")
	ErrSlugConflict = errors.New("collection slug already exists in this space")
)

type CreateRequest struct {
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Icon     *string    `json:"icon,omitempty"`
	Position *int32     `json:"position,omitempty"`
}

type UpdateRequest struct {
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Icon     *string    `json:"icon,omitempty"`
	Position *int32     `json:"position,omitempty"`
}

type Response struct {
	ID        uuid.UUID  `json:"id"`
	SpaceID   uuid.UUID  `json:"space_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Icon      *string    `json:"icon,omitempty"`
	Position  int32      `json:"position"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func toResponse(c dbtenant.Collection) Response {
	return Response{
		ID:        c.ID,
		SpaceID:   c.SpaceID,
		ParentID:  db.UUIDPtr(c.ParentID),
		Name:      c.Name,
		Slug:      c.Slug,
		Icon:      db.TextPtr(c.Icon),
		Position:  c.Position,
		CreatedBy: db.UUIDPtr(c.CreatedBy),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toCreateParams(spaceID uuid.UUID, req CreateRequest, userID pgtype.UUID) dbtenant.CreateCollectionParams {
	var pos int32
	if req.Position != nil {
		pos = *req.Position
	}
	return dbtenant.CreateCollectionParams{
		SpaceID:   spaceID,
		ParentID:  db.NullUUID(req.ParentID),
		Name:      req.Name,
		Slug:      req.Slug,
		Icon:      db.NullText(req.Icon),
		Position:  pos,
		CreatedBy: userID,
	}
}
