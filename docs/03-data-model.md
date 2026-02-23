# BookOwl — Data Model Specification

## 1. Schema Strategy

Identical to NightOwl:
- `public` schema: global tables (tenants, api_keys)
- `tenant_<slug>` schema: all tenant-specific data
- Global migrations: `migrations/global/`
- Tenant migrations: `migrations/tenant/`

Tenant slugs are the same as NightOwl — a BookOwl tenant `acme` corresponds to NightOwl tenant `acme`. This is what enables service-to-service auth and shared identity.

## 2. Global Tables (public schema)

### tenants

```sql
CREATE TABLE public.tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL UNIQUE,   -- matches NightOwl tenant slug
    name        TEXT NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    -- config: nightowl_api_url, nightowl_api_key, default_space_id
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### api_keys

```sql
CREATE TABLE public.api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL UNIQUE,
    key_prefix  TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'editor',  -- admin, editor, viewer
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 3. Tenant Tables

### users

Migration: `000001_create_users`

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id     TEXT NOT NULL UNIQUE,  -- OIDC subject claim (same as NightOwl)
    email           TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'editor',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_email ON users(email);
```

### spaces

Migration: `000002_create_spaces`

```sql
CREATE TABLE spaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,        -- URL-safe slug, unique per tenant
    description TEXT,
    icon        TEXT,                 -- emoji or icon name
    is_private  BOOLEAN DEFAULT false,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(slug)
);
```

### space_members

Migration: `000003_create_space_members`

```sql
CREATE TABLE space_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    role        TEXT NOT NULL DEFAULT 'editor',  -- admin, editor, viewer
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(space_id, user_id)
);

CREATE INDEX idx_space_members_space ON space_members(space_id);
CREATE INDEX idx_space_members_user ON space_members(user_id);
```

### collections

Migration: `000004_create_collections`

```sql
CREATE TABLE collections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES collections(id),  -- NULL = top-level
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    icon        TEXT,
    position    INTEGER NOT NULL DEFAULT 0,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(space_id, slug)
);

CREATE INDEX idx_collections_space ON collections(space_id);
CREATE INDEX idx_collections_parent ON collections(parent_id);
```

### documents

Migration: `000005_create_documents`

```sql
CREATE TABLE documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id        UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    collection_id   UUID REFERENCES collections(id),
    title           TEXT NOT NULL,
    slug            TEXT NOT NULL,
    content         JSONB NOT NULL DEFAULT '{}',  -- ProseMirror/Tiptap JSON
    content_text    TEXT GENERATED ALWAYS AS (     -- plain text for FTS, extracted in Go
        -- NOTE: content_text is set by the application, not a generated column.
        -- See document service for extraction logic.
        ''
    ) STORED,
    doc_type        TEXT NOT NULL DEFAULT 'document',  -- document, runbook, post-mortem, sop, adr
    status          TEXT NOT NULL DEFAULT 'draft',     -- draft, published, archived
    tags            TEXT[] NOT NULL DEFAULT '{}',
    icon            TEXT,
    position        INTEGER NOT NULL DEFAULT 0,
    word_count      INTEGER NOT NULL DEFAULT 0,
    version         INTEGER NOT NULL DEFAULT 1,

    -- NightOwl integration
    nightowl_incident_id  TEXT,    -- NightOwl incident UUID (for post-mortems)
    nightowl_alert_id     TEXT,    -- NightOwl alert UUID (for incident-linked runbooks)

    search_vector   TSVECTOR,
    created_by      UUID REFERENCES users(id),
    updated_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(space_id, slug)
);

CREATE INDEX idx_documents_space ON documents(space_id);
CREATE INDEX idx_documents_collection ON documents(collection_id);
CREATE INDEX idx_documents_type ON documents(doc_type);
CREATE INDEX idx_documents_tags ON documents USING GIN(tags);
CREATE INDEX idx_documents_search ON documents USING GIN(search_vector);
CREATE INDEX idx_documents_nightowl_incident ON documents(nightowl_incident_id)
    WHERE nightowl_incident_id IS NOT NULL;
```

**Full-text search trigger:**

```sql
CREATE OR REPLACE FUNCTION update_document_search_vector() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.doc_type, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.content_text, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(array_to_string(NEW.tags, ' '), '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_documents_search_vector
    BEFORE INSERT OR UPDATE ON documents
    FOR EACH ROW EXECUTE FUNCTION update_document_search_vector();
```

**Note on `content_text`:** The document service extracts plain text from the Tiptap JSON (walking all text nodes) and sets `content_text` before INSERT/UPDATE. This plain text is what feeds the FTS trigger. The `GENERATED ALWAYS` syntax above is a placeholder comment — in practice the application sets it.

### document_versions

Migration: `000006_create_document_versions`

```sql
CREATE TABLE document_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    title           TEXT NOT NULL,
    content         JSONB NOT NULL,   -- full snapshot of the document content at this version
    content_text    TEXT NOT NULL,    -- plain text snapshot for diff display
    doc_type        TEXT NOT NULL,
    tags            TEXT[] NOT NULL DEFAULT '{}',
    changed_by      UUID REFERENCES users(id),
    change_summary  TEXT,             -- optional: "Updated step 3 prerequisites"
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(document_id, version)
);

