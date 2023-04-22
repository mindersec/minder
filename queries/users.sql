-- name: CreateUser :exec
INSERT INTO users (email, int_id, salted_password, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;
