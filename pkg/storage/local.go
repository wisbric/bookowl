package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalBackend stores files on the local filesystem.
type LocalBackend struct {
	BasePath string
}

func NewLocalBackend(basePath string) *LocalBackend {
	return &LocalBackend{BasePath: basePath}
}

func (b *LocalBackend) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	path := filepath.Join(b.BasePath, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

func (b *LocalBackend) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	path := filepath.Join(b.BasePath, key)
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening file: %w", err)
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, fmt.Errorf("stat file: %w", err)
	}
	return f, stat.Size(), nil
}

func (b *LocalBackend) Delete(_ context.Context, key string) error {
	path := filepath.Join(b.BasePath, key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file: %w", err)
	}
	return nil
}

func (b *LocalBackend) URL(key string) string {
	// key format: images/{tenant_slug}/{uuid}.{ext}
	// URL format: /api/v1/images/{uuid}
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		return ""
	}
	filename := parts[len(parts)-1]
	id := strings.TrimSuffix(filename, filepath.Ext(filename))
	return "/api/v1/images/" + id
}

func (b *LocalBackend) Name() string { return "local" }
