# BookOwl — Image Storage Specification

## 1. Problem

The block editor supports image uploads (BK-07). Images cannot be stored as base64 inside the document `content` JSONB column — a single image would bloat the column by hundreds of kilobytes, make FTS extraction expensive, and make the content field unusable for diffs and versioning.

Images need a separate storage layer. For KRITIS and public sector clients, that storage must be self-hosted. No calls to S3, Cloudflare Images, or any external CDN.

## 2. Decision

Two storage backends, selectable per tenant via config:

| Backend | When to use | Implementation |
|---------|-------------|----------------|
| **Local filesystem** | Development, small deployments, single-node | Write to a mounted volume |
| **S3-compatible** | Production, Kubernetes, multi-replica | MinIO, Ceph RGW, AWS S3, Hetzner Object Storage |

The default for `make seed` and `docker-compose.yml` is **local filesystem**. The default for the Helm chart is **S3-compatible** (requires MinIO or equivalent).

Never add Google Cloud Storage, Azure Blob, or any hyperscaler-specific SDK. S3-compatible covers everything through the standard AWS SDK.

## 3. Storage Interface

```go
// pkg/storage/storage.go

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
```

## 4. Data Model

### 4.1 `storage_objects` Table

Migration: `000008_create_storage_objects`

```sql
CREATE TABLE storage_objects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key     TEXT NOT NULL UNIQUE,    -- path within storage backend
    filename        TEXT NOT NULL,           -- original filename from upload
    content_type    TEXT NOT NULL,           -- image/jpeg, image/png, image/gif, image/webp
    size_bytes      BIGINT NOT NULL,
    backend         TEXT NOT NULL,           -- "local" or "s3"
    uploaded_by     UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_storage_objects_key ON storage_objects(storage_key);
CREATE INDEX idx_storage_objects_uploaded_by ON storage_objects(uploaded_by);
```

### 4.2 `document_images` Table

Tracks which images are referenced in which documents. Used for orphan detection.

Migration: `000009_create_document_images`

```sql
CREATE TABLE document_images (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    storage_id      UUID NOT NULL REFERENCES storage_objects(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(document_id, storage_id)
);

CREATE INDEX idx_document_images_document ON document_images(document_id);
CREATE INDEX idx_document_images_storage ON document_images(storage_id);
```

When a document is saved, the service:
1. Extracts all image URLs from the Tiptap JSON content
2. Resolves each URL back to a `storage_objects.id`
3. Upserts `document_images` rows for the current set
4. Removes `document_images` rows for images no longer in the content

This keeps the reference table accurate without requiring explicit client-side tracking.

## 5. API Endpoints

```
POST   /api/v1/images
       Content-Type: multipart/form-data
       Field: file (the image)
       → Validates content type (jpeg, png, gif, webp only)
       → Validates size (max 10MB)
       → Generates storage key: images/{tenant_slug}/{uuid}.{ext}
       → Stores via backend
       → Inserts storage_objects row
       → Returns: { "id": "uuid", "url": "/api/v1/images/uuid" }

GET    /api/v1/images/:id
       → Local backend: reads from filesystem, streams with correct Content-Type + Cache-Control
       → S3 backend: 302 redirect to presigned URL (1h expiry) or public URL
       → Returns 404 if not found or belongs to different tenant

DELETE /api/v1/images/:id
       → Deletes from storage backend
       → Deletes storage_objects row (cascades document_images)
       → Returns 204
       → Note: editor calls this when user removes an image block
```

### Content-Type validation

Only accept:
- `image/jpeg`
- `image/png`
- `image/gif`
- `image/webp`
- `image/svg+xml`

Validate by reading the first 512 bytes and using `http.DetectContentType`, not by trusting the client's Content-Type header.

### File size limit

Maximum 10MB per image. Return 413 with a clear error message if exceeded.

## 6. Local Filesystem Backend

```go
// pkg/storage/local.go

type LocalBackend struct {
    BasePath string   // e.g. /data/bookowl-images
    BaseURL  string   // e.g. http://localhost:8081 (for constructing serve URLs)
}

func (b *LocalBackend) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
    path := filepath.Join(b.BasePath, key)
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return fmt.Errorf("creating directory: %w", err)
    }
    f, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("creating file: %w", err)
    }
    defer f.Close()
    if _, err := io.Copy(f, r); err != nil {
        return fmt.Errorf("writing file: %w", err)
    }
    return nil
}

func (b *LocalBackend) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
    path := filepath.Join(b.BasePath, key)
    f, err := os.Open(path)
    if err != nil {
        return nil, 0, fmt.Errorf("opening file: %w", err)
    }
    stat, err := f.Stat()
    if err != nil {
        f.Close()
        return nil, 0, fmt.Errorf("stat file: %w", err)
    }
    return f, stat.Size(), nil
}

func (b *LocalBackend) Delete(ctx context.Context, key string) error {
    path := filepath.Join(b.BasePath, key)
    if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("removing file: %w", err)
    }
    return nil
}

func (b *LocalBackend) URL(key string) string {
    // Extract the UUID from the key — the API serves it by ID, not by key
    // key format: images/{tenant_slug}/{uuid}.{ext}
    // URL format: /api/v1/images/{uuid}
    parts := strings.Split(key, "/")
    if len(parts) < 3 {
        return ""
    }
    filename := parts[len(parts)-1]
    uuid := strings.TrimSuffix(filename, filepath.Ext(filename))
    return "/api/v1/images/" + uuid
}

func (b *LocalBackend) Name() string { return "local" }
```

