package comment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type Store struct {
	q *dbtenant.Queries
}

func NewStore(conn dbtenant.DBTX) *Store {
	return &Store{q: dbtenant.New(conn)}
}

func (s *Store) Create(ctx context.Context, params dbtenant.CreateCommentParams) (dbtenant.Comment, error) {
	row, err := s.q.CreateComment(ctx, params)
	if err != nil {
		return dbtenant.Comment{}, fmt.Errorf("creating comment: %w", err)
	}
	return row, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (dbtenant.Comment, error) {
	row, err := s.q.GetCommentByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Comment{}, ErrNotFound
		}
		return dbtenant.Comment{}, fmt.Errorf("getting comment: %w", err)
	}
	return row, nil
}

func (s *Store) ListByDocument(ctx context.Context, docID uuid.UUID) ([]dbtenant.Comment, error) {
	rows, err := s.q.ListCommentsByDocument(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	return rows, nil
}

func (s *Store) ListReplies(ctx context.Context, parentID uuid.UUID) ([]dbtenant.Comment, error) {
	rows, err := s.q.ListReplies(ctx, pgtype.UUID{Bytes: parentID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("listing replies: %w", err)
	}
	return rows, nil
}

func (s *Store) UpdateBody(ctx context.Context, params dbtenant.UpdateCommentBodyParams) (dbtenant.Comment, error) {
	row, err := s.q.UpdateCommentBody(ctx, params)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Comment{}, ErrNotFound
		}
		return dbtenant.Comment{}, fmt.Errorf("updating comment: %w", err)
	}
	return row, nil
}

func (s *Store) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.SoftDeleteComment(ctx, id); err != nil {
		return fmt.Errorf("soft deleting comment: %w", err)
	}
	return nil
}

func (s *Store) Resolve(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID) error {
	if err := s.q.ResolveComment(ctx, dbtenant.ResolveCommentParams{
		ID:         id,
		ResolvedBy: pgtype.UUID{Bytes: resolvedBy, Valid: true},
	}); err != nil {
		return fmt.Errorf("resolving comment: %w", err)
	}
	return nil
}

func (s *Store) Unresolve(ctx context.Context, id uuid.UUID) error {
	if err := s.q.UnresolveComment(ctx, id); err != nil {
		return fmt.Errorf("unresolving comment: %w", err)
	}
	return nil
}

func (s *Store) Count(ctx context.Context, docID uuid.UUID) (int64, error) {
	count, err := s.q.CountCommentsByDocument(ctx, docID)
	if err != nil {
		return 0, fmt.Errorf("counting comments: %w", err)
	}
	return count, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (dbtenant.User, error) {
	return s.q.GetUserByID(ctx, id)
}

func (s *Store) GetUsersByEmails(ctx context.Context, emails []string) ([]dbtenant.User, error) {
	return s.q.GetUsersByEmails(ctx, emails)
}

func (s *Store) CreateNotification(ctx context.Context, params dbtenant.CreateNotificationParams) (dbtenant.Notification, error) {
	return s.q.CreateNotification(ctx, params)
}
