package document

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/pkg/image"
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

	// If a template_id is provided, pre-fill content and doc_type from the template.
	if req.TemplateID != nil {
		tmpl, err := store.Queries().GetTemplateByID(ctx, *req.TemplateID)
		if err == nil {
			if len(req.Content) == 0 {
				req.Content = tmpl.Content
			}
			if req.DocType == "" {
				req.DocType = tmpl.DocType
			}
		}
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

	// Sync image references from Tiptap content.
	image.SyncDocumentImages(ctx, store.Queries(), row.ID, req.Content)

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

	// Sync image references from Tiptap content.
	image.SyncDocumentImages(ctx, store.Queries(), row.ID, req.Content)

	return toResponse(row), nil
}

func (s *Service) Delete(ctx context.Context, store *Store, id uuid.UUID) error {
	return store.Delete(ctx, id)
}

// Move moves a document to a different space (and optionally a collection within it).
// Handles slug collisions by appending -1, -2, etc.
func (s *Service) Move(ctx context.Context, store *Store, id uuid.UUID, req MoveRequest) (Response, error) {
	doc, err := store.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}

	// Resolve a unique slug in the target space.
	slug := doc.Slug
	for i := 1; ; i++ {
		exists, err := store.SlugExistsInSpace(ctx, req.SpaceID, slug)
		if err != nil {
			return Response{}, fmt.Errorf("checking slug: %w", err)
		}
		if !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", doc.Slug, i)
	}

	if err := store.MoveToSpace(ctx, id, req.SpaceID, req.CollectionID, slug); err != nil {
		return Response{}, err
	}

	moved, err := store.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	return toResponse(moved), nil
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

	// Sync image references from restored content.
	image.SyncDocumentImages(ctx, store.Queries(), row.ID, ver.Content)

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
