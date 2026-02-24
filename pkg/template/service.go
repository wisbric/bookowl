package template

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Create(ctx context.Context, store *Store, req CreateRequest, userID pgtype.UUID) (Response, error) {
	if req.Name == "" {
		return Response{}, fmt.Errorf("template name is required")
	}
	if len(req.Content) == 0 {
		return Response{}, fmt.Errorf("template content is required")
	}

	docType := req.DocType
	if docType == "" {
		docType = "document"
	}

	row, err := store.Create(ctx, dbtenant.CreateTemplateParams{
		Name:        req.Name,
		Description: db.ValidText(req.Description),
		DocType:     docType,
		Content:     req.Content,
		Icon:        db.NullText(req.Icon),
		IsSystem:    false,
		IsGlobal:    req.IsGlobal,
		SpaceID:     db.NullUUID(req.SpaceID),
		CreatedBy:   userID,
		SortOrder:   0,
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Get(ctx context.Context, store *Store, id uuid.UUID) (Response, error) {
	row, err := store.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) List(ctx context.Context, store *Store, spaceID *uuid.UUID, docType string) ([]Response, error) {
	var rows []dbtenant.Template
	var err error

	if docType != "" && spaceID != nil {
		rows, err = store.ListByDocType(ctx, docType, *spaceID)
	} else {
		rows, err = store.List(ctx, spaceID)
	}
	if err != nil {
		return nil, err
	}

	result := make([]Response, len(rows))
	for i, r := range rows {
		result[i] = toResponse(r)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, store *Store, id uuid.UUID, req UpdateRequest) (Response, error) {
	if req.Name == "" {
		return Response{}, fmt.Errorf("template name is required")
	}

	// Check that the template exists and is not a system template.
	existing, err := store.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if existing.IsSystem {
		return Response{}, ErrSystemReadOnly
	}

	row, err := store.Update(ctx, dbtenant.UpdateTemplateParams{
		ID:          id,
		Name:        req.Name,
		Description: db.ValidText(req.Description),
		Content:     req.Content,
		Icon:        db.NullText(req.Icon),
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Delete(ctx context.Context, store *Store, id uuid.UUID) error {
	existing, err := store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return ErrSystemReadOnly
	}
	return store.Delete(ctx, id)
}

func (s *Service) SaveDocumentAsTemplate(ctx context.Context, store *Store, docID uuid.UUID, req SaveAsTemplateRequest, userID pgtype.UUID) (Response, error) {
	if req.Name == "" {
		return Response{}, fmt.Errorf("template name is required")
	}

	doc, err := store.GetDocumentByID(ctx, docID)
	if err != nil {
		return Response{}, err
	}

	row, err := store.Create(ctx, dbtenant.CreateTemplateParams{
		Name:        req.Name,
		Description: db.ValidText(req.Description),
		DocType:     doc.DocType,
		Content:     doc.Content,
		Icon:        doc.Icon,
		IsSystem:    false,
		IsGlobal:    req.IsGlobal,
		SpaceID:     db.NullUUID(req.SpaceID),
		CreatedBy:   userID,
		SortOrder:   0,
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

type SaveAsTemplateRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	IsGlobal    bool       `json:"is_global"`
	SpaceID     *uuid.UUID `json:"space_id,omitempty"`
}
