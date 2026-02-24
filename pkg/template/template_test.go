package template

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

func TestToResponse(t *testing.T) {
	id := uuid.New()
	spaceID := uuid.New()
	createdBy := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	content := json.RawMessage(`{"type":"doc","content":[]}`)

	row := dbtenant.Template{
		ID:          id,
		Name:        "Test Template",
		Description: pgtype.Text{String: "A description", Valid: true},
		DocType:     "runbook",
		Content:     content,
		Icon:        pgtype.Text{String: "book-open", Valid: true},
		IsSystem:    true,
		IsGlobal:    true,
		SpaceID:     pgtype.UUID{Bytes: spaceID, Valid: true},
		CreatedBy:   pgtype.UUID{Bytes: createdBy, Valid: true},
		SortOrder:   5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := toResponse(row)

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Name != "Test Template" {
		t.Errorf("Name = %q, want %q", resp.Name, "Test Template")
	}
	if resp.Description == nil || *resp.Description != "A description" {
		t.Errorf("Description = %v, want %q", resp.Description, "A description")
	}
	if resp.DocType != "runbook" {
		t.Errorf("DocType = %q, want %q", resp.DocType, "runbook")
	}
	if resp.Icon == nil || *resp.Icon != "book-open" {
		t.Errorf("Icon = %v, want %q", resp.Icon, "book-open")
	}
	if !resp.IsSystem {
		t.Error("IsSystem = false, want true")
	}
	if !resp.IsGlobal {
		t.Error("IsGlobal = false, want true")
	}
	if resp.SpaceID == nil || *resp.SpaceID != spaceID {
		t.Errorf("SpaceID = %v, want %v", resp.SpaceID, spaceID)
	}
	if resp.CreatedBy == nil || *resp.CreatedBy != createdBy {
		t.Errorf("CreatedBy = %v, want %v", resp.CreatedBy, createdBy)
	}
	if resp.SortOrder != 5 {
		t.Errorf("SortOrder = %d, want %d", resp.SortOrder, 5)
	}
}

func TestToResponse_NullFields(t *testing.T) {
	row := dbtenant.Template{
		ID:          uuid.New(),
		Name:        "Minimal",
		DocType:     "document",
		Content:     json.RawMessage(`{}`),
		Description: pgtype.Text{Valid: false},
		Icon:        pgtype.Text{Valid: false},
		SpaceID:     pgtype.UUID{Valid: false},
		CreatedBy:   pgtype.UUID{Valid: false},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := toResponse(row)

	if resp.Description != nil {
		t.Errorf("Description = %v, want nil", resp.Description)
	}
	if resp.Icon != nil {
		t.Errorf("Icon = %v, want nil", resp.Icon)
	}
	if resp.SpaceID != nil {
		t.Errorf("SpaceID = %v, want nil", resp.SpaceID)
	}
	if resp.CreatedBy != nil {
		t.Errorf("CreatedBy = %v, want nil", resp.CreatedBy)
	}
}

func TestErrors(t *testing.T) {
	if ErrNotFound.Error() != "template not found" {
		t.Errorf("ErrNotFound = %q", ErrNotFound.Error())
	}
	if ErrSystemReadOnly.Error() != "system templates cannot be modified" {
		t.Errorf("ErrSystemReadOnly = %q", ErrSystemReadOnly.Error())
	}
}
