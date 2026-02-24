package template

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

func (s *Store) Queries() *dbtenant.Queries {
	return s.q
}

func (s *Store) Create(ctx context.Context, params dbtenant.CreateTemplateParams) (dbtenant.Template, error) {
	row, err := s.q.CreateTemplate(ctx, params)
	if err != nil {
		return dbtenant.Template{}, fmt.Errorf("creating template: %w", err)
	}
	return row, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (dbtenant.Template, error) {
	row, err := s.q.GetTemplateByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Template{}, ErrNotFound
		}
		return dbtenant.Template{}, fmt.Errorf("getting template: %w", err)
	}
	return row, nil
}

func (s *Store) List(ctx context.Context, spaceID *uuid.UUID) ([]dbtenant.Template, error) {
	if spaceID != nil {
		rows, err := s.q.ListTemplates(ctx, *spaceID)
		if err != nil {
			return nil, fmt.Errorf("listing templates: %w", err)
		}
		return rows, nil
	}
	rows, err := s.q.ListAllTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing all templates: %w", err)
	}
	return rows, nil
}

func (s *Store) ListByDocType(ctx context.Context, docType string, spaceID uuid.UUID) ([]dbtenant.Template, error) {
	rows, err := s.q.ListTemplatesByDocType(ctx, dbtenant.ListTemplatesByDocTypeParams{
		DocType: docType,
		SpaceID: spaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing templates by doc type: %w", err)
	}
	return rows, nil
}

func (s *Store) Update(ctx context.Context, params dbtenant.UpdateTemplateParams) (dbtenant.Template, error) {
	row, err := s.q.UpdateTemplate(ctx, params)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Template{}, ErrNotFound
		}
		return dbtenant.Template{}, fmt.Errorf("updating template: %w", err)
	}
	return row, nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.DeleteTemplate(ctx, id); err != nil {
		return fmt.Errorf("deleting template: %w", err)
	}
	return nil
}

func (s *Store) GetSystemByDocType(ctx context.Context, docType string) (dbtenant.Template, error) {
	row, err := s.q.GetSystemTemplateByDocType(ctx, docType)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Template{}, ErrNotFound
		}
		return dbtenant.Template{}, fmt.Errorf("getting system template: %w", err)
	}
	return row, nil
}

func (s *Store) GetDocumentByID(ctx context.Context, id uuid.UUID) (dbtenant.Document, error) {
	row, err := s.q.GetDocumentByID(ctx, id)
	if err != nil {
		if db.IsNotFound(err) {
			return dbtenant.Document{}, ErrNotFound
		}
		return dbtenant.Document{}, fmt.Errorf("getting document: %w", err)
	}
	return row, nil
}
