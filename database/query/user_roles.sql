-- name: AssignRoleToUser :one
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) RETURNING *;

-- name: GetUserRoles :many
SELECT * FROM user_roles WHERE user_id = $1;

-- name: RevokeRoleFromUser :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;