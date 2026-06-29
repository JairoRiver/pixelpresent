-- name: CreateReaction :one
INSERT INTO reactions (
    gift_id, kind, emoji, message, voice_storage_key
) VALUES (
    @gift_id, @kind, @emoji, @message, @voice_storage_key
)
RETURNING *;

-- name: ListReactionsByGift :many
-- Chronological wall of reactions for the creator's view (oldest first).
-- Borrar las reacciones al borrar el regalo lo cubre la FK ON DELETE CASCADE;
-- el borrado de una reacción individual (moderación) queda pendiente de diseño.
SELECT * FROM reactions
WHERE gift_id = $1
ORDER BY created_at ASC;
