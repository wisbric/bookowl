package space

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type Store struct {
	q *dbtenant.Queries
}

func NewStore(conn dbtenant.DBTX) *Store {
	return &Store{q: dbtenant.New(conn)}
}

func (s *Store) Create(ctx context.Context, params dbtenant.CreateSpaceParams) (dbtenant.Space, error) {
	row, err := s.q.CreateSpace(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return dbtenant.Space{}, ErrSlugConflict
		}
		return dbtenant.Space{}, fmt.Errorf("creating space: %w", err)
	}
	return row, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (dbtenant.Space, error) {
	row, err := s.q.GetSpaceByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Space{}, ErrNotFound
		}
		return dbtenant.Space{}, fmt.Errorf("getting space: %w", err)
	}
	return row, nil
}

func (s *Store) GetBySlug(ctx context.Context, slug string) (dbtenant.Space, error) {
	row, err := s.q.GetSpaceBySlug(ctx, slug)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Space{}, ErrNotFound
		}
		return dbtenant.Space{}, fmt.Errorf("getting space by slug: %w", err)
	}
	return row, nil
}

func (s *Store) List(ctx context.Context) ([]dbtenant.Space, error) {
	rows, err := s.q.ListSpaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing spaces: %w", err)
	}
	return rows, nil
}

func (s *Store) Update(ctx context.Context, params dbtenant.UpdateSpaceParams) (dbtenant.Space, error) {
	row, err := s.q.UpdateSpace(ctx, params)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Space{}, ErrNotFound
		}
		if db.IsUniqueViolation(err) {
			return dbtenant.Space{}, ErrSlugConflict
		}
		return dbtenant.Space{}, fmt.Errorf("updating space: %w", err)
	}
	return row, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteSpace(ctx, id); err != nil {
		return fmt.Errorf("deleting space: %w", err)
	}
	return nil
}

func (s *Store) AddMember(ctx context.Context, params dbtenant.AddSpaceMemberParams) (dbtenant.SpaceMember, error) {
	row, err := s.q.AddSpaceMember(ctx, params)
	if err != nil {
		return dbtenant.SpaceMember{}, fmt.Errorf("adding space member: %w", err)
	}
	return row, nil
}

func (s *Store) RemoveMember(ctx context.Context, spaceID, userID uuid.UUID) error {
	if err := s.q.RemoveSpaceMember(ctx, dbtenant.RemoveSpaceMemberParams{
		SpaceID: spaceID,
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("removing space member: %w", err)
	}
	return nil
}

func (s *Store) ListMembers(ctx context.Context, spaceID uuid.UUID) ([]dbtenant.ListSpaceMembersRow, error) {
	rows, err := s.q.ListSpaceMembers(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("listing space members: %w", err)
	}
	return rows, nil
}

func (s *Store) GetTree(ctx context.Context, spaceID uuid.UUID) ([]dbtenant.GetSpaceTreeRow, error) {
	rows, err := s.q.GetSpaceTree(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("getting space tree: %w", err)
	}
	return rows, nil
}

func (s *Store) GetUncollectedDocuments(ctx context.Context, spaceID uuid.UUID) ([]dbtenant.GetUncollectedDocumentsRow, error) {
	rows, err := s.q.GetUncollectedDocuments(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("getting uncollected documents: %w", err)
	}
	return rows, nil
}
