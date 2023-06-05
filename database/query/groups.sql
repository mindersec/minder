-- name: CreateGroup :one
INSERT INTO groups (
    organization_id,
    name,
    description,
    is_protected
    ) VALUES (
        $1, $2, $3, $4
) RETURNING *;

-- name: GetGroupByID :one
SELECT * FROM groups WHERE id = $1;

-- name: GetGroupByName :one
SELECT * FROM groups WHERE name = $1;

-- name: ListGroups :many
SELECT * FROM groups
WHERE organization_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: ListGroupsByOrganizationID :many
SELECT * FROM groups WHERE organization_id = $1;

-- name: UpdateGroup :one
UPDATE groups 
SET organization_id = $2, name = $3, description = $4, is_protected = $5, updated_at = NOW() 
WHERE id = $1 RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = $1;