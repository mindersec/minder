-- name: CreateUser :one
INSERT INTO users (organization_id, email, identity_subject, first_name, last_name) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserBySubject :one
SELECT * FROM users WHERE identity_subject = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListUsersByOrganization :many
SELECT * FROM users
WHERE organization_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListUsersByGroup :many
SELECT users.* FROM users
JOIN user_groups ON users.id = user_groups.user_id
WHERE user_groups.group_id = $1
ORDER BY users.id
LIMIT $2
OFFSET $3;

