ALTER TABLE public.api_keys DROP COLUMN IF EXISTS last_used_at;
ALTER TABLE public.api_keys DROP COLUMN IF EXISTS expires_at;
ALTER TABLE public.api_keys DROP COLUMN IF EXISTS is_personal;
ALTER TABLE public.api_keys DROP COLUMN IF EXISTS user_id;

DROP TABLE IF EXISTS public.local_admins;
