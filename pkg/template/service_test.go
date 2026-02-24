package template

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestService_Create_Validation(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	userID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	tests := []struct {
		name    string
		req     CreateRequest
		wantErr string
	}{
		{
			name:    "empty name",
			req:     CreateRequest{Content: json.RawMessage(`{}`)},
			wantErr: "template name is required",
		},
		{
			name:    "empty content",
			req:     CreateRequest{Name: "Test"},
			wantErr: "template content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store is nil — validation errors fire before store access.
			_, err := svc.Create(ctx, nil, tt.req, userID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestService_Update_NameRequired(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Store is nil — validation fires before store access.
	_, err := svc.Update(ctx, nil, uuid.New(), UpdateRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if err.Error() != "template name is required" {
		t.Errorf("error = %q", err.Error())
	}
}

func TestService_SaveDocumentAsTemplate_NameRequired(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	userID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	// Store is nil — validation fires before store access.
	_, err := svc.SaveDocumentAsTemplate(ctx, nil, uuid.New(), SaveAsTemplateRequest{Name: ""}, userID)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if err.Error() != "template name is required" {
		t.Errorf("error = %q", err.Error())
	}
}
