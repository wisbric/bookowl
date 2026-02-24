-- name: SearchDocuments :many
WITH fts AS (
    SELECT
        d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
        s.name AS space_name, s.slug AS space_slug,
        ts_rank(d.search_vector, plainto_tsquery('english', @query::text)) AS rank,
        ts_headline('english', d.title, plainto_tsquery('english', @query::text)) AS title_highlight,
        ts_headline('english', d.content_text, plainto_tsquery('english', @query::text),
            'StartSel=<mark>, StopSel=</mark>, MaxWords=30, MinWords=15') AS content_highlight
    FROM documents d
    JOIN spaces s ON s.id = d.space_id
    WHERE d.search_vector @@ plainto_tsquery('english', @query::text)
      AND d.status != 'archived'
    ORDER BY rank DESC
    LIMIT @result_limit OFFSET @result_offset
),
fuzzy AS (
    SELECT
        d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
        s.name AS space_name, s.slug AS space_slug,
        GREATEST(
            similarity(d.title, @query::text),
            similarity(d.content_text, @query::text) * 0.5
        )::float4 AS rank,
        d.title AS title_highlight,
        '' AS content_highlight
    FROM documents d
    JOIN spaces s ON s.id = d.space_id
    WHERE d.status != 'archived'
      AND (d.title % @query::text OR d.content_text % @query::text)
      AND NOT EXISTS (SELECT 1 FROM fts WHERE fts.id = d.id)
    ORDER BY rank DESC
    LIMIT @result_limit OFFSET @result_offset
)
SELECT * FROM fts
UNION ALL
SELECT * FROM fuzzy;

-- name: SearchDocumentsBySpace :many
WITH fts AS (
    SELECT
        d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
        s.name AS space_name, s.slug AS space_slug,
        ts_rank(d.search_vector, plainto_tsquery('english', @query::text)) AS rank,
        ts_headline('english', d.title, plainto_tsquery('english', @query::text)) AS title_highlight,
        ts_headline('english', d.content_text, plainto_tsquery('english', @query::text),
            'StartSel=<mark>, StopSel=</mark>, MaxWords=30, MinWords=15') AS content_highlight
    FROM documents d
    JOIN spaces s ON s.id = d.space_id
    WHERE d.search_vector @@ plainto_tsquery('english', @query::text)
      AND d.status != 'archived'
      AND d.space_id = @space_id
    ORDER BY rank DESC
    LIMIT @result_limit OFFSET @result_offset
),
fuzzy AS (
    SELECT
        d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
        s.name AS space_name, s.slug AS space_slug,
        GREATEST(
            similarity(d.title, @query::text),
            similarity(d.content_text, @query::text) * 0.5
        )::float4 AS rank,
        d.title AS title_highlight,
        '' AS content_highlight
    FROM documents d
    JOIN spaces s ON s.id = d.space_id
    WHERE d.status != 'archived'
      AND d.space_id = @space_id
      AND (d.title % @query::text OR d.content_text % @query::text)
      AND NOT EXISTS (SELECT 1 FROM fts WHERE fts.id = d.id)
    ORDER BY rank DESC
    LIMIT @result_limit OFFSET @result_offset
)
SELECT * FROM fts
UNION ALL
SELECT * FROM fuzzy;

-- name: SearchDocumentsByType :many
WITH fts AS (
    SELECT
        d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
        s.name AS space_name, s.slug AS space_slug,
        ts_rank(d.search_vector, plainto_tsquery('english', @query::text)) AS rank,
        ts_headline('english', d.title, plainto_tsquery('english', @query::text)) AS title_highlight,
        ts_headline('english', d.content_text, plainto_tsquery('english', @query::text),
            'StartSel=<mark>, StopSel=</mark>, MaxWords=30, MinWords=15') AS content_highlight
    FROM documents d
    JOIN spaces s ON s.id = d.space_id
    WHERE d.search_vector @@ plainto_tsquery('english', @query::text)
      AND d.status != 'archived'
      AND d.doc_type = @doc_type
    ORDER BY rank DESC
    LIMIT @result_limit OFFSET @result_offset
),
fuzzy AS (
    SELECT
        d.id, d.title, d.doc_type, d.slug, d.space_id, d.collection_id,
        s.name AS space_name, s.slug AS space_slug,
        GREATEST(
            similarity(d.title, @query::text),
            similarity(d.content_text, @query::text) * 0.5
        )::float4 AS rank,
        d.title AS title_highlight,
        '' AS content_highlight
    FROM documents d
    JOIN spaces s ON s.id = d.space_id
    WHERE d.status != 'archived'
      AND d.doc_type = @doc_type
      AND (d.title % @query::text OR d.content_text % @query::text)
      AND NOT EXISTS (SELECT 1 FROM fts WHERE fts.id = d.id)
    ORDER BY rank DESC
    LIMIT @result_limit OFFSET @result_offset
)
SELECT * FROM fts
UNION ALL
SELECT * FROM fuzzy;

-- name: GetSpaceTree :many
SELECT
    c.id AS collection_id, c.name AS collection_name, c.slug AS collection_slug,
    c.parent_id, c.position AS collection_position, c.icon AS collection_icon,
    d.id AS document_id, d.title AS document_title, d.slug AS document_slug,
    d.doc_type, d.status, d.position AS document_position, d.icon AS document_icon
FROM collections c
LEFT JOIN documents d ON d.collection_id = c.id AND d.status != 'archived'
WHERE c.space_id = $1
ORDER BY c.position ASC, d.position ASC;

-- name: GetUncollectedDocuments :many
SELECT id, title, slug, doc_type, status, position, icon
FROM documents
WHERE space_id = $1
  AND collection_id IS NULL
  AND status != 'archived'
ORDER BY position ASC, title ASC;
