DROP INDEX IF EXISTS idx_api_keys_hash;
DROP INDEX IF EXISTS idx_api_keys_tenant;

ALTER TABLE public.api_keys
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS is_personal,
    DROP COLUMN IF EXISTS user_id;
