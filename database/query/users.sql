-- name: CreateUser :one
INSERT INTO users (organization_id, identity_subject) VALUES ($1, $2) RETURNING *;

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

-- name: ListUsersByProject :many
SELECT users.* FROM users
JOIN user_projects ON users.id = user_projects.user_id
WHERE user_projects.project_id = $1
ORDER BY users.id
LIMIT $2
OFFSET $3;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
