package document

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

type Store struct {
	conn dbtenant.DBTX
	q    *dbtenant.Queries
}

func NewStore(conn dbtenant.DBTX) *Store {
	return &Store{conn: conn, q: dbtenant.New(conn)}
}

// Queries returns the underlying sqlc queries for use with cross-package helpers.
func (s *Store) Queries() *dbtenant.Queries {
	return s.q
}

func (s *Store) Create(ctx context.Context, params dbtenant.CreateDocumentParams) (dbtenant.Document, error) {
	row, err := s.q.CreateDocument(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return dbtenant.Document{}, ErrSlugConflict
		}
		return dbtenant.Document{}, fmt.Errorf("creating document: %w", err)
	}
	return row, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (dbtenant.Document, error) {
	row, err := s.q.GetDocumentByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Document{}, ErrNotFound
		}
		return dbtenant.Document{}, fmt.Errorf("getting document: %w", err)
	}
	return row, nil
}

func (s *Store) GetBySlug(ctx context.Context, spaceID uuid.UUID, slug string) (dbtenant.Document, error) {
	row, err := s.q.GetDocumentBySlug(ctx, dbtenant.GetDocumentBySlugParams{
		SpaceID: spaceID,
		Slug:    slug,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Document{}, ErrNotFound
		}
		return dbtenant.Document{}, fmt.Errorf("getting document by slug: %w", err)
	}
	return row, nil
}

func (s *Store) List(ctx context.Context, limit, offset int32) ([]dbtenant.Document, error) {
	rows, err := s.q.ListDocuments(ctx, dbtenant.ListDocumentsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing documents: %w", err)
	}
	return rows, nil
}

func (s *Store) ListBySpace(ctx context.Context, spaceID uuid.UUID, limit, offset int32) ([]dbtenant.Document, error) {
	rows, err := s.q.ListDocumentsBySpace(ctx, dbtenant.ListDocumentsBySpaceParams{
		SpaceID: spaceID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing documents by space: %w", err)
	}
	return rows, nil
}

func (s *Store) ListByCollection(ctx context.Context, collectionID uuid.UUID, limit, offset int32) ([]dbtenant.Document, error) {
	rows, err := s.q.ListDocumentsByCollection(ctx, dbtenant.ListDocumentsByCollectionParams{
		CollectionID: db.ValidUUID(collectionID),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing documents by collection: %w", err)
	}
	return rows, nil
}

func (s *Store) Update(ctx context.Context, params dbtenant.UpdateDocumentParams) (dbtenant.Document, error) {
	row, err := s.q.UpdateDocument(ctx, params)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Document{}, ErrNotFound
		}
		if db.IsUniqueViolation(err) {
			return dbtenant.Document{}, ErrSlugConflict
		}
		return dbtenant.Document{}, fmt.Errorf("updating document: %w", err)
	}
	return row, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteDocument(ctx, id); err != nil {
		return fmt.Errorf("deleting document: %w", err)
	}
	return nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	return s.q.CountDocuments(ctx)
}

func (s *Store) CountBySpace(ctx context.Context, spaceID uuid.UUID) (int64, error) {
	return s.q.CountDocumentsBySpace(ctx, spaceID)
}

func (s *Store) CreateVersion(ctx context.Context, params dbtenant.CreateDocumentVersionParams) (dbtenant.DocumentVersion, error) {
	row, err := s.q.CreateDocumentVersion(ctx, params)
	if err != nil {
		return dbtenant.DocumentVersion{}, fmt.Errorf("creating document version: %w", err)
	}
	return row, nil
}

func (s *Store) GetVersion(ctx context.Context, docID uuid.UUID, version int32) (dbtenant.DocumentVersion, error) {
	row, err := s.q.GetDocumentVersion(ctx, dbtenant.GetDocumentVersionParams{
		DocumentID: docID,
		Version:    version,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.DocumentVersion{}, ErrNotFound
		}
		return dbtenant.DocumentVersion{}, fmt.Errorf("getting document version: %w", err)
	}
	return row, nil
}

func (s *Store) GetVersionByID(ctx context.Context, id uuid.UUID) (dbtenant.DocumentVersion, error) {
	row, err := s.q.GetDocumentVersionByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.DocumentVersion{}, ErrNotFound
		}
		return dbtenant.DocumentVersion{}, fmt.Errorf("getting document version by ID: %w", err)
	}
	return row, nil
}

func (s *Store) ListVersions(ctx context.Context, docID uuid.UUID, limit, offset int32) ([]dbtenant.ListDocumentVersionsRow, error) {
	rows, err := s.q.ListDocumentVersions(ctx, dbtenant.ListDocumentVersionsParams{
		DocumentID: docID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing document versions: %w", err)
	}
	return rows, nil
}

func (s *Store) MoveToSpace(ctx context.Context, id, spaceID uuid.UUID, collectionID *uuid.UUID, slug string) error {
	colID := db.NullUUID(collectionID)
	_, err := s.conn.Exec(ctx,
		`UPDATE documents SET space_id = $2, collection_id = $3, slug = $4, updated_at = now() WHERE id = $1`,
		id, spaceID, colID, slug,
	)
	if err != nil {
		return fmt.Errorf("moving document to space: %w", err)
	}
	return nil
}

func (s *Store) SlugExistsInSpace(ctx context.Context, spaceID uuid.UUID, slug string) (bool, error) {
	var exists bool
	err := s.conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM documents WHERE space_id = $1 AND slug = $2)`,
		spaceID, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking slug existence: %w", err)
	}
	return exists, nil
}

func (s *Store) Search(ctx context.Context, query string, limit, offset int32) ([]dbtenant.SearchDocumentsRow, error) {
	rows, err := s.q.SearchDocuments(ctx, dbtenant.SearchDocumentsParams{
		Query:        query,
		ResultLimit:  limit,
		ResultOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("searching documents: %w", err)
	}
	return rows, nil
}

func (s *Store) SearchBySpace(ctx context.Context, query string, spaceID uuid.UUID, limit, offset int32) ([]dbtenant.SearchDocumentsBySpaceRow, error) {
	rows, err := s.q.SearchDocumentsBySpace(ctx, dbtenant.SearchDocumentsBySpaceParams{
		Query:        query,
		SpaceID:      spaceID,
		ResultLimit:  limit,
		ResultOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("searching documents by space: %w", err)
	}
	return rows, nil
}

func (s *Store) SearchByType(ctx context.Context, query, docType string, limit, offset int32) ([]dbtenant.SearchDocumentsByTypeRow, error) {
	rows, err := s.q.SearchDocumentsByType(ctx, dbtenant.SearchDocumentsByTypeParams{
		Query:        query,
		DocType:      docType,
		ResultLimit:  limit,
		ResultOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("searching documents by type: %w", err)
	}
	return rows, nil
}
