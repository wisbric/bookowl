CREATE TABLE IF NOT EXISTS templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    description     TEXT,
    doc_type        TEXT NOT NULL DEFAULT 'document',
    content         JSONB NOT NULL,
    icon            TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT false,
    is_global       BOOLEAN NOT NULL DEFAULT false,
    space_id        UUID REFERENCES spaces(id) ON DELETE CASCADE,
    created_by      UUID REFERENCES users(id),
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_templates_space ON templates(space_id);
CREATE INDEX IF NOT EXISTS idx_templates_global ON templates(is_global) WHERE is_global = true;
CREATE INDEX IF NOT EXISTS idx_templates_system ON templates(is_system) WHERE is_system = true;