**Development config:** Set `BOOKOWL_STORAGE_BACKEND=local` and `BOOKOWL_STORAGE_LOCAL_PATH=/tmp/bookowl-images`. The path is created automatically on first use.

**Kubernetes note:** Local backend requires a PersistentVolumeClaim and only works with a single API replica (or a shared ReadWriteMany volume). Use S3 backend in production.

## 7. S3-Compatible Backend

```go
// pkg/storage/s3.go

import "github.com/aws/aws-sdk-go-v2/service/s3"

type S3Backend struct {
    client     *s3.Client
    bucket     string
    publicURL  string   // e.g. https://minio.example.com/bookowl (for public access)
    usePresign bool     // if true: generate presigned URLs; if false: public bucket URL
    presignTTL time.Duration
}
```

Config via env vars:

```bash
BOOKOWL_STORAGE_BACKEND=s3
BOOKOWL_STORAGE_S3_ENDPOINT=https://minio.example.com   # omit for AWS S3
BOOKOWL_STORAGE_S3_BUCKET=bookowl-images
BOOKOWL_STORAGE_S3_REGION=eu-central-1
BOOKOWL_STORAGE_S3_ACCESS_KEY_ID=<key>
BOOKOWL_STORAGE_S3_SECRET_ACCESS_KEY=<secret>
BOOKOWL_STORAGE_S3_PUBLIC_URL=https://minio.example.com/bookowl-images  # for redirect mode
BOOKOWL_STORAGE_S3_USE_PRESIGN=false   # true = presigned URLs, false = redirect to public URL
```

**URL strategy:**

- `USE_PRESIGN=false` (default): `GET /api/v1/images/:id` returns a 302 redirect to `{PUBLIC_URL}/{storage_key}`. Requires the bucket to be publicly readable. Simpler, faster.
- `USE_PRESIGN=true`: `GET /api/v1/images/:id` generates a 1-hour presigned S3 URL and redirects. Use this if the bucket must not be public (stricter security). The browser follows the redirect and loads the image directly from MinIO/S3 — BookOwl is not in the data path.

For KRITIS clients the recommended setup is: MinIO in the same Kubernetes cluster, same namespace, not publicly exposed, presigned URLs. Images never leave the cluster.

## 8. Key Generation

Storage keys follow this pattern:

```
images/{tenant_slug}/{uuid}.{ext}
```

Examples:
```
images/acme/550e8400-e29b-41d4-a716-446655440000.jpg
images/techgmbh/6ba7b810-9dad-11d1-80b4-00c04fd430c8.png
```

This provides tenant isolation at the storage level — even if the bucket is shared between tenants (acceptable for a single deployment), keys are namespaced. Never mix tenant data in keys.

## 9. Tiptap Integration

The Tiptap `Image` extension needs to be configured to upload to BookOwl rather than accepting external URLs directly.

```typescript
// web/src/components/editor/extensions/ImageUpload.ts

import Image from '@tiptap/extension-image'

export const ImageUpload = Image.extend({
  addOptions() {
    return {
      ...this.parent?.(),
      uploadFn: async (file: File): Promise<string> => {
        const formData = new FormData()
        formData.append('file', file)
        const res = await fetch('/api/v1/images', {
          method: 'POST',
          body: formData,
          headers: { 'X-API-Key': getApiKey() },  // or use OIDC token
        })
        if (!res.ok) throw new Error('Upload failed')
        const { url } = await res.json()
        return url
      },
    }
  },

  addProseMirrorPlugins() {
    return [
      // Handle drag-and-drop and paste
      new Plugin({
        props: {
          handleDOMEvents: {
            drop: handleImageDrop(this.options.uploadFn),
            paste: handleImagePaste(this.options.uploadFn),
          },
        },
      }),
    ]
  },
})
```

When an image is inserted into the editor, the URL stored in the Tiptap JSON is the BookOwl API URL (`/api/v1/images/:id`). This means images are always served through BookOwl — they never reference external URLs unless the user pastes a raw URL (which is allowed but not uploaded).

