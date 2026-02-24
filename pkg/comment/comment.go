package comment

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

var (
	ErrNotFound          = errors.New("comment not found")
	ErrEditWindowExpired = errors.New("edit window has expired")
	ErrReplyToReply      = errors.New("cannot reply to a reply")
	ErrNotAuthor         = errors.New("only the author can perform this action")
	ErrNotTopLevel       = errors.New("only top-level comments can be resolved")
	ErrEmptyBody         = errors.New("comment body cannot be empty")
)

type AuthorInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Initials    string    `json:"initials"`
	AvatarURL   *string   `json:"avatar_url"`
}

type CommentResponse struct {
	ID           uuid.UUID         `json:"id"`
	DocumentID   uuid.UUID         `json:"document_id"`
	ParentID     *uuid.UUID        `json:"parent_id"`
	Author       AuthorInfo        `json:"author"`
	Body         string            `json:"body"`
	BodyRendered string            `json:"body_rendered"`
	IsResolved   bool              `json:"is_resolved"`
	IsDeleted    bool              `json:"is_deleted"`
	EditedAt     *time.Time        `json:"edited_at"`
	CreatedAt    time.Time         `json:"created_at"`
	Replies      []CommentResponse `json:"replies"`
}

type CreateRequest struct {
	Body string `json:"body"`
}

type UpdateRequest struct {
	Body string `json:"body"`
}

type CountResponse struct {
	Count int64 `json:"count"`
}

func toResponse(c dbtenant.Comment, author AuthorInfo, replies []CommentResponse) CommentResponse {
	resp := CommentResponse{
		ID:           c.ID,
		DocumentID:   c.DocumentID,
		ParentID:     db.UUIDPtr(c.ParentID),
		Author:       author,
		Body:         c.Body,
		BodyRendered: c.BodyRendered,
		IsResolved:   c.IsResolved,
		IsDeleted:    c.IsDeleted,
		CreatedAt:    c.CreatedAt,
		Replies:      replies,
	}
	if c.EditedAt.Valid {
		t := c.EditedAt.Time
		resp.EditedAt = &t
	}
	return resp
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
