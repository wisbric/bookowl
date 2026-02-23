package collection

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

func (s *Store) Create(ctx context.Context, params dbtenant.CreateCollectionParams) (dbtenant.Collection, error) {
	row, err := s.q.CreateCollection(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return dbtenant.Collection{}, ErrSlugConflict
		}
		return dbtenant.Collection{}, fmt.Errorf("creating collection: %w", err)
	}
	return row, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (dbtenant.Collection, error) {
	row, err := s.q.GetCollectionByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Collection{}, ErrNotFound
		}
		return dbtenant.Collection{}, fmt.Errorf("getting collection: %w", err)
	}
	return row, nil
}

func (s *Store) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]dbtenant.Collection, error) {
	rows, err := s.q.ListCollectionsBySpace(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("listing collections: %w", err)
	}
	return rows, nil
}

func (s *Store) Update(ctx context.Context, params dbtenant.UpdateCollectionParams) (dbtenant.Collection, error) {
	row, err := s.q.UpdateCollection(ctx, params)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Collection{}, ErrNotFound
		}
		if db.IsUniqueViolation(err) {
			return dbtenant.Collection{}, ErrSlugConflict
		}
		return dbtenant.Collection{}, fmt.Errorf("updating collection: %w", err)
	}
	return row, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteCollection(ctx, id); err != nil {
		return fmt.Errorf("deleting collection: %w", err)
	}
	return nil
}
