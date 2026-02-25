CREATE TABLE document_yjs_state (
    document_id   UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    yjs_state     BYTEA NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
