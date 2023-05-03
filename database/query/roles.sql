-- name: CreateRole :one
INSERT INTO roles (organisation_id, name) VALUES ($1, $2) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles;

-- name: UpdateRole :one
UPDATE roles SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;