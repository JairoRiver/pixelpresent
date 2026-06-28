-- name: CreateGift :one
INSERT INTO gifts (
    creator_id, title, message, pixel_art, reveal_type, reveal_config,
    view_token, recipient_email, scheduled_open_at, scheduled_send_at,
    single_open, expires_at
) VALUES (
    @creator_id, @title, @message, @pixel_art, @reveal_type, @reveal_config,
    @view_token, @recipient_email, @scheduled_open_at, @scheduled_send_at,
    @single_open, @expires_at
)
RETURNING *;

-- name: GetGiftByID :one
SELECT * FROM gifts
WHERE id = $1
LIMIT 1;

-- name: GetGiftByViewToken :one
SELECT * FROM gifts
WHERE view_token = $1
LIMIT 1;

-- name: GiftViewTokenExists :one
SELECT EXISTS (
    SELECT 1 FROM gifts WHERE view_token = $1
);

-- name: UpdateGift :one
-- Full-replace of the creator-editable fields (the editor holds the whole gift
-- state): passing NULL clears a nullable field. view_token, creator_id and
-- created_at are immutable; updated_at is bumped. Ownership is enforced in the
-- service, not here.
UPDATE gifts SET
    title             = @title,
    message           = @message,
    pixel_art         = @pixel_art,
    reveal_type       = @reveal_type,
    reveal_config     = @reveal_config,
    recipient_email   = @recipient_email,
    scheduled_open_at = @scheduled_open_at,
    scheduled_send_at = @scheduled_send_at,
    single_open       = @single_open,
    expires_at        = @expires_at,
    updated_at        = now()
WHERE id = @id
RETURNING *;

-- name: ListGiftsByUser :many
SELECT * FROM gifts
WHERE creator_id = $1
ORDER BY created_at DESC;
