package storage

import (
	"context"
	"io"
)

// Backend is the interface all storage implementations satisfy.
type Backend interface {
	// Put stores an object and returns its storage key.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Get retrieves an object by key.
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error

	// URL returns the public URL for a key.
	// For local: returns /api/v1/images/:id (served by BookOwl)
	// For S3: returns a presigned URL or public bucket URL depending on config
	URL(key string) string

	// Name returns the backend identifier ("local" or "s3").
	Name() string
}
