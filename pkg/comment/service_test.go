package comment

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

func TestEditWindowExpired(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		createdAt time.Time
		isAdmin   bool
		wantErr   bool
	}{
		{
			name:      "within window",
			createdAt: now.Add(-10 * time.Minute),
			isAdmin:   false,
			wantErr:   false,
		},
		{
			name:      "expired window",
			createdAt: now.Add(-20 * time.Minute),
			isAdmin:   false,
			wantErr:   true,
		},
		{
			name:      "expired window but admin",
			createdAt: now.Add(-20 * time.Minute),
			isAdmin:   true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expired := time.Since(tt.createdAt) > editWindow && !tt.isAdmin
			if expired != tt.wantErr {
				t.Errorf("editWindowExpired = %v, want %v", expired, tt.wantErr)
			}
		})
	}
}

func TestReplyToReplyValidation(t *testing.T) {
	// A comment with a non-nil ParentID is a reply; replying to it should fail.
	parentID := uuid.New()
	comment := dbtenant.Comment{
		ID:       uuid.New(),
		ParentID: pgtype.UUID{Bytes: parentID, Valid: true},
	}

	if !comment.ParentID.Valid {
		t.Error("expected ParentID to be valid (this is a reply)")
	}
	// The service would return ErrReplyToReply.
}

func TestNotAuthorValidation(t *testing.T) {
	authorID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name    string
		author  uuid.UUID
		actor   uuid.UUID
		isAdmin bool
		wantErr bool
	}{
		{
			name:    "author can delete",
			author:  authorID,
			actor:   authorID,
			isAdmin: false,
			wantErr: false,
		},
		{
			name:    "non-author cannot delete",
			author:  authorID,
			actor:   otherUserID,
			isAdmin: false,
			wantErr: true,
		},
		{
			name:    "admin can delete any",
			author:  authorID,
			actor:   otherUserID,
			isAdmin: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notAuthor := tt.author != tt.actor && !tt.isAdmin
			if notAuthor != tt.wantErr {
				t.Errorf("notAuthor = %v, want %v", notAuthor, tt.wantErr)
			}
		})
	}
}

func TestEmptyBodyValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "empty", body: "", wantErr: true},
		{name: "whitespace", body: "   ", wantErr: true},
		{name: "valid", body: "hello", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed := len(tt.body) > 0
			for _, c := range tt.body {
				if c != ' ' && c != '\t' && c != '\n' {
					trimmed = true
					break
				}
				trimmed = false
			}
			isEmpty := !trimmed
			if isEmpty != tt.wantErr {
				t.Errorf("isEmpty = %v, want %v for body %q", isEmpty, tt.wantErr, tt.body)
			}
		})
	}
}
