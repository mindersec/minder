-- name: CreateUser :one
INSERT INTO users (organisation_id, group_id, role_id, email, username, password, first_name, last_name, is_protected) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByUserName :one
SELECT * FROM users WHERE username = $1;

-- name: ListUsers :many
SELECT * FROM users;

-- name: UpdateUser :one
UPDATE users SET organisation_id = $2, group_id = $3, role_id = $4, email = $5, username = $6, password = $7, first_name = $8, last_name = $9, is_protected = $10, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;