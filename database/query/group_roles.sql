-- name: AddRoleToGroup :one
INSERT INTO group_roles (group_id, role_id) VALUES ($1, $2) RETURNING *;

-- name: GetGroupRoles :many
SELECT * FROM group_roles WHERE group_id = $1;

-- name: RemoveRoleFromGroup :exec
DELETE FROM group_roles WHERE group_id = $1 AND role_id = $2;
