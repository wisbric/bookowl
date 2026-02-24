package template

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

var (
	ErrNotFound       = errors.New("template not found")
	ErrSystemReadOnly = errors.New("system templates cannot be modified")
)

type CreateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	DocType     string          `json:"doc_type"`
	Content     json.RawMessage `json:"content"`
	Icon        *string         `json:"icon,omitempty"`
	IsGlobal    bool            `json:"is_global"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
}

type UpdateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Content     json.RawMessage `json:"content"`
	Icon        *string         `json:"icon,omitempty"`
	SortOrder   int32           `json:"sort_order"`
}

type Response struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	DocType     string          `json:"doc_type"`
	Content     json.RawMessage `json:"content"`
	Icon        *string         `json:"icon,omitempty"`
	IsSystem    bool            `json:"is_system"`
	IsGlobal    bool            `json:"is_global"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty"`
	SortOrder   int32           `json:"sort_order"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func toResponse(t dbtenant.Template) Response {
	return Response{
		ID:          t.ID,
		Name:        t.Name,
		Description: db.TextPtr(t.Description),
		DocType:     t.DocType,
		Content:     t.Content,
		Icon:        db.TextPtr(t.Icon),
		IsSystem:    t.IsSystem,
		IsGlobal:    t.IsGlobal,
		SpaceID:     db.UUIDPtr(t.SpaceID),
		CreatedBy:   db.UUIDPtr(t.CreatedBy),
		SortOrder:   t.SortOrder,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
