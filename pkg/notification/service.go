package notification

import (
	"context"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) UnreadCount(ctx context.Context, store *Store, userID uuid.UUID) (int64, error) {
	return store.UnreadCount(ctx, userID)
}

func (s *Service) List(ctx context.Context, store *Store, userID uuid.UUID, limit, offset int32) ([]NotificationResponse, error) {
	rows, err := store.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Collect unique actor IDs.
	actorIDs := make(map[uuid.UUID]bool)
	for _, n := range rows {
		if n.ActorID.Valid {
			actorIDs[uuid.UUID(n.ActorID.Bytes)] = true
		}
	}

	// Fetch actors.
	actors := make(map[uuid.UUID]AuthorInfo)
	for id := range actorIDs {
		u, err := store.GetUserByID(ctx, id)
		if err != nil {
			actors[id] = AuthorInfo{ID: id, DisplayName: "Unknown", Initials: "?"}
			continue
		}
		actors[id] = makeAuthorInfo(u)
	}

	result := make([]NotificationResponse, 0, len(rows))
	for _, n := range rows {
		var actor *AuthorInfo
		if n.ActorID.Valid {
			a := actors[uuid.UUID(n.ActorID.Bytes)]
			actor = &a
		}
		result = append(result, toResponse(n, actor))
	}

	return result, nil
}

func (s *Service) MarkRead(ctx context.Context, store *Store, userID uuid.UUID, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := store.MarkRead(ctx, userID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, store *Store, userID uuid.UUID) error {
	return store.MarkAllRead(ctx, userID)
}

// Used by the db helper to convert pgtype.UUID.
var _ = db.UUIDPtr
