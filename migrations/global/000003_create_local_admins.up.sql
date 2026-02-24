CREATE TABLE public.local_admins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_slug     TEXT NOT NULL UNIQUE,
    username        TEXT NOT NULL DEFAULT 'admin',
    password_hash   TEXT NOT NULL,
    must_change     BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.api_keys ADD COLUMN user_id UUID;
ALTER TABLE public.api_keys ADD COLUMN is_personal BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.api_keys ADD COLUMN expires_at TIMESTAMPTZ;
ALTER TABLE public.api_keys ADD COLUMN last_used_at TIMESTAMPTZ;
