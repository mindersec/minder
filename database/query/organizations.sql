-- name: CreateOrganization :one
INSERT INTO organizations (
    name,
    company
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetOrganization :one 
SELECT * FROM organizations 
WHERE id = $1 LIMIT 1;

-- name: GetOrganizationByName :one 
SELECT * FROM organizations 
WHERE name = $1 LIMIT 1;


-- name: GetOrganizationForUpdate :one
SELECT * FROM organizations
WHERE name = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = $2, company = $3, updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;