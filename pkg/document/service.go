package document

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
	if req.Title == "" {
		return Response{}, fmt.Errorf("title is required")
	}
	if req.Slug == "" {
		return Response{}, fmt.Errorf("slug is required")
	}

	contentText := ExtractText(req.Content)
	wordCount := CountWords(contentText)

	docType := req.DocType
	if docType == "" {
		docType = "document"
	}
	status := req.Status
	if status == "" {
		status = "draft"
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	row, err := store.Create(ctx, dbtenant.CreateDocumentParams{
		SpaceID:            req.SpaceID,
		CollectionID:       db.NullUUID(req.CollectionID),
		Title:              req.Title,
		Slug:               req.Slug,
		Content:            req.Content,
		ContentText:        contentText,
		DocType:            docType,
		Status:             status,
		Tags:               tags,
		Icon:               db.NullText(req.Icon),
		Position:           req.Position,
		WordCount:          wordCount,
		NightowlIncidentID: db.NullText(req.NightOwlIncidentID),
		NightowlAlertID:    db.NullText(req.NightOwlAlertID),
		CreatedBy:          userID,
		UpdatedBy:          userID,
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

func (s *Service) List(ctx context.Context, store *Store, limit, offset int32) ([]Response, int64, error) {
	total, err := store.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := store.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]Response, len(rows))
	for i, r := range rows {
		result[i] = toResponse(r)
	}
	return result, total, nil
}

func (s *Service) Update(ctx context.Context, store *Store, id uuid.UUID, req UpdateRequest, userID pgtype.UUID) (Response, error) {
	if req.Title == "" {
		return Response{}, fmt.Errorf("title is required")
	}
	if req.Slug == "" {
		return Response{}, fmt.Errorf("slug is required")
	}

	// Get current document to snapshot as a version before updating.
	current, err := store.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}

	// Save current state as a version.
	_, err = store.CreateVersion(ctx, dbtenant.CreateDocumentVersionParams{
		DocumentID:    current.ID,
		Version:       current.Version,
		Title:         current.Title,
		Content:       current.Content,
		ContentText:   current.ContentText,
		DocType:       current.DocType,
		Tags:          current.Tags,
		ChangedBy:     userID,
		ChangeSummary: db.NullText(req.ChangeSummary),
	})
	if err != nil {
		return Response{}, fmt.Errorf("saving version snapshot: %w", err)
	}

	contentText := ExtractText(req.Content)
	wordCount := CountWords(contentText)

	docType := req.DocType
	if docType == "" {
		docType = current.DocType
	}
	status := req.Status
	if status == "" {
		status = current.Status
	}
	tags := req.Tags
	if tags == nil {
		tags = current.Tags
	}

	row, err := store.Update(ctx, dbtenant.UpdateDocumentParams{
		ID:                 id,
		Title:              req.Title,
		Slug:               req.Slug,
		Content:            req.Content,
		ContentText:        contentText,
		CollectionID:       db.NullUUID(req.CollectionID),
		DocType:            docType,
		Status:             status,
		Tags:               tags,
		Icon:               db.NullText(req.Icon),
		Position:           req.Position,
		WordCount:          wordCount,
		NightowlIncidentID: db.NullText(req.NightOwlIncidentID),
		NightowlAlertID:    db.NullText(req.NightOwlAlertID),
		UpdatedBy:          userID,
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Delete(ctx context.Context, store *Store, id uuid.UUID) error {
	return store.Delete(ctx, id)
}

func (s *Service) ListVersions(ctx context.Context, store *Store, docID uuid.UUID, limit, offset int32) ([]VersionResponse, error) {
	rows, err := store.ListVersions(ctx, docID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]VersionResponse, len(rows))
	for i, r := range rows {
		result[i] = toVersionListResponse(r)
	}
	return result, nil
}

func (s *Service) GetVersion(ctx context.Context, store *Store, versionID uuid.UUID) (VersionResponse, error) {
	row, err := store.GetVersionByID(ctx, versionID)
	if err != nil {
		return VersionResponse{}, err
	}
	return toVersionResponse(row), nil
}

func (s *Service) Restore(ctx context.Context, store *Store, docID, versionID uuid.UUID, userID pgtype.UUID) (Response, error) {
	ver, err := store.GetVersionByID(ctx, versionID)
	if err != nil {
		return Response{}, err
	}
	if ver.DocumentID != docID {
		return Response{}, ErrNotFound
	}

	// Save the current state as a version before restoring.
	current, err := store.GetByID(ctx, docID)
	if err != nil {
		return Response{}, err
	}

	summary := fmt.Sprintf("Restored to version %d", ver.Version)
	_, err = store.CreateVersion(ctx, dbtenant.CreateDocumentVersionParams{
		DocumentID:    current.ID,
		Version:       current.Version,
		Title:         current.Title,
		Content:       current.Content,
		ContentText:   current.ContentText,
		DocType:       current.DocType,
		Tags:          current.Tags,
		ChangedBy:     userID,
		ChangeSummary: db.ValidText(summary),
	})
	if err != nil {
		return Response{}, fmt.Errorf("saving version before restore: %w", err)
	}

	contentText := ExtractText(ver.Content)
	wordCount := CountWords(contentText)

	row, err := store.Update(ctx, dbtenant.UpdateDocumentParams{
		ID:                 docID,
		Title:              ver.Title,
		Slug:               current.Slug,
		Content:            ver.Content,
		ContentText:        contentText,
		CollectionID:       current.CollectionID,
		DocType:            ver.DocType,
		Status:             current.Status,
		Tags:               ver.Tags,
		Icon:               current.Icon,
		Position:           current.Position,
		WordCount:          wordCount,
		NightowlIncidentID: current.NightowlIncidentID,
		NightowlAlertID:    current.NightowlAlertID,
		UpdatedBy:          userID,
	})
	if err != nil {
		return Response{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Search(ctx context.Context, store *Store, query string, spaceID *uuid.UUID, docType string, limit, offset int32) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	if spaceID != nil {
		rows, err := store.SearchBySpace(ctx, query, *spaceID, limit, offset)
		if err != nil {
			return nil, err
		}
		result := make([]SearchResult, len(rows))
		for i, r := range rows {
			result[i] = SearchResult{
				ID:               r.ID,
				Title:            r.Title,
				DocType:          r.DocType,
				Slug:             r.Slug,
				SpaceID:          r.SpaceID,
				CollectionID:     db.UUIDPtr(r.CollectionID),
				SpaceName:        r.SpaceName,
				SpaceSlug:        r.SpaceSlug,
				Rank:             r.Rank,
				TitleHighlight:   string(r.TitleHighlight),
				ContentHighlight: string(r.ContentHighlight),
			}
		}
		return result, nil
	}

	if docType != "" {
		rows, err := store.SearchByType(ctx, query, docType, limit, offset)
		if err != nil {
			return nil, err
		}
		result := make([]SearchResult, len(rows))
		for i, r := range rows {
			result[i] = SearchResult{
				ID:               r.ID,
				Title:            r.Title,
				DocType:          r.DocType,
				Slug:             r.Slug,
				SpaceID:          r.SpaceID,
				CollectionID:     db.UUIDPtr(r.CollectionID),
				SpaceName:        r.SpaceName,
				SpaceSlug:        r.SpaceSlug,
				Rank:             r.Rank,
				TitleHighlight:   string(r.TitleHighlight),
				ContentHighlight: string(r.ContentHighlight),
			}
		}
		return result, nil
	}

	rows, err := store.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]SearchResult, len(rows))
	for i, r := range rows {
		result[i] = SearchResult{
			ID:               r.ID,
			Title:            r.Title,
			DocType:          r.DocType,
			Slug:             r.Slug,
			SpaceID:          r.SpaceID,
			CollectionID:     db.UUIDPtr(r.CollectionID),
			SpaceName:        r.SpaceName,
			SpaceSlug:        r.SpaceSlug,
			Rank:             r.Rank,
			TitleHighlight:   string(r.TitleHighlight),
			ContentHighlight: string(r.ContentHighlight),
		}
	}
	return result, nil
}
