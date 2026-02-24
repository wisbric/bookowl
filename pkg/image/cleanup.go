package image

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/pkg/storage"
)

// CleanupOrphans deletes storage objects not referenced in any document_images row
// and older than minAge (default: 24h to allow for in-progress edits).
func CleanupOrphans(ctx context.Context, q *dbtenant.Queries, backend storage.Backend, minAge time.Duration) (int, error) {
	// Convert duration to pgtype.Interval.
	interval := pgtype.Interval{
		Microseconds: minAge.Microseconds(),
		Valid:        true,
	}

	orphans, err := q.ListOrphanedStorageObjects(ctx, interval)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, obj := range orphans {
		if err := backend.Delete(ctx, obj.StorageKey); err != nil {
			slog.Warn("failed to delete orphan from storage", "key", obj.StorageKey, "err", err)
			continue
		}
		if err := q.DeleteStorageObject(ctx, obj.ID); err != nil {
			slog.Warn("failed to delete orphan record", "id", obj.ID, "err", err)
			continue
		}
		deleted++
	}
	return deleted, nil
}
