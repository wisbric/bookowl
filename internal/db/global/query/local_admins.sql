-- name: CreateLocalAdmin :one
INSERT INTO public.local_admins (tenant_slug, username, password_hash, must_change)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLocalAdmin :one
SELECT * FROM public.local_admins
WHERE tenant_slug = $1 AND username = $2;

-- name: GetLocalAdminByTenant :one
SELECT * FROM public.local_admins
WHERE tenant_slug = $1;

-- name: UpdateLocalAdminPassword :one
UPDATE public.local_admins
SET password_hash = $2, must_change = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateLocalAdminLastLogin :exec
UPDATE public.local_admins
SET last_login_at = now(), updated_at = now()
WHERE id = $1;
