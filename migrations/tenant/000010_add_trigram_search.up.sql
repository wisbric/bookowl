CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_documents_title_trgm ON documents USING GIN(title gin_trgm_ops);
CREATE INDEX idx_documents_content_text_trgm ON documents USING GIN(content_text gin_trgm_ops);
