CREATE TABLE public.api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL UNIQUE,
    key_prefix  TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'editor',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
