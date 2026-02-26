package notification

import (
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type AuthorInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Initials    string    `json:"initials"`
}

type NotificationResponse struct {
	ID         uuid.UUID   `json:"id"`
	Type       string      `json:"type"`
	DocumentID *uuid.UUID  `json:"document_id"`
	CommentID  *uuid.UUID  `json:"comment_id"`
	Actor      *AuthorInfo `json:"actor"`
	IsRead     bool        `json:"is_read"`
	CreatedAt  time.Time   `json:"created_at"`
}

type CountResponse struct {
	Count int64 `json:"count"`
}

type MarkReadRequest struct {
	IDs []uuid.UUID `json:"ids"`
	All bool        `json:"all"`
}

func toResponse(n dbtenant.Notification, actor *AuthorInfo) NotificationResponse {
	return NotificationResponse{
		ID:         n.ID,
		Type:       n.Type,
		DocumentID: db.UUIDPtr(n.DocumentID),
		CommentID:  db.UUIDPtr(n.CommentID),
		Actor:      actor,
		IsRead:     n.IsRead,
		CreatedAt:  n.CreatedAt,
	}
}

func makeAuthorInfo(u dbtenant.User) AuthorInfo {
	return AuthorInfo{
		ID:          u.ID,
		DisplayName: u.DisplayName,
		Initials:    initials(u.DisplayName),
	}
}

func initials(name string) string {
	parts := splitWords(name)
	if len(parts) >= 2 {
		return string(upper(parts[0][0])) + string(upper(parts[len(parts)-1][0]))
	}
	if len(name) >= 2 {
		return string(upper(name[0])) + string(upper(name[1]))
	}
	if len(name) == 1 {
		return string(upper(name[0]))
	}
	return "?"
}

func splitWords(s string) []string {
	var words []string
	word := ""
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if word != "" {
				words = append(words, word)
				word = ""
			}
		} else {
			word += string(c)
		}
	}
	if word != "" {
		words = append(words, word)
	}
	return words
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}
