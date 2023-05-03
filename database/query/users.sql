-- name: CreateUser :one
INSERT INTO users (organisation_id, group_id, email, password, first_name, last_name, is_admin, is_super_admin) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users;

-- name: UpdateUser :one
UPDATE users SET organisation_id = $2, group_id = $3, email = $4, password = $5, first_name = $6, last_name = $7, is_admin = $8, is_super_admin = $9, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;