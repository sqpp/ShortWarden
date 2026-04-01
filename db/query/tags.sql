-- name: CreateUserTag :one
INSERT INTO user_tags (user_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: ListUserTags :many
SELECT *
FROM user_tags
WHERE user_id = $1
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: DeleteUserTag :exec
DELETE FROM user_tags
WHERE user_id = $1 AND name = $2;

-- name: TagCountsFromLinks :many
SELECT
  t.tag::text AS name,
  count(*)::bigint AS link_count
FROM (
  SELECT unnest(tags) AS tag
  FROM links
  WHERE user_id = $1 AND deleted_at IS NULL
) t
GROUP BY 1
ORDER BY 2 DESC, 1 ASC
LIMIT $2 OFFSET $3;

