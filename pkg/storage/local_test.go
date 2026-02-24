package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalBackend_PutGetDelete(t *testing.T) {
	dir := t.TempDir()
	b := NewLocalBackend(dir)

	ctx := context.Background()
	key := "images/acme/test-uuid.jpg"
	data := []byte("fake image data")

	// Put
	err := b.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "image/jpeg")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Verify file exists on disk.
	path := filepath.Join(dir, key)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not found on disk: %v", err)
	}

	// Get
	reader, size, err := b.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = reader.Close() }()

	if size != int64(len(data)) {
		t.Errorf("size = %d, want %d", size, len(data))
	}

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("data mismatch")
	}

	// Delete
	err = b.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still exists after delete")
	}
}

func TestLocalBackend_DeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	b := NewLocalBackend(dir)

	// Should not error when deleting non-existent file.
	err := b.Delete(context.Background(), "images/acme/no-such-file.jpg")
	if err != nil {
		t.Errorf("Delete non-existent: %v", err)
	}
}

func TestLocalBackend_URL(t *testing.T) {
	b := NewLocalBackend("/tmp/test")

	tests := []struct {
		key  string
		want string
	}{
		{"images/acme/550e8400-e29b-41d4-a716-446655440000.jpg", "/api/v1/images/550e8400-e29b-41d4-a716-446655440000"},
		{"images/bigcorp/abc.png", "/api/v1/images/abc"},
		{"short", ""},
	}
	for _, tt := range tests {
		got := b.URL(tt.key)
		if got != tt.want {
			t.Errorf("URL(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestLocalBackend_Name(t *testing.T) {
	b := NewLocalBackend("/tmp")
	if b.Name() != "local" {
		t.Errorf("Name() = %q, want %q", b.Name(), "local")
	}
}
