-- name: CreateTemplate :one
INSERT INTO templates (name, description, doc_type, content, icon, is_system, is_global, space_id, created_by, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetTemplateByID :one
SELECT * FROM templates WHERE id = $1;

-- name: ListTemplates :many
SELECT * FROM templates
WHERE (is_system = true OR is_global = true OR space_id = @space_id::uuid)
ORDER BY
    CASE WHEN is_system THEN 0 WHEN is_global THEN 1 ELSE 2 END,
    sort_order,
    name;

-- name: ListAllTemplates :many
SELECT * FROM templates
ORDER BY
    CASE WHEN is_system THEN 0 WHEN is_global THEN 1 ELSE 2 END,
    sort_order,
    name;

-- name: ListTemplatesByDocType :many
SELECT * FROM templates
WHERE doc_type = $1
  AND (is_system = true OR is_global = true OR space_id = @space_id::uuid)
ORDER BY
    CASE WHEN is_system THEN 0 WHEN is_global THEN 1 ELSE 2 END,
    sort_order,
    name;

-- name: UpdateTemplate :one
UPDATE templates
SET name = $2, description = $3, content = $4, icon = $5, sort_order = $6, updated_at = now()
WHERE id = $1 AND is_system = false
RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM templates WHERE id = $1 AND is_system = false;

-- name: GetSystemTemplateByDocType :one
SELECT * FROM templates
WHERE is_system = true AND doc_type = $1
ORDER BY sort_order
LIMIT 1;

-- name: GetSystemTemplateByName :one
SELECT * FROM templates
WHERE is_system = true AND name = $1
LIMIT 1;
