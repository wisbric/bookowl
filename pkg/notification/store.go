package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type Store struct {
	q *dbtenant.Queries
}

func NewStore(conn dbtenant.DBTX) *Store {
	return &Store{q: dbtenant.New(conn)}
}

func (s *Store) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.q.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}

func (s *Store) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbtenant.Notification, error) {
	rows, err := s.q.ListNotifications(ctx, dbtenant.ListNotificationsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}
	return rows, nil
}

func (s *Store) MarkRead(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	if err := s.q.MarkNotificationRead(ctx, dbtenant.MarkNotificationReadParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("marking notification read: %w", err)
	}
	return nil
}

func (s *Store) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.q.MarkAllNotificationsRead(ctx, userID); err != nil {
		return fmt.Errorf("marking all notifications read: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (dbtenant.User, error) {
	return s.q.GetUserByID(ctx, id)
}
