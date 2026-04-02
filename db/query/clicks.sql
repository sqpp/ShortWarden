-- name: InsertClickEvent :one
INSERT INTO click_events (link_id, clicked_at, referrer, user_agent, ip, country, device)
VALUES ($1, now(), $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListClickEventsForLink :many
SELECT *
FROM click_events
WHERE link_id = $1
ORDER BY clicked_at DESC
LIMIT $2 OFFSET $3;

-- name: CountClickEventsForLink :one
SELECT count(*)::bigint AS click_count
FROM click_events
WHERE link_id = $1;

-- name: ClickTotalsByDay :many
SELECT
  (date_trunc('day', clicked_at))::timestamptz AS day,
  count(*)::bigint AS clicks
FROM click_events
WHERE link_id = $1
  AND clicked_at >= $2
  AND clicked_at < $3
GROUP BY 1
ORDER BY 1 ASC;

