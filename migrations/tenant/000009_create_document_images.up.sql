CREATE TABLE document_images (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    storage_id      UUID NOT NULL REFERENCES storage_objects(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(document_id, storage_id)
);

CREATE INDEX idx_document_images_document ON document_images(document_id);
CREATE INDEX idx_document_images_storage ON document_images(storage_id);
