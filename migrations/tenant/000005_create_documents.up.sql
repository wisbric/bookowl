CREATE TABLE documents (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id              UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    collection_id         UUID REFERENCES collections(id),
    title                 TEXT NOT NULL,
    slug                  TEXT NOT NULL,
    content               JSONB NOT NULL DEFAULT '{}',
    content_text          TEXT NOT NULL DEFAULT '',
    doc_type              TEXT NOT NULL DEFAULT 'document',
    status                TEXT NOT NULL DEFAULT 'draft',
    tags                  TEXT[] NOT NULL DEFAULT '{}',
    icon                  TEXT,
    position              INTEGER NOT NULL DEFAULT 0,
    word_count            INTEGER NOT NULL DEFAULT 0,
    version               INTEGER NOT NULL DEFAULT 1,
    nightowl_incident_id  TEXT,
    nightowl_alert_id     TEXT,
    search_vector         TSVECTOR,
    created_by            UUID REFERENCES users(id),
    updated_by            UUID REFERENCES users(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(space_id, slug)
);

CREATE INDEX idx_documents_space ON documents(space_id);
CREATE INDEX idx_documents_collection ON documents(collection_id);
CREATE INDEX idx_documents_type ON documents(doc_type);
CREATE INDEX idx_documents_tags ON documents USING GIN(tags);
CREATE INDEX idx_documents_search ON documents USING GIN(search_vector);
CREATE INDEX idx_documents_nightowl_incident ON documents(nightowl_incident_id)
    WHERE nightowl_incident_id IS NOT NULL;

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
