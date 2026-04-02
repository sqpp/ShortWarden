-- name: ListRecentClicks :many
SELECT
  ce.id,
  ce.clicked_at,
  ce.referrer,
  ce.user_agent,
  ce.ip,
  ce.country,
  ce.device,
  l.id AS link_id,
  l.alias,
  l.domain_id
FROM click_events ce
JOIN links l ON l.id = ce.link_id
WHERE l.user_id = $1
  AND l.deleted_at IS NULL
ORDER BY ce.clicked_at DESC
LIMIT $2;

-- name: ListTopLinksByClicks :many
SELECT
  l.*,
  COALESCE(count(ce.id), 0)::bigint AS clicks
FROM links l
LEFT JOIN click_events ce
  ON ce.link_id = l.id
 AND ce.clicked_at >= now() - ($3::int * interval '1 day')
WHERE l.user_id = $1
  AND l.deleted_at IS NULL
GROUP BY l.id
ORDER BY clicks DESC, l.created_at DESC
LIMIT $2;

