package document

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

var (
	ErrNotFound    = errors.New("document not found")
	ErrSlugConflict = errors.New("document slug already exists in this space")
)

type CreateRequest struct {
	SpaceID            uuid.UUID        `json:"space_id"`
	CollectionID       *uuid.UUID       `json:"collection_id,omitempty"`
	Title              string           `json:"title"`
	Slug               string           `json:"slug"`
	Content            json.RawMessage  `json:"content"`
	DocType            string           `json:"doc_type,omitempty"`
	Status             string           `json:"status,omitempty"`
	Tags               []string         `json:"tags,omitempty"`
	Icon               *string          `json:"icon,omitempty"`
	Position           int32            `json:"position,omitempty"`
	NightOwlIncidentID *string          `json:"nightowl_incident_id,omitempty"`
	NightOwlAlertID    *string          `json:"nightowl_alert_id,omitempty"`
	TemplateID         *uuid.UUID       `json:"template_id,omitempty"`
}

type UpdateRequest struct {
	CollectionID       *uuid.UUID       `json:"collection_id,omitempty"`
	Title              string           `json:"title"`
	Slug               string           `json:"slug"`
	Content            json.RawMessage  `json:"content"`
	DocType            string           `json:"doc_type,omitempty"`
	Status             string           `json:"status,omitempty"`
	Tags               []string         `json:"tags,omitempty"`
	Icon               *string          `json:"icon,omitempty"`
	Position           int32            `json:"position,omitempty"`
	NightOwlIncidentID *string          `json:"nightowl_incident_id,omitempty"`
	NightOwlAlertID    *string          `json:"nightowl_alert_id,omitempty"`
	ChangeSummary      *string          `json:"change_summary,omitempty"`
}

type Response struct {
	ID                 uuid.UUID       `json:"id"`
	SpaceID            uuid.UUID       `json:"space_id"`
	CollectionID       *uuid.UUID      `json:"collection_id,omitempty"`
	Title              string          `json:"title"`
	Slug               string          `json:"slug"`
	Content            json.RawMessage `json:"content"`
	DocType            string          `json:"doc_type"`
	Status             string          `json:"status"`
	Tags               []string        `json:"tags"`
	Icon               *string         `json:"icon,omitempty"`
	Position           int32           `json:"position"`
	WordCount          int32           `json:"word_count"`
	Version            int32           `json:"version"`
	NightOwlIncidentID *string         `json:"nightowl_incident_id,omitempty"`
	NightOwlAlertID    *string         `json:"nightowl_alert_id,omitempty"`
	CreatedBy          *uuid.UUID      `json:"created_by,omitempty"`
	UpdatedBy          *uuid.UUID      `json:"updated_by,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type VersionResponse struct {
	ID            uuid.UUID       `json:"id"`
	DocumentID    uuid.UUID       `json:"document_id"`
	Version       int32           `json:"version"`
	Title         string          `json:"title"`
	Content       json.RawMessage `json:"content,omitempty"`
	DocType       string          `json:"doc_type"`
	Tags          []string        `json:"tags"`
	ChangedBy     *uuid.UUID      `json:"changed_by,omitempty"`
	ChangeSummary *string         `json:"change_summary,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type SearchResult struct {
	ID               uuid.UUID  `json:"id"`
	Title            string     `json:"title"`
	DocType          string     `json:"doc_type"`
	Slug             string     `json:"slug"`
	SpaceID          uuid.UUID  `json:"space_id"`
	CollectionID     *uuid.UUID `json:"collection_id,omitempty"`
	SpaceName        string     `json:"space_name"`
	SpaceSlug        string     `json:"space_slug"`
	Rank             float32    `json:"rank"`
	TitleHighlight   string     `json:"title_highlight"`
	ContentHighlight string     `json:"content_highlight"`
}

func toResponse(d dbtenant.Document) Response {
	return Response{
		ID:                 d.ID,
		SpaceID:            d.SpaceID,
		CollectionID:       db.UUIDPtr(d.CollectionID),
		Title:              d.Title,
		Slug:               d.Slug,
		Content:            d.Content,
		DocType:            d.DocType,
		Status:             d.Status,
		Tags:               d.Tags,
		Icon:               db.TextPtr(d.Icon),
		Position:           d.Position,
		WordCount:          d.WordCount,
		Version:            d.Version,
		NightOwlIncidentID: db.TextPtr(d.NightowlIncidentID),
		NightOwlAlertID:    db.TextPtr(d.NightowlAlertID),
		CreatedBy:          db.UUIDPtr(d.CreatedBy),
		UpdatedBy:          db.UUIDPtr(d.UpdatedBy),
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

func toVersionResponse(v dbtenant.DocumentVersion) VersionResponse {
	return VersionResponse{
		ID:            v.ID,
		DocumentID:    v.DocumentID,
		Version:       v.Version,
		Title:         v.Title,
		Content:       v.Content,
		DocType:       v.DocType,
		Tags:          v.Tags,
		ChangedBy:     db.UUIDPtr(v.ChangedBy),
		ChangeSummary: db.TextPtr(v.ChangeSummary),
		CreatedAt:     v.CreatedAt,
	}
}

func toVersionListResponse(v dbtenant.ListDocumentVersionsRow) VersionResponse {
	return VersionResponse{
		ID:            v.ID,
		DocumentID:    v.DocumentID,
		Version:       v.Version,
		Title:         v.Title,
		DocType:       v.DocType,
		Tags:          v.Tags,
		ChangedBy:     db.UUIDPtr(v.ChangedBy),
		ChangeSummary: db.TextPtr(v.ChangeSummary),
		CreatedAt:     v.CreatedAt,
	}
}

// ExtractText walks a Tiptap/ProseMirror JSON document tree and extracts all text content.
func ExtractText(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}

	var node struct {
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(content, &node); err != nil {
		return ""
	}

	// Skip diagram blocks — no meaningful text to extract from XML/SVG.
	if node.Type == "diagramBlock" {
		return ""
	}

	var b strings.Builder
	if node.Text != "" {
		b.WriteString(node.Text)
	}

	if len(node.Content) > 0 {
		var children []json.RawMessage
		if err := json.Unmarshal(node.Content, &children); err == nil {
			for _, child := range children {
				text := ExtractText(child)
				if text != "" {
					if b.Len() > 0 {
						b.WriteByte(' ')
					}
					b.WriteString(text)
				}
			}
		}
	}

	return b.String()
}

// CountWords counts words in a string by splitting on whitespace.
func CountWords(s string) int32 {
	if s == "" {
		return 0
	}
	return int32(len(strings.Fields(s)))
}
