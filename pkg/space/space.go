package space

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

var (
	ErrNotFound    = errors.New("space not found")
	ErrSlugConflict = errors.New("space slug already exists")
)

type CreateRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	IsPrivate   *bool   `json:"is_private,omitempty"`
}

type UpdateRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	IsPrivate   *bool   `json:"is_private,omitempty"`
}

type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}

type Response struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description,omitempty"`
	Icon        *string    `json:"icon,omitempty"`
	IsPrivate   bool       `json:"is_private"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type MemberResponse struct {
	ID          uuid.UUID `json:"id"`
	SpaceID     uuid.UUID `json:"space_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type TreeNode struct {
	CollectionID   uuid.UUID   `json:"collection_id"`
	CollectionName string      `json:"collection_name"`
	CollectionSlug string      `json:"collection_slug"`
	ParentID       *uuid.UUID  `json:"parent_id,omitempty"`
	Position       int32       `json:"position"`
	Icon           *string     `json:"icon,omitempty"`
	Documents      []TreeDoc   `json:"documents"`
}

type TreeDoc struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Slug     string    `json:"slug"`
	DocType  string    `json:"doc_type"`
	Status   string    `json:"status"`
	Position int32     `json:"position"`
	Icon     *string   `json:"icon,omitempty"`
}

func toResponse(s dbtenant.Space) Response {
	return Response{
		ID:          s.ID,
		Name:        s.Name,
		Slug:        s.Slug,
		Description: db.TextPtr(s.Description),
		Icon:        db.TextPtr(s.Icon),
		IsPrivate:   db.BoolVal(s.IsPrivate),
		CreatedBy:   db.UUIDPtr(s.CreatedBy),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func toMemberResponse(m dbtenant.ListSpaceMembersRow) MemberResponse {
	return MemberResponse{
		ID:          m.ID,
		SpaceID:     m.SpaceID,
		UserID:      m.UserID,
		Role:        m.Role,
		Email:       m.UserEmail,
		DisplayName: m.UserDisplayName,
		CreatedAt:   m.CreatedAt,
	}
}

func buildTree(rows []dbtenant.GetSpaceTreeRow) []TreeNode {
	nodeMap := make(map[uuid.UUID]*TreeNode)
	var order []uuid.UUID

	for _, r := range rows {
		node, exists := nodeMap[r.CollectionID]
		if !exists {
			node = &TreeNode{
				CollectionID:   r.CollectionID,
				CollectionName: r.CollectionName,
				CollectionSlug: r.CollectionSlug,
				ParentID:       db.UUIDPtr(r.ParentID),
				Position:       r.CollectionPosition,
				Icon:           db.TextPtr(r.CollectionIcon),
				Documents:      []TreeDoc{},
			}
			nodeMap[r.CollectionID] = node
			order = append(order, r.CollectionID)
		}

		if r.DocumentID.Valid {
			docID := uuid.UUID(r.DocumentID.Bytes)
			node.Documents = append(node.Documents, TreeDoc{
				ID:       docID,
				Title:    r.DocumentTitle.String,
				Slug:     r.DocumentSlug.String,
				DocType:  r.DocType.String,
				Status:   r.Status.String,
				Position: r.DocumentPosition.Int32,
				Icon:     db.TextPtr(r.DocumentIcon),
			})
		}
	}

	result := make([]TreeNode, 0, len(order))
	for _, id := range order {
		result = append(result, *nodeMap[id])
	}
	return result
}

func toCreateParams(req CreateRequest, userID pgtype.UUID) dbtenant.CreateSpaceParams {
	return dbtenant.CreateSpaceParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: db.NullText(req.Description),
		Icon:        db.NullText(req.Icon),
		IsPrivate:   db.NullBool(req.IsPrivate),
		CreatedBy:   userID,
	}
}
