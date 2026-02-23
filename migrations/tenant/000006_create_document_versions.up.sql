CREATE TABLE document_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    title           TEXT NOT NULL,
    content         JSONB NOT NULL,
    content_text    TEXT NOT NULL,
    doc_type        TEXT NOT NULL,
    tags            TEXT[] NOT NULL DEFAULT '{}',
    changed_by      UUID REFERENCES users(id),
    change_summary  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(document_id, version)
);

CREATE INDEX idx_document_versions_doc ON document_versions(document_id, version DESC);