CREATE INDEX idx_document_versions_doc ON document_versions(document_id, version DESC);
```

A new version is saved whenever the document is updated. The version number increments from `documents.version`. The current document content lives in `documents.content`; all historical snapshots are in `document_versions`.

### audit_log

Migration: `000007_create_audit_log`

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id),
    api_key_id  UUID,
    action      TEXT NOT NULL,     -- created, updated, deleted, restored, published
    resource    TEXT NOT NULL,     -- document, space, collection
    resource_id UUID,
    detail      JSONB DEFAULT '{}',
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_resource ON audit_log(resource, resource_id);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at DESC);
```

## 4. Key Queries

### Full-text document search

```sql
SELECT
    d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
    s.name AS space_name, s.slug AS space_slug,
    ts_rank(d.search_vector, query) AS rank,
    ts_headline('english', d.title, query) AS title_highlight,
    ts_headline('english', d.content_text, query,
        'StartSel=<mark>, StopSel=</mark>, MaxWords=30, MinWords=15') AS content_highlight
FROM documents d
JOIN spaces s ON s.id = d.space_id
CROSS JOIN plainto_tsquery('english', $1) query
WHERE d.search_vector @@ query
  AND d.status != 'archived'
ORDER BY rank DESC
LIMIT $2 OFFSET $3;
```

### Space tree (for sidebar navigation)

```sql
SELECT
    c.id AS collection_id, c.name AS collection_name, c.slug AS collection_slug,
    c.parent_id, c.position AS collection_position, c.icon AS collection_icon,
    d.id AS document_id, d.title AS document_title, d.slug AS document_slug,
    d.doc_type, d.status, d.position AS document_position, d.icon AS document_icon
FROM collections c
LEFT JOIN documents d ON d.collection_id = c.id AND d.status != 'archived'
WHERE c.space_id = $1
ORDER BY c.position ASC, d.position ASC;
```

### Runbooks for NightOwl integration

```sql
SELECT id, title, slug, content_text, tags, updated_at
FROM documents
WHERE doc_type = 'runbook'
  AND status = 'published'
ORDER BY updated_at DESC;
```

## 5. Content Storage

Document content is stored as **Tiptap/ProseMirror JSON** in the `content` JSONB column. Example structure:

```json
{
  "type": "doc",
  "content": [
    { "type": "heading", "attrs": { "level": 1 }, "content": [{ "type": "text", "text": "Pod CrashLoopBackOff Runbook" }] },
    { "type": "callout", "attrs": { "type": "warning" }, "content": [{ "type": "text", "text": "Check logs first before restarting" }] },
    {
      "type": "liveContextBlock",
      "attrs": {
        "subtype": "oncall",
        "rosterId": "uuid-of-de-on-call-roster",
        "label": "Current DE On-Call"
      }
    },
    { "type": "heading", "attrs": { "level": 2 }, "content": [{ "type": "text", "text": "Diagnosis Steps" }] },
    { "type": "taskList", "content": [
      { "type": "taskItem", "attrs": { "checked": false }, "content": [{ "type": "text", "text": "Check pod events: kubectl describe pod -n <namespace>" }] },
      { "type": "taskItem", "attrs": { "checked": false }, "content": [{ "type": "text", "text": "Check logs: kubectl logs <pod> --previous" }] }
    ]}
  ]
}
```

The application walks this JSON to extract `content_text` for FTS. The frontend editor (Tiptap) reads and writes this format directly.

## 6. Migration History

| # | Name | Description |
|---|------|-------------|
| Global 001 | `create_tenants` | Tenants with slug, config JSONB |
| Global 002 | `create_api_keys` | API keys with role |
| Tenant 001 | `create_users` | Users with OIDC external_id |
| Tenant 002 | `create_spaces` | Spaces with slug, privacy |
| Tenant 003 | `create_space_members` | Space membership with role |
| Tenant 004 | `create_collections` | Collections with parent (one level nesting) |
| Tenant 005 | `create_documents` | Documents with FTS trigger |
| Tenant 006 | `create_document_versions` | Version snapshots |
| Tenant 007 | `create_audit_log` | Audit trail |
