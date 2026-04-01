-- name: CreateLink :one
INSERT INTO links (user_id, domain_id, alias, target_url, title, tags, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLinkByID :one
SELECT *
FROM links
WHERE id = $1 AND user_id = $2;

-- name: ListLinks :many
SELECT *
FROM links
WHERE user_id = $1
  AND (deleted_at IS NULL OR $3::boolean = true)
  AND ($5::text = '' OR $5::text = ANY(tags))
ORDER BY created_at DESC
LIMIT $2 OFFSET $4;

-- name: UpdateLink :one
UPDATE links
SET domain_id = $3,
    alias = $4,
    target_url = $5,
    title = $6,
    tags = $7,
    expires_at = $8,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SoftDeleteLink :one
UPDATE links
SET deleted_at = now(),
    updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetActiveLinkByAlias :one
SELECT *
FROM links
WHERE alias = $1
  AND deleted_at IS NULL
  AND (expires_at IS NULL OR expires_at > now());

