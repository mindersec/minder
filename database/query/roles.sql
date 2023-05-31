-- name: CreateRole :one
INSERT INTO roles (
    group_id, 
    name,
    is_admin,
    is_protected
    ) VALUES (
        $1, $2, $3, $4
) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles
WHERE group_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListRolesByGroupID :many
SELECT * FROM roles WHERE group_id = $1;

-- name: UpdateRole :one
UPDATE roles 
SET group_id = $2, name = $3, is_admin = $4, is_protected = $5, updated_at = NOW() 
WHERE id = $1 RETURNING *;


-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;