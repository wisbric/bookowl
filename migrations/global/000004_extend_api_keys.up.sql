ALTER TABLE public.api_keys
    ADD COLUMN IF NOT EXISTS user_id      UUID,
    ADD COLUMN IF NOT EXISTS is_personal  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS expires_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON public.api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash   ON public.api_keys(key_hash);
