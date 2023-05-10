-- name: CreateUser :one
INSERT INTO users (organisation_id, group_id, email, password, name, avatar_url, provider, provider_id, is_admin, is_super_admin) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users;

-- name: UpdateUser :one
UPDATE users SET organisation_id = $2, group_id = $3, email = $4, password = $5, name = $6, avatar_url = $7, provider = $8, provider_id = $9, is_admin = $10, is_super_admin = $11, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;