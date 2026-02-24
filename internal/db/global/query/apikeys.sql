-- name: CreateAPIKey :one
INSERT INTO public.api_keys (tenant_id, key_hash, key_prefix, description, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreatePersonalToken :one
INSERT INTO public.api_keys (tenant_id, key_hash, key_prefix, description, role, user_id, is_personal, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, true, $7)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT
    ak.*,
    t.slug AS tenant_slug
FROM public.api_keys ak
JOIN public.tenants t ON t.id = ak.tenant_id
WHERE ak.key_hash = $1;

-- name: GetAPIKeyByID :one
SELECT * FROM public.api_keys
WHERE id = $1;

-- name: ListAPIKeysByTenant :many
SELECT * FROM public.api_keys
WHERE tenant_id = $1 AND is_personal = false
ORDER BY created_at DESC;

-- name: ListPersonalTokens :many
SELECT * FROM public.api_keys
WHERE tenant_id = $1 AND user_id = $2 AND is_personal = true
ORDER BY created_at DESC;

-- name: DeleteAPIKey :exec
DELETE FROM public.api_keys
WHERE id = $1 AND tenant_id = $2;

-- name: DeletePersonalToken :exec
DELETE FROM public.api_keys
WHERE id = $1 AND tenant_id = $2 AND user_id = $3 AND is_personal = true;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE public.api_keys
SET last_used_at = now()
WHERE id = $1;