## 10. Orphan Cleanup

Images that are uploaded but never referenced in a document (upload abandoned mid-edit) or images removed from all documents need periodic cleanup.

```go
// pkg/storage/cleanup.go

// CleanupOrphans deletes storage objects not referenced in any document_images row
// and older than minAge (default: 24h to allow for in-progress edits).
func CleanupOrphans(ctx context.Context, store Store, backend Backend, minAge time.Duration) (int, error) {
    orphans, err := store.ListOrphanedObjects(ctx, minAge)
    if err != nil {
        return 0, fmt.Errorf("listing orphans: %w", err)
    }
    deleted := 0
    for _, obj := range orphans {
        if err := backend.Delete(ctx, obj.StorageKey); err != nil {
            slog.Warn("failed to delete orphan", "key", obj.StorageKey, "err", err)
            continue
        }
        if err := store.DeleteObject(ctx, obj.ID); err != nil {
            slog.Warn("failed to delete orphan record", "id", obj.ID, "err", err)
            continue
        }
        deleted++
    }
    return deleted, nil
}
```

This runs as a **weekly cron** in the API process (not the worker, since BookOwl has no worker mode). Schedule it in `internal/app/app.go` using a simple `time.Ticker`.

```go
// Query for orphans:
SELECT so.id, so.storage_key
FROM storage_objects so
LEFT JOIN document_images di ON di.storage_id = so.id
WHERE di.id IS NULL
  AND so.created_at < now() - $1::interval
LIMIT 100;
```

Batch size: 100 objects per run to avoid long-running transactions.

## 11. Configuration Reference

```bash
# Backend selection
BOOKOWL_STORAGE_BACKEND=local        # "local" or "s3"

# Local backend
BOOKOWL_STORAGE_LOCAL_PATH=/data/bookowl-images

# S3-compatible backend
BOOKOWL_STORAGE_S3_ENDPOINT=https://minio.example.com
BOOKOWL_STORAGE_S3_BUCKET=bookowl-images
BOOKOWL_STORAGE_S3_REGION=eu-central-1
BOOKOWL_STORAGE_S3_ACCESS_KEY_ID=CHANGEME
BOOKOWL_STORAGE_S3_SECRET_ACCESS_KEY=CHANGEME
BOOKOWL_STORAGE_S3_PUBLIC_URL=https://minio.example.com/bookowl-images
BOOKOWL_STORAGE_S3_USE_PRESIGN=false

# Limits
BOOKOWL_STORAGE_MAX_IMAGE_SIZE_MB=10   # default: 10
```

## 12. Helm Values (image storage)

```yaml
storage:
  backend: s3   # "local" or "s3"

  local:
    path: /data/bookowl-images
    # Requires a PVC:
    pvc:
      enabled: false
      storageClass: longhorn
      size: 10Gi

  s3:
    endpoint: https://minio.example.com
    bucket: bookowl-images
    region: eu-central-1
    publicUrl: https://minio.example.com/bookowl-images
    usePresign: false
    # accessKeyId and secretAccessKey set via external secret
```

## 13. MinIO Deployment (recommended for production)

Deploy MinIO in the `data` namespace alongside PostgreSQL and Redis:

```yaml
# Via Bitnami Helm chart
helm install minio oci://registry-1.docker.io/bitnamicharts/minio \
  --namespace data \
  --set auth.rootUser=admin \
  --set auth.rootPassword=CHANGEME \
  --set defaultBuckets=bookowl-images \
  --set persistence.size=50Gi \
  --set persistence.storageClass=longhorn \
  --set metrics.enabled=true \
  --set metrics.serviceMonitor.enabled=true
```

MinIO is not exposed via ingress — BookOwl accesses it via the internal service URL `http://minio.data.svc:9000`. The public URL for presigned redirects uses an internal URL too, since the browser in KRITIS environments has access to the cluster ingress but not to internal services — configure an ingress for MinIO if using presigned redirect mode.

## 14. Acceptance Criteria

- [ ] Images upload via drag-and-drop and paste into editor
- [ ] Uploaded images appear immediately in the document (optimistic UI)
- [ ] Images persist across page reload
- [ ] Deleting an image block calls DELETE and removes the file
- [ ] Image URLs are tenant-scoped (image from tenant A returns 404 when requested by tenant B)
- [ ] Local backend: images served directly from API
- [ ] S3 backend: GET /api/v1/images/:id returns a 302 redirect
- [ ] Files > 10MB return 413 with a clear error in the editor
- [ ] Non-image file types return 415
- [ ] Orphan cleanup job runs weekly, logs count of deleted objects
- [ ] `BOOKOWL_STORAGE_BACKEND` switch works without code changes
