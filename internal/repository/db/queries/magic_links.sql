-- name: CreateMagicLink :one
INSERT INTO magic_links (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMagicLinkByTokenHash :one
SELECT * FROM magic_links
WHERE token_hash = $1
LIMIT 1;

-- name: MarkMagicLinkConsumed :one
UPDATE magic_links
SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL
RETURNING *;
