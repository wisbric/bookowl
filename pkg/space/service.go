package space

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
		return Response{}, fmt.Errorf("name is required")
	}
	if req.Slug == "" {
		return Response{}, fmt.Errorf("slug is required")
	}

	row, err := store.Create(ctx, toCreateParams(req, userID))
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

func (s *Service) List(ctx context.Context, store *Store) ([]Response, error) {
	rows, err := store.List(ctx)
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

	row, err := store.Update(ctx, dbtenant.UpdateSpaceParams{
		ID:          id,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: db.NullText(req.Description),
		Icon:        db.NullText(req.Icon),
		IsPrivate:   db.NullBool(req.IsPrivate),
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Delete(ctx context.Context, store *Store, id uuid.UUID) error {
	return store.Delete(ctx, id)
}

func (s *Service) AddMember(ctx context.Context, store *Store, spaceID uuid.UUID, req AddMemberRequest) error {
	if req.Role == "" {
		req.Role = "editor"
	}
	_, err := store.AddMember(ctx, dbtenant.AddSpaceMemberParams{
		SpaceID: spaceID,
		UserID:  req.UserID,
		Role:    req.Role,
	})
	return err
}

func (s *Service) RemoveMember(ctx context.Context, store *Store, spaceID, userID uuid.UUID) error {
	return store.RemoveMember(ctx, spaceID, userID)
}

func (s *Service) ListMembers(ctx context.Context, store *Store, spaceID uuid.UUID) ([]MemberResponse, error) {
	rows, err := store.ListMembers(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	result := make([]MemberResponse, len(rows))
	for i, r := range rows {
		result[i] = toMemberResponse(r)
	}
	return result, nil
}

func (s *Service) GetTree(ctx context.Context, store *Store, spaceID uuid.UUID) ([]TreeNode, error) {
	rows, err := store.GetTree(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	return buildTree(rows), nil
}
