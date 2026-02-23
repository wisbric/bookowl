package collection

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

func (s *Service) Create(ctx context.Context, store *Store, spaceID uuid.UUID, req CreateRequest, userID pgtype.UUID) (Response, error) {
	if req.Name == "" {
		return Response{}, fmt.Errorf("name is required")
	}
	if req.Slug == "" {
		return Response{}, fmt.Errorf("slug is required")
	}

	row, err := store.Create(ctx, toCreateParams(spaceID, req, userID))
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

func (s *Service) List(ctx context.Context, store *Store, spaceID uuid.UUID) ([]Response, error) {
	rows, err := store.ListBySpace(ctx, spaceID)
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
		return Response{}, fmt.Errorf("name is required")
	}
	if req.Slug == "" {
		return Response{}, fmt.Errorf("slug is required")
	}

	var pos int32
	if req.Position != nil {
		pos = *req.Position
	}

	row, err := store.Update(ctx, dbtenant.UpdateCollectionParams{
		ID:       id,
		Name:     req.Name,
		Slug:     req.Slug,
		Icon:     db.NullText(req.Icon),
		Position: pos,
		ParentID: db.NullUUID(req.ParentID),
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Delete(ctx context.Context, store *Store, id uuid.UUID) error {
	return store.Delete(ctx, id)
}
