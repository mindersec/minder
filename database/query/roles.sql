-- name: CreateRole :one
INSERT INTO roles (
    group_id, 
    name
    ) VALUES (
        $1, $2
) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles
WHERE group_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateRole :one
UPDATE roles 
SET group_id = $2, name = $3, updated_at = NOW() 
WHERE id = $1 RETURNING *;


-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;