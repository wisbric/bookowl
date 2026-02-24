package image

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

// SyncDocumentImages extracts image references from Tiptap JSON content
// and upserts/removes document_images rows to keep them in sync.
func SyncDocumentImages(ctx context.Context, q *dbtenant.Queries, docID uuid.UUID, content json.RawMessage) {
	imageIDs := ExtractImageIDs(content)

	if len(imageIDs) == 0 {
		// No images in content — remove all references.
		if err := q.DeleteDocumentImagesByDocument(ctx, docID); err != nil {
			slog.Error("clearing document images", "doc_id", docID, "error", err)
		}
		return
	}

	// Upsert references for current images.
	for _, storageID := range imageIDs {
		if err := q.UpsertDocumentImage(ctx, dbtenant.UpsertDocumentImageParams{
			DocumentID: docID,
			StorageID:  storageID,
		}); err != nil {
			slog.Error("upserting document image", "doc_id", docID, "storage_id", storageID, "error", err)
		}
	}

	// Remove references for images no longer in the content.
	if err := q.DeleteStaleDocumentImages(ctx, dbtenant.DeleteStaleDocumentImagesParams{
		DocumentID: docID,
		KeepIds:    imageIDs,
	}); err != nil {
		slog.Error("removing stale document images", "doc_id", docID, "error", err)
	}
}
