package image

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/bookowl/pkg/storage"
	"github.com/wisbric/bookowl/pkg/tenant"
)

const maxImageSize = 10 << 20 // 10MB

var allowedContentTypes = map[string]string{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

type UploadResponse struct {
	ID  uuid.UUID `json:"id"`
	URL string    `json:"url"`
}

type Handler struct {
	backend storage.Backend
}

func NewHandler(backend storage.Backend) *Handler {
	return &Handler{backend: backend}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Upload)
	r.Get("/{id}", h.Serve)
	r.Delete("/{id}", h.Delete)
	return r
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Limit request body to maxImageSize + 1KB for multipart overhead.
	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize+1024)

	file, header, err := r.FormFile("file")
	if err != nil {
		if err.Error() == "http: request body too large" {
httpserver.RespondError(w, http.StatusRequestEntityTooLarge, "error", "image exceeds 10MB limit")
			return
		}
httpserver.RespondError(w, http.StatusBadRequest, "error", "missing or invalid file field")
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxImageSize {
httpserver.RespondError(w, http.StatusRequestEntityTooLarge, "error", "image exceeds 10MB limit")
		return
	}

	// Detect content type from file content, not from the header.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
httpserver.RespondError(w, http.StatusBadRequest, "error", "cannot read file")
		return
	}
	detectedType := http.DetectContentType(buf[:n])

	// SVGs are detected as text/xml or application/xml; check the filename extension too.
	if strings.HasSuffix(strings.ToLower(header.Filename), ".svg") {
		detectedType = "image/svg+xml"
	}

	ext, ok := allowedContentTypes[detectedType]
	if !ok {
httpserver.RespondError(w, http.StatusUnsupportedMediaType, "error", fmt.Sprintf("unsupported content type: %s", detectedType))
		return
	}

	// Seek back to the start to include the sniffed bytes.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "cannot rewind file")
		return
	}

	// Generate storage key.
	t, ok := tenant.FromContext(r.Context())
	if !ok {
httpserver.RespondError(w, http.StatusInternalServerError, "error", "tenant not resolved")
		return
	}

	objectID := uuid.New()
	storageKey := fmt.Sprintf("images/%s/%s%s", t.Slug, objectID.String(), ext)

	// Store the file.
	if err := h.backend.Put(r.Context(), storageKey, file, header.Size, detectedType); err != nil {
		slog.Error("storing image", "key", storageKey, "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to store image")
		return
	}

	// Insert storage_objects record.
	q := dbtenant.New(tenant.ConnFromContext(r.Context()))
	userID := tenant.UserIDFromContext(r.Context())

	obj, err := q.CreateStorageObject(r.Context(), dbtenant.CreateStorageObjectParams{
		StorageKey:  storageKey,
		Filename:    header.Filename,
		ContentType: detectedType,
		SizeBytes:   header.Size,
		Backend:     h.backend.Name(),
		UploadedBy:  userID,
	})
	if err != nil {
		// Best-effort cleanup of the stored file.
		_ = h.backend.Delete(r.Context(), storageKey)
		slog.Error("inserting storage object", "key", storageKey, "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to save image record")
		return
	}

	httpserver.Respond(w, http.StatusCreated, UploadResponse{
		ID:  obj.ID,
		URL: h.backend.URL(storageKey),
	})
}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	q := dbtenant.New(tenant.ConnFromContext(r.Context()))
	obj, err := q.GetStorageObject(r.Context(), id)
	if err != nil {
		if db.IsNotFound(err) {
httpserver.RespondError(w, http.StatusNotFound, "error", "image not found")
			return
		}
		slog.Error("getting storage object", "id", id, "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}

	// For S3 with presign/public URL, redirect.
	if s3b, ok := h.backend.(*storage.S3Backend); ok {
		var redirectURL string
		if s3b.Name() == "s3" {
			presigned, err := s3b.PresignedURL(r.Context(), obj.StorageKey)
			if err != nil {
				slog.Error("presigning S3 URL", "key", obj.StorageKey, "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to generate image URL")
				return
			}
			redirectURL = presigned
		}
		if redirectURL != "" {
			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}
	}

	// Local backend: stream the file.
	reader, size, err := h.backend.Get(r.Context(), obj.StorageKey)
	if err != nil {
		slog.Error("reading image file", "key", obj.StorageKey, "error", err)
httpserver.RespondError(w, http.StatusNotFound, "error", "image file not found")
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, reader); err != nil {
		slog.Error("streaming image", "key", obj.StorageKey, "error", err)
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	q := dbtenant.New(tenant.ConnFromContext(r.Context()))
	obj, err := q.GetStorageObject(r.Context(), id)
	if err != nil {
		if db.IsNotFound(err) {
httpserver.RespondError(w, http.StatusNotFound, "error", "image not found")
			return
		}
		slog.Error("getting storage object for delete", "id", id, "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}

	// Delete from storage backend.
	if err := h.backend.Delete(r.Context(), obj.StorageKey); err != nil {
		slog.Warn("failed to delete image from backend", "key", obj.StorageKey, "error", err)
	}

	// Delete the DB record (cascades to document_images).
	if err := q.DeleteStorageObject(r.Context(), id); err != nil {
		slog.Error("deleting storage object record", "id", id, "error", err)
httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to delete image record")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExtractImageIDs walks a Tiptap JSON document and extracts all image UUIDs
// from image nodes whose src matches /api/v1/images/{uuid}.
func ExtractImageIDs(content []byte) []uuid.UUID {
	if len(content) == 0 {
		return nil
	}
	var ids []uuid.UUID
	extractImagesRecursive(content, &ids)
	return ids
}

func extractImagesRecursive(data []byte, ids *[]uuid.UUID) {
	// Fast path: avoid full JSON unmarshal. Walk the raw bytes looking for image src fields.
	// This is safe because we only extract UUIDs from known URL patterns.
	// But for correctness, let's use proper JSON parsing.
	type node struct {
		Type    string          `json:"type"`
		Attrs   map[string]any  `json:"attrs,omitempty"`
		Content []jsonNode      `json:"content,omitempty"`
	}
	type doc = node

	var n doc
	if err := jsonUnmarshal(data, &n); err != nil {
		return
	}
	walkNode(n.Type, n.Attrs, n.Content, ids)
}

type jsonNode struct {
	Type    string          `json:"type"`
	Attrs   map[string]any  `json:"attrs,omitempty"`
	Content []jsonNode      `json:"content,omitempty"`
}

func walkNode(nodeType string, attrs map[string]any, children []jsonNode, ids *[]uuid.UUID) {
	if nodeType == "image" {
		if src, ok := attrs["src"].(string); ok {
			if id := extractIDFromImageURL(src); id != uuid.Nil {
				*ids = append(*ids, id)
			}
		}
	}
	for _, child := range children {
		walkNode(child.Type, child.Attrs, child.Content, ids)
	}
}

// extractIDFromImageURL extracts a UUID from /api/v1/images/{uuid}.
func extractIDFromImageURL(src string) uuid.UUID {
	const prefix = "/api/v1/images/"
	if !strings.HasPrefix(src, prefix) {
		return uuid.Nil
	}
	raw := strings.TrimPrefix(src, prefix)
	// Remove any query string or fragment.
	if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
		raw = raw[:idx]
	}
	// Remove file extension if present.
	raw = strings.TrimSuffix(raw, filepath.Ext(raw))
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
