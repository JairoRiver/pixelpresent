-- name: CreateUser :one
INSERT INTO users (email)
VALUES ($1)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE
    (sqlc.narg('id')::uuid IS NOT NULL OR sqlc.narg('email')::citext IS NOT NULL) AND
    (sqlc.narg('id')::uuid IS NULL OR id = sqlc.narg('id')) AND
    (sqlc.narg('email')::citext IS NULL OR email = sqlc.narg('email'))
LIMIT 1;
