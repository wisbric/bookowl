package comment

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

const editWindow = 15 * time.Minute

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Create(ctx context.Context, store *Store, docID uuid.UUID, req CreateRequest, userID uuid.UUID) (CommentResponse, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return CommentResponse{}, ErrEmptyBody
	}

	rendered := RenderBody(body)

	row, err := store.Create(ctx, dbtenant.CreateCommentParams{
		DocumentID:   docID,
		AuthorID:     userID,
		Body:         body,
		BodyRendered: rendered,
	})
	if err != nil {
		return CommentResponse{}, fmt.Errorf("creating comment: %w", err)
	}

	// Create notifications for @mentions.
	s.notifyMentions(ctx, store, body, docID, row.ID, userID)

	author, _ := store.GetUserByID(ctx, userID)
	return toResponse(row, makeAuthorInfo(author), []CommentResponse{}), nil
}

func (s *Service) Reply(ctx context.Context, store *Store, docID uuid.UUID, parentID uuid.UUID, req CreateRequest, userID uuid.UUID) (CommentResponse, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return CommentResponse{}, ErrEmptyBody
	}

	parent, err := store.GetByID(ctx, parentID)
	if err != nil {
		return CommentResponse{}, err
	}

	// Only allow replies to top-level comments.
	if parent.ParentID.Valid {
		return CommentResponse{}, ErrReplyToReply
	}

	rendered := RenderBody(body)

	row, err := store.Create(ctx, dbtenant.CreateCommentParams{
		DocumentID:   docID,
		ParentID:     pgtype.UUID{Bytes: parentID, Valid: true},
		AuthorID:     userID,
		Body:         body,
		BodyRendered: rendered,
	})
	if err != nil {
		return CommentResponse{}, fmt.Errorf("creating reply: %w", err)
	}

	// Notify parent comment author.
	if parent.AuthorID != userID {
		_, err := store.CreateNotification(ctx, dbtenant.CreateNotificationParams{
			UserID:     parent.AuthorID,
			Type:       "comment_reply",
			DocumentID: db.ValidUUID(docID),
			CommentID:  db.ValidUUID(row.ID),
			ActorID:    db.ValidUUID(userID),
		})
		if err != nil {
			slog.Error("creating reply notification", "error", err)
		}
	}

	// Notify @mentions.
	s.notifyMentions(ctx, store, body, docID, row.ID, userID)

	author, _ := store.GetUserByID(ctx, userID)
	return toResponse(row, makeAuthorInfo(author), nil), nil
}

func (s *Service) Update(ctx context.Context, store *Store, commentID uuid.UUID, req UpdateRequest, userID uuid.UUID, isAdmin bool) (CommentResponse, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return CommentResponse{}, ErrEmptyBody
	}

	existing, err := store.GetByID(ctx, commentID)
	if err != nil {
		return CommentResponse{}, err
	}

	if existing.AuthorID != userID && !isAdmin {
		return CommentResponse{}, ErrNotAuthor
	}

	if time.Since(existing.CreatedAt) > editWindow && !isAdmin {
		return CommentResponse{}, ErrEditWindowExpired
	}

	rendered := RenderBody(body)

	row, err := store.UpdateBody(ctx, dbtenant.UpdateCommentBodyParams{
		ID:           commentID,
		Body:         body,
		BodyRendered: rendered,
	})
	if err != nil {
		return CommentResponse{}, err
	}

	author, _ := store.GetUserByID(ctx, existing.AuthorID)
	return toResponse(row, makeAuthorInfo(author), nil), nil
}

func (s *Service) Delete(ctx context.Context, store *Store, commentID uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	existing, err := store.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	if existing.AuthorID != userID && !isAdmin {
		return ErrNotAuthor
	}

	return store.SoftDelete(ctx, commentID)
}

func (s *Service) Resolve(ctx context.Context, store *Store, commentID uuid.UUID, userID uuid.UUID) error {
	existing, err := store.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	if existing.ParentID.Valid {
		return ErrNotTopLevel
	}

	return store.Resolve(ctx, commentID, userID)
}

func (s *Service) Unresolve(ctx context.Context, store *Store, commentID uuid.UUID) error {
	existing, err := store.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	if existing.ParentID.Valid {
		return ErrNotTopLevel
	}

	return store.Unresolve(ctx, commentID)
}

func (s *Service) List(ctx context.Context, store *Store, docID uuid.UUID, includeResolved bool) ([]CommentResponse, error) {
	topLevel, err := store.ListByDocument(ctx, docID)
	if err != nil {
		return nil, err
	}

	// Collect all unique author IDs for batch lookup.
	authorIDs := make(map[uuid.UUID]bool)
	for _, c := range topLevel {
		authorIDs[c.AuthorID] = true
	}

	// Fetch replies for each top-level comment.
	repliesMap := make(map[uuid.UUID][]dbtenant.Comment)
	for _, c := range topLevel {
		replies, err := store.ListReplies(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		repliesMap[c.ID] = replies
		for _, r := range replies {
			authorIDs[r.AuthorID] = true
		}
	}

	// Fetch all authors.
	authors := make(map[uuid.UUID]AuthorInfo)
	for id := range authorIDs {
		u, err := store.GetUserByID(ctx, id)
		if err != nil {
			authors[id] = AuthorInfo{ID: id, DisplayName: "Unknown", Initials: "?"}
			continue
		}
		authors[id] = makeAuthorInfo(u)
	}

	result := make([]CommentResponse, 0, len(topLevel))
	for _, c := range topLevel {
		if !includeResolved && c.IsResolved {
			continue
		}

		replyResponses := make([]CommentResponse, 0, len(repliesMap[c.ID]))
		for _, r := range repliesMap[c.ID] {
			replyResponses = append(replyResponses, toResponse(r, authors[r.AuthorID], nil))
		}

		result = append(result, toResponse(c, authors[c.AuthorID], replyResponses))
	}

	return result, nil
}

func (s *Service) Count(ctx context.Context, store *Store, docID uuid.UUID) (int64, error) {
	return store.Count(ctx, docID)
}

func (s *Service) notifyMentions(ctx context.Context, store *Store, body string, docID, commentID, actorID uuid.UUID) {
	mentions := ExtractMentions(body)
	if len(mentions) == 0 {
		return
	}

	// Resolve mentioned usernames to users by treating them as emails or partial emails.
	users, err := store.GetUsersByEmails(ctx, mentions)
	if err != nil {
		slog.Error("resolving mentioned users", "error", err)
		return
	}

	for _, u := range users {
		if u.ID == actorID {
			continue // skip self-mention
		}
		_, err := store.CreateNotification(ctx, dbtenant.CreateNotificationParams{
			UserID:     u.ID,
			Type:       "comment_mention",
			DocumentID: db.ValidUUID(docID),
			CommentID:  db.ValidUUID(commentID),
			ActorID:    db.ValidUUID(actorID),
		})
		if err != nil {
			slog.Error("creating mention notification", "error", err, "user_id", u.ID)
		}
	}
}
