-- name: CreateRole :one
INSERT INTO roles (
    organization_id,
    group_id, 
    name,
    is_admin,
    is_protected
    ) VALUES (
        $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE organization_id =$1 AND name = $2;

-- name: ListRoles :many
SELECT * FROM roles
WHERE organization_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListRolesByGroupID :many
SELECT * FROM roles WHERE group_id = $1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: UpdateRole :one
UPDATE roles 
SET organization_id = $2, group_id = $3, name = $4, is_admin = $5, is_protected = $6, updated_at = NOW() 
WHERE id = $1 RETURNING *;


-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;