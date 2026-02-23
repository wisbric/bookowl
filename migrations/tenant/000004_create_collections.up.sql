CREATE TABLE collections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES collections(id),
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
