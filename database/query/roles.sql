-- name: CreateRole :one
INSERT INTO roles (
    organisation_id, 
    name
    ) VALUES (
        $1, $2
) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles
WHERE organisation_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateRole :one
UPDATE roles 
SET organisation_id = $2, name = $3, updated_at = NOW() 
WHERE id = $1 RETURNING *;


-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;